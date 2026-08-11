# Food Delivery App — Implementation Plan

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go |
| HTTP Framework | Gin |
| ORM | GORM |
| Database | PostgreSQL 18 (Docker Compose) |
| Frontend | Go `html/template` + HTMX + Bootstrap 5 (CDN) |
| Storage | MinIO (Docker Compose) via AWS SDK for Go v2 (`aws-sdk-go-v2`) |
| Auth | Session-based (gorilla/sessions + PostgreSQL session store) |
| Password hashing | bcrypt |
| Config | `.env` + godotenv |
| Testing | Go `testing` + `net/http/httptest` + testify |

---

## Project Layout

```
delivery-app/
├── cmd/
│   ├── server/          # main entry point
│   │   └── main.go
│   └── seed/            # seeder CLI
│       └── main.go
├── internal/
│   ├── config/          # env loading
│   ├── database/        # GORM connection + AutoMigrate
│   ├── middleware/       # auth, role check, CSRF
│   ├── models/          # GORM models
│   ├── handlers/        # Gin handlers grouped by domain
│   ├── services/        # business logic
│   ├── repository/      # DB queries
│   └── storage/         # MinIO/S3 helpers
├── pkg/
│   └── geo/             # Haversine distance calculation
├── static/              # CSS, JS assets
├── templates/           # Go html/template files
│   ├── layout/
│   ├── auth/
│   ├── admin/
│   ├── restaurant/
│   ├── rider/
│   └── customer/
├── .env
├── .env.example
├── docker-compose.yml
├── go.mod
└── go.sum
```

---

## Database Models

### `users`
| Column | Type | Notes |
|---|---|---|
| id | serial | PK |
| name | varchar | |
| email | varchar | unique |
| password_hash | varchar | bcrypt |
| role | varchar | admin / staff / restaurant / rider / customer |
| phone | varchar | nullable |
| address | varchar | nullable (customer delivery address) |
| lat | float8 | nullable |
| lng | float8 | nullable |
| created_at | timestamptz | |
| updated_at | timestamptz | |

### `restaurants`
| Column | Type | Notes |
|---|---|---|
| id | serial | PK |
| owner_id | FK → users.id | restaurant user who owns this |
| name | varchar | |
| description | text | nullable |
| address | varchar | |
| lat | float8 | restaurant location |
| lng | float8 | |
| image_path | varchar | MinIO public bucket path |
| delivery_base_fee_satang | int | base fee in satang |
| delivery_per_km_satang | int | per-km rate in satang |
| is_open | bool | |
| created_at | timestamptz | |
| updated_at | timestamptz | |

### `menu_items`
| Column | Type | Notes |
|---|---|---|
| id | serial | PK |
| restaurant_id | FK → restaurants.id | |
| name | varchar | |
| description | text | nullable |
| price_satang | int | price in satang |
| image_path | varchar | MinIO public bucket path |
| is_available | bool | |
| created_at | timestamptz | |
| updated_at | timestamptz | |

### `orders`
| Column | Type | Notes |
|---|---|---|
| id | serial | PK |
| customer_id | FK → users.id | |
| restaurant_id | FK → restaurants.id | |
| rider_id | FK → users.id | nullable, set when rider accepts |
| status | varchar | pending / confirmed / ready_for_pickup / in_delivery / delivered / cancelled |
| delivery_address | varchar | snapshot at checkout |
| delivery_lat | float8 | snapshot at checkout |
| delivery_lng | float8 | snapshot at checkout |
| subtotal_satang | int | sum of items |
| delivery_fee_satang | int | computed at checkout (Haversine) |
| total_satang | int | subtotal + delivery_fee |
| rider_pickup_proof | varchar | MinIO private bucket path |
| rider_delivery_proof | varchar | MinIO private bucket path |
| note | text | customer note |
| created_at | timestamptz | |
| updated_at | timestamptz | |

### `order_items`
| Column | Type | Notes |
|---|---|---|
| id | serial | PK |
| order_id | FK → orders.id | |
| menu_item_id | FK → menu_items.id | |
| name | varchar | snapshot of item name |
| price_satang | int | snapshot of item price |
| quantity | int | |

### `sessions`
Managed by the gorilla/sessions pgstore library.

---

## MinIO Buckets

| Bucket | Access | Contents |
|---|---|---|
| `delivery-public` | Public read | Restaurant images, menu item images |
| `delivery-private` | Private | Rider pickup proof photos, delivery proof photos |

Private proof photos are served via `/internal/proof/{order_id}/{type}` Go handler that validates the requester role before proxying from MinIO.

---

## Money / Pricing

- All monetary values stored as **integer satang** (1 THB = 100 satang).
- Display: divide by 100, format as `฿X.XX` via a template helper.
- Delivery fee formula:
  ```
  distance_km = haversine(restaurant_lat, restaurant_lng, customer_lat, customer_lng)
  delivery_fee = delivery_base_fee + (distance_km × delivery_per_km_rate)
  ```
  Both rates stored in satang; computed in `pkg/geo`.

---

## Authentication & Sessions

- Sessions via `gorilla/sessions` with PostgreSQL-backed `pgstore`.
- Session holds: `user_id`, `role`.
- **Middleware `RequireLogin`** — redirects to `/login` if no valid session.
- **Middleware `RequireRole(roles ...string)`** — returns 403 if user role not in allowed list.
- Passwords hashed with bcrypt (cost 12).
- Session secret loaded from `.env`.

---

## Role-Based Access Control

| Route Prefix | Allowed Roles |
|---|---|
| `/admin/*` | admin, staff |
| `/restaurant/*` | restaurant (owner only) |
| `/rider/*` | rider |
| `/customer/*` | customer |
| `/` (browse) | any authenticated |

---

## Order Flow

```
Customer places order → pending
Restaurant confirms  → confirmed
Rider accepts order  → (rider_id assigned, status stays confirmed)
Rider at restaurant: takes pickup photo → Start Delivery → in_delivery
Rider at customer:   takes delivery photo → Delivery Completed → delivered

Restaurant or admin can cancel → cancelled
```

### Rider Google Maps Links
- To restaurant: `https://www.google.com/maps/dir/?api=1&destination={lat},{lng}`
- To customer: `https://www.google.com/maps/dir/?api=1&destination={lat},{lng}`
- Open in new tab via `target="_blank"`.

---

## Image Upload Flow

1. User submits multipart form.
2. Go handler validates file type (image only) + size limit.
3. Handler uploads to MinIO via `aws-sdk-go-v2` S3 client.
4. Object key stored in DB column.
5. Public images: served via MinIO public endpoint directly.
6. Private images: served via Go proxy handler with role check.

---

## Basket (Shopping Cart)

- Stored server-side in the gorilla/sessions session.
- Structure: `restaurantID + map[menuItemID]quantity`.
- Adding an item from a **different restaurant** → HTMX modal to confirm basket clear.
- Basket cleared on successful checkout.

---

## Customer Address

- Single address per user: `users.address`, `users.lat`, `users.lng`.
- Edited from profile page.
- **Required before checkout** — checkout handler redirects to address form if not set.
- Address snapshot saved to `orders` table at checkout time.

---

## Admin & Staff Capabilities

- CRUD all users (including role assignment).
- CRUD all restaurants.
- CRUD all menu items.
- View and update all orders.
- View proof photos.
- Dashboard with order/user stats.

---

## Feature Checklist (Phases)

### Phase 1 — Foundation
- [ ] Folder structure + go.mod updates
- [ ] Config loading (.env + godotenv)
- [ ] GORM DB connection + AutoMigrate all models
- [ ] Session store setup (gorilla/sessions + pgstore)
- [ ] Base HTML layout template (Bootstrap 5 CDN, HTMX CDN)
- [ ] MinIO client setup (aws-sdk-go-v2)
- [ ] Seed command (`cmd/seed/main.go`)

### Phase 2 — Authentication
- [ ] Register handler + template
- [ ] Login handler + template
- [ ] Logout handler
- [ ] RequireLogin middleware
- [ ] RequireRole middleware
- [ ] Test: auth flow

### Phase 3 — User Profile & Address
- [ ] View/edit profile page
- [ ] Address form (lat/lng fields)
- [ ] Test: address save/update

### Phase 4 — Restaurant & Menu (Restaurant Role)
- [ ] Restaurant dashboard
- [ ] CRUD restaurant (with image upload)
- [ ] CRUD menu items (with image upload)
- [ ] Test: restaurant CRUD

### Phase 5 — Customer Browse & Basket
- [ ] Restaurant listing page
- [ ] Restaurant detail + menu page
- [ ] Add to basket (HTMX)
- [ ] View/edit basket (HTMX)
- [ ] Restaurant switch modal (HTMX)
- [ ] Test: basket operations

### Phase 6 — Checkout & Orders
- [ ] Checkout page (address, basket, delivery fee preview)
- [ ] Haversine delivery fee calculation (pkg/geo)
- [ ] Place order handler (creates order + order_items, clears basket)
- [ ] Customer order list + detail page
- [ ] Test: checkout + order creation

### Phase 7 — Restaurant Order Management
- [ ] Restaurant order list (own restaurant only)
- [ ] Update status: confirm / cancel / ready_for_pickup
- [ ] Test: restaurant order management

### Phase 8 — Rider Flow
- [ ] Rider available order list (confirmed + no rider)
- [ ] Accept order (assign rider_id)
- [ ] Navigate to restaurant link
- [ ] Upload pickup proof → Start Delivery → in_delivery
- [ ] Navigate to customer link
- [ ] Upload delivery proof → Delivery Completed → delivered
- [ ] Test: full rider flow

### Phase 9 — Admin Panel
- [ ] Admin dashboard with stats
- [ ] CRUD users
- [ ] CRUD all restaurants + menu items
- [ ] View all orders (with proof photo access)
- [ ] Test: admin CRUD

### Phase 10 — Polish
- [ ] Private proof-photo proxy handler
- [ ] Server-side input validation
- [ ] Error pages (403, 404, 500)
- [ ] CSRF protection
- [ ] Full integration test pass

---

## Key Dependencies to Add

```
github.com/joho/godotenv
github.com/gorilla/sessions
github.com/gorilla/securecookie
github.com/antonlindstrom/pgstore        # PostgreSQL session store
gorm.io/gorm
gorm.io/driver/postgres
github.com/aws/aws-sdk-go-v2/config
github.com/aws/aws-sdk-go-v2/service/s3
github.com/stretchr/testify
```

---

## Environment Variables (`.env.example`)

```env
APP_PORT=8080
APP_ENV=development
SESSION_SECRET=change-me-32-bytes-minimum

DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=example
DB_NAME=delivery_app

MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=miniominio
MINIO_SECRET_KEY=miniominio
MINIO_USE_SSL=false
MINIO_PUBLIC_BUCKET=delivery-public
MINIO_PRIVATE_BUCKET=delivery-private
MINIO_PUBLIC_URL=http://localhost:9000/delivery-public
```

---

## Key Architectural Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Auth | Session-based | Simpler with SSR + HTMX; no token management |
| Session store | PostgreSQL via pgstore | No extra infra; consistent with existing DB |
| Money | Integer satang | Avoids floating-point errors |
| Delivery fee | Haversine (server-side) | No external API key required |
| Delivery rate | Per-restaurant configurable | Flexibility for different zones |
| Basket | Server-side session | Secure, no client-side state leakage |
| Address | Single address per user | Simple; snapshot saved to order at checkout |
| Image upload | Client → Go → MinIO (aws-sdk-go-v2) | Consistent SDK; server controls validation |
| Public/private images | Two MinIO buckets | Proof photos protected; food images fast CDN-style |
| HTMX pattern | `HX-Request` header check → return partial or full page | Standard HTMX + SSR pattern |
| Migrations | GORM AutoMigrate | Dev speed; schema evolves with models |
| Bootstrap | CDN | No build step needed |
