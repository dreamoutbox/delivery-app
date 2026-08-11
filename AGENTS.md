# Project: delivery-app

## Overview
Multi-restaurant food delivery app built with Go + HTMX + Bootstrap 5.

## Tech Stack
- **Go** — backend language
- **Gin** — HTTP framework
- **GORM** — ORM with AutoMigrate (no migration files)
- **PostgreSQL 18** — database (Docker Compose)
- **HTMX** — frontend interactivity (SSR partials via `HX-Request` header check)
- **Go `html/template`** — server-side templates
- **Bootstrap 5** — CSS framework (CDN)
- **gorilla/sessions + pgstore** — session-based auth (PostgreSQL session store)
- **bcrypt (cost 12)** — password hashing
- **aws-sdk-go-v2** — MinIO/S3 image upload
- **MinIO** — object storage (Docker Compose); two buckets: `delivery-public`, `delivery-private`
- **godotenv** — `.env` config loading
- **testify** — test assertions

## Key Conventions
- Money stored as **integer satang** (1 THB = 100 satang); displayed as `฿X.XX` via template helper.
- HTMX: handlers check `c.GetHeader("HX-Request") == "true"` — return HTML partial or full page accordingly.
- Roles: `admin`, `staff`, `restaurant`, `rider`, `customer` — stored in `users.role`.
- Restaurant ownership: `restaurants.owner_id` FK → `users.id`.
- Order states: `pending → confirmed → ready_for_pickup → in_delivery → delivered / cancelled`.
- Basket stored in server-side session (single restaurant; modal on switch).
- Customer has one address; snapshot copied to order at checkout.
- Delivery fee = Haversine distance × per-restaurant rate (stored in satang).
- Rider proof photos (pickup + delivery) go to `delivery-private` bucket; served via Go proxy handler with role check.
- Public food/restaurant images go to `delivery-public` bucket; served via MinIO public URL directly.
- Tests: each feature has a `*_test.go` using `testing` + `httptest` + testify.

## Project Layout (short)
```
cmd/server/      # main entry point
cmd/seed/        # seed CLI
internal/
  config/        # env loading
  database/      # GORM connection + AutoMigrate
  middleware/    # RequireLogin, RequireRole
  models/        # GORM models
  handlers/      # Gin handlers by domain
  services/      # business logic
  repository/    # DB queries
  storage/       # MinIO helpers
pkg/geo/         # Haversine distance
static/          # CSS/JS assets
templates/       # html/template files
```

## Rules for AI Agent

### After completing any feature or change, update the Memory section below:
1. Add a dated entry describing **what was implemented/changed** (one or two lines).
2. Mark completed phases in `plan.md` by changing `- [ ]` to `- [x]`.
3. Note any deviation from the plan and why.
4. Keep the Memory section concise — bullet points, no prose.

---

## Memory

<!-- AI Agent: append progress entries below after each feature/change -->

- **2026-08-11** — Project initialized. `plan.md` written. Tech stack finalized via grill session. No code written yet.
- **2026-08-11** — Completed Phase 1 (Foundation): Created config, GORM models (`User`, `Restaurant`, `MenuItem`, `Order`, `OrderItem`), DB connection & AutoMigrate, MinIO S3 SDK helper, Haversine formula + tests in `pkg/geo`, base templates, seed CLI (`cmd/seed`), server entry point (`cmd/server/main.go`).
- **2026-08-11** — Completed Phase 2 (Authentication): Implemented session helper (`internal/session`), `RequireLogin` & `RequireRole` middlewares, login/register/logout handlers and templates (`templates/auth`), error page template, and auth tests.
- **2026-08-11** — Completed Phase 3 (User Profile & Address): Created `internal/handlers/user.go`, `templates/profile.html` with address and lat/lng coordinates inputs/presets, profile routes under `RequireLogin`, and user profile test (`user_test.go`).
- **2026-08-11** — Completed Phase 4 (Restaurant & Menu): Created `internal/handlers/restaurant.go`, `templates/restaurant/dashboard.html` with restaurant profile edit & MinIO public image upload, menu items CRUD & MinIO upload, `/restaurant` routes under `RequireRole(restaurant, admin)`, and unit test (`restaurant_test.go`).
- **2026-08-11** — Completed Phase 5 & 6 (Customer Browse, Basket & Checkout): Created `internal/session/basket.go` session storage, `internal/handlers/customer.go`, customer templates (`restaurant_detail`, `cart`, `switch_modal`, `orders`, `order_detail`), Haversine distance-based delivery fee calculation at checkout, order placement, and unit tests (`customer_test.go`).
- **2026-08-11** — Completed Phases 7-10 (Restaurant Order Management, Rider Flow, Admin Panel, Private Proof Proxy): Added order status workflow in restaurant dashboard, rider dashboard with Google Maps navigation & MinIO private proof upload for pickup & delivery, admin dashboard stats & user role management, private proof photo proxy handler (`/internal/proof/:id/:type`), and unit tests (`rider_test.go`, `admin_test.go`).
