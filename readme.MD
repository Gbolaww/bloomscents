# Bloom Scents

A mobile-first PWA for a perfume business: onboarding, customer signup/login, a product catalog, Paystack checkout, a WhatsApp enquiry link per product, order history, and admin email notifications on paid orders.

This is a single Go service — the frontend files are embedded directly into the binary at compile time (via `//go:embed`), so one deploy gives you both the API and the website on one URL. No separate frontend hosting needed.

## Structure

```
bloomscents/
└── backend/
    ├── frontend/     Mobile PWA (index.html, manifest.json, service-worker.js) — embedded into the binary
    ├── main.go       Routes + embedded static file serving
    ├── auth.go, products.go, orders.go, paystack.go, email.go, db.go, models.go
    ├── migrations.sql
    └── go.mod
```

Everything lives under `backend/` — that whole folder is what you push as one GitHub repo and deploy as one Railway service.

## 1. Backend setup

```
cd backend
go mod tidy      # downloads lib/pq, golang-jwt, x/crypto
```

Set these environment variables (locally via `export`, or in Railway's dashboard):

| Variable | Example | Notes |
|---|---|---|
| `DATABASE_URL` | `postgres://user:pass@host:5432/dbname?sslmode=require` | Railway gives you this when you add a Postgres plugin |
| `JWT_SECRET` | a long random string | used to sign login sessions |
| `PAYSTACK_SECRET_KEY` | `sk_test_...` | from your Paystack dashboard — start with the test key |
| `RESEND_API_KEY` | `re_...` | from resend.com — free tier covers 3,000 emails/month |
| `RESEND_FROM` | `onboarding@resend.dev` | optional; defaults to this Resend test sender until you verify your own domain |
| `ADMIN_EMAIL` | `you@example.com` | where new-order notifications go |
| `PORT` | `8080` | Railway sets this automatically |

Run the migration once against your database:

```
psql "$DATABASE_URL" -f migrations.sql
```

**If you're upgrading an existing deployment** (you already ran `migrations.sql` before and have live data), also run `migrations_v2.sql` — this adds cart support (multiple items per order):

```
psql "$DATABASE_URL" -f migrations_v2.sql
```

**If you're upgrading further** (cart support was v2), also run `migrations_v3.sql` — this adds Google Sign-In support:

```
psql "$DATABASE_URL" -f migrations_v3.sql
```

### Google Sign-In setup
1. Go to console.cloud.google.com, create a project (or use an existing one)
2. APIs & Services → Credentials → Create Credentials → OAuth client ID → Web application
3. Under "Authorized JavaScript origins," add your live site URL (e.g. `https://bloomscents-production.up.railway.app`)
4. Copy the generated Client ID
5. In `backend/frontend/index.html`, set `const GOOGLE_CLIENT_ID = '...'` to that value
6. On Railway, add an environment variable `GOOGLE_CLIENT_ID` with the **same** value (the backend uses it to verify tokens actually came from your app)

Until this is configured, the Google Sign-In button stays hidden automatically — email/password signup still works fine either way.

### Hero image
`backend/frontend/hero-bella-vita.jpg` is used on the homepage hero and as the onboarding background. Swap this file for a real product photo whenever you have one — same filename, and it'll show up in both places automatically.

This creates the tables and seeds 3 placeholder products — edit or replace them directly in the `products` table (via `psql`, a GUI, or now the admin dashboard's Products tab) with real names, notes, prices (in kobo — e.g. ₦15,000 = `1500000`), and image URLs.

**Creating the admin login:** there's no signup form for admins on purpose. Generate a bcrypt hash for the admin's password (e.g. using an online bcrypt generator or `htpasswd -bnBC 10 "" yourpassword | tr -d ':\n'` as a rough stand-in — for production, write a 5-line Go script using `bcrypt.GenerateFromPassword`), then insert directly:

```sql
INSERT INTO admins (email, password_hash) VALUES ('admin@bloomscents.com', '<bcrypt-hash>');
```

Run locally:
```
go run .
```

Deploy to Railway: push this `backend/` folder as its own service, add a Postgres plugin, set the environment variables above, and Railway will build and run it (`go build` is auto-detected).

**Paystack webhook:** in your Paystack dashboard, set the webhook URL to:
`https://YOUR-BACKEND-URL/api/paystack/webhook`

## 2. Frontend setup

Open `backend/frontend/index.html` and update the top of the `<script>` block:

```js
const API_BASE = ''; // leave empty — same-origin, since the frontend is served by this same backend
const WHATSAPP_NUMBER = '2348012345678'; // the business's WhatsApp number, country code + number, no + or leading 0
```

Since the frontend is now embedded into the Go binary and served from the same service, `API_BASE` can just be an empty string — the browser will call `/api/...` on whatever domain the page is loaded from, no separate URL needed.

Add two icon files for the "Add to Home Screen" experience:
- `backend/frontend/icon-192.png` (192×192)
- `backend/frontend/icon-512.png` (512×512)

Any square version of the Bloom Scents logo works — I can generate simple placeholder ones if you'd like. After adding them, rebuild (`go build`) so they get embedded into the binary.

Nothing to deploy separately — pushing and deploying `backend/` (step 3 below) ships the frontend too.

## 3. How it works end-to-end

1. Customer opens the app → onboarding → signs up or logs in (or browses as guest)
2. Taps a product → sees details → either:
   - **Enquire**: opens WhatsApp with a pre-filled message naming that exact product
   - **Add to Cart**: adds it to their cart (visible via the cart icon/badge)
3. From the cart, they can adjust quantities, remove items, add a delivery address, and tap **Checkout** — this pays for the whole cart in one Paystack transaction
4. On successful payment, Paystack calls the webhook → order marked `paid` → you get an email with every item in that order
5. Customer can see their order history (with all items per order) under Account
6. You (or her) log in via the "Admin login" link on the customer login screen, landing on a dashboard with two tabs:
   - **Orders**: every order, its items, status, and delivery address
   - **Products**: add, edit, or delete perfumes directly — no more editing the database by hand

## Next steps you may want later
- Order status updates (e.g. "shipped") visible to the customer
- Product photo uploads (currently image_url is a plain text field — pairs well with Cloudinary, which you've already used before)
- Push notifications or SMS for order updates