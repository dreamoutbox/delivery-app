# Postmortem: SeaweedFS Volume Slot Exhaustion on Multi-Bucket Proof Uploads

**Incident Date:** 2026-10-26  
**Status:** Resolved  
**Severity:** High  
**Impact:** Riders were unable to submit pickup or delivery proof photos on live deployments, causing order completion failure and blocking the core delivery lifecycle.

---

## Executive Summary

During end-to-end validation against the VPS deployment (`152.42.181.182:8080`), rider photo proof uploads consistently failed with the error page:
`Upload Failed: Failed to upload proof photo.`

Investigation revealed that SeaweedFS defaults to `-volume.max=8` volume slots per volume node. When initial restaurant and menu images were seeded into the `delivery-public` S3 bucket, SeaweedFS allocated 7 volumes for `delivery-public` (plus 1 default volume), completely filling all 8 available volume slots (`Max: 8, Free: 0`). When the rider subsequently attempted to upload private proof photos to the `delivery-private` bucket, SeaweedFS failed to allocate a writable volume for the `delivery-private` collection, returning an S3 `InternalError` (HTTP 500) to the application.

The issue was resolved by configuring `-volume.max=100` on the SeaweedFS server across both the VPS systemd service, the bare metal deployment script, and the Docker Compose definition.

---

## Symptoms

- When submitting the form `POST /rider/orders/:id/start` (Take Pickup Proof Photo), the user was redirected to the error template with:
  ```
  Upload Failed
  Failed to upload proof photo.
  ```
- Application logs (`journalctl -u delivery-app`) showed S3 `PutObject` failures to bucket `delivery-private`.
- Playwright E2E suite failed at `await expect(page.getByText('Pickup Proof Uploaded')).toBeVisible()`.

---

## Root Cause Analysis

### 1. SeaweedFS Bucket-to-Collection Mapping
SeaweedFS implements S3 buckets by mapping each bucket to an internal collection. In this project:
- Public images (food, restaurant banners) reside in bucket `delivery-public` -> collection `delivery-public`.
- Rider proof images reside in bucket `delivery-private` -> collection `delivery-private`.

### 2. Default Volume Slot Limit
By default, `weed server` runs with `-volume.max=8`, meaning the volume server is capped at 8 data volume files (`.dat` / `.idx`).

When seeding the database with restaurant and menu images, the volume server allocated 7 volumes dedicated to collection `delivery-public` plus 1 default volume:
```json
{
  "Max": 8,
  "Free": 0,
  "Layouts": [
    { "collection": "", "writables": [8] },
    { "collection": "delivery-public", "writables": [1, 2, 3, 4, 5, 6, 7] }
  ]
}
```

### 3. Collection Starvation
Because all 8 volume slots were consumed by `delivery-public`, no free slots remained for any other collection. When the rider submitted a proof photo:
1. `delivery-app` called AWS S3 SDK `PutObject` targeting bucket `delivery-private`.
2. SeaweedFS S3 layer requested a writable volume from SeaweedFS master for collection `delivery-private`.
3. SeaweedFS master reported:
   ```
   rpc error: code = Unknown desc = failed to find writable volumes for collection:delivery-private ... No writable volumes and no free volumes left
   ```
4. SeaweedFS S3 returned HTTP 500 `InternalError`.
5. `delivery-app` aborted the upload and returned HTTP 500 to the browser.

---

## Timeline

- **2026-10-26 07:15** — Playwright E2E test against VPS failed on rider pickup proof upload step.
- **2026-10-26 07:20** — SSH connection established to VPS; inspected `journalctl -u delivery-app` and confirmed S3 upload failure on `delivery-private`.
- **2026-10-26 07:25** — Executed test upload via Python `boto3` script directly against S3 endpoint (`http://127.0.0.1:9000`); verified `delivery-public` succeeded but `delivery-private` failed with HTTP 500.
- **2026-10-26 07:28** — Queried SeaweedFS master directory status (`curl http://127.0.0.1:9333/dir/status`), identifying volume limit exhaustion (`Max: 8, Free: 0`).
- **2026-10-26 07:32** — Updated `/etc/systemd/system/seaweedfs.service` on VPS to include `-volume.max=100`; executed `systemctl daemon-reload && systemctl restart seaweedfs`.
- **2026-10-26 07:35** — Verified SeaweedFS status reported `Max: 100, Free: 92`. Re-ran S3 test upload; `delivery-private` succeeded and allocated new volumes (IDs 9-15).
- **2026-10-26 07:44** — Truncated leftover test orders in PostgreSQL on VPS.
- **2026-10-26 07:45** — Re-ran full Playwright E2E test suite against VPS (`http://152.42.181.182:8080/`); all tests passed.
- **2026-10-26 07:46** — Updated `docker-compose.yml`, `demo-launcher-entrypoint.sh`, and `AGENTS.md` to persist `-volume.max=100`.

---

## Corrective and Preventative Actions

| Action | Component | Status |
|---|---|---|
| Configure `-volume.max=100` on live VPS SeaweedFS service | VPS `/etc/systemd/system/seaweedfs.service` | Completed |
| Update deployment launcher script to write `-volume.max=100` into SeaweedFS service | `demo-launcher-entrypoint.sh` | Completed |
| Update local development Docker Compose to run `weed server` with `-volume.max=100` | `docker-compose.yml` | Completed |
| Add automated headless E2E testing script with base URL override support | `dev-test-playwright.sh` | Completed |
| Document SeaweedFS multi-collection volume behavior | `AGENTS.md` | Completed |

---

## Lessons Learned

1. **Multi-Collection Volume Allocation in SeaweedFS:**
   Unlike traditional monolithic object stores, SeaweedFS partitions storage by collection. Each active collection requires its own volume allocation. When running multiple buckets, volume slots cannot be shared across collections.
2. **Default Limits:**
   Default service options suitable for minimal single-node setups (`-volume.max=8`) must be tuned explicitly when migrating between storage providers (e.g. MinIO to SeaweedFS).
3. **E2E Integration Testing:**
   Automated end-to-end testing with realistic file payloads and multi-role workflows is critical for catching storage quota and permission failures prior to release.
