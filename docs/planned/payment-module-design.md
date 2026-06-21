# Payment Module — Design Spec

**Date:** 2026-06-21 · **Status:** Draft, pre-approval · **Owner:** Andi Alifsyah

## 1. Context & motivation

YPA Handayani currently has **no payment infrastructure**. When a student registers for
a course (e.g. Kursus Mengemudi SIM A — Rp 2.500.000 + Rp 150.000 registration fee) all
payment coordination happens offline — WhatsApp chats, phone calls, and cash at the
outlet. There is no digital record linking payments to students, courses, or quota.

The `courses` table already stores `price` and `registration_fee`; the `students` table
tracks enrollment + quota hours. The missing piece is a **Payment** entity that bridges
the two: *"Student X paid Rp Y for Course Z via method W, confirmed at time T."*

Two integration levels are needed:

1. **Online payments via Xendit** — Indonesia's leading payment gateway. Its
   [Invoice API v2](https://docs.xendit.co/docs/overview) creates a hosted checkout
   page supporting VA, QRIS, e-wallets (OVO/DANA/ShopeePay/LinkAja), credit/debit
   cards, and retail outlets (Alfamart/Indomaret). We create an invoice server-side,
   hand the URL to the student, and receive a webhook when they pay. No custom payment
   UI required.

2. **Manual payments** — this is a local vocational school with physical outlets. Many
   customers pay cash at the counter, transfer via mobile banking and confirm over
   WhatsApp, or are older adults unfamiliar with digital payment links. Manual channels
   are **first-class**, not a legacy fallback.

## 2. Goals & non-goals

**Goals**
- Unified `payments` data model covering all channels (Xendit, manual transfer,
  WhatsApp-confirmed, cash at outlet).
- Xendit Invoice API v2 integration for online payment collection + webhook-based
  status updates.
- Admin dashboard UI for payment management: create, list, review, confirm manual
  payments, upload proof-of-transfer.
- Link confirmed payments to student enrollment: auto-activate `Student` record +
  allocate quota hours.
- Student-facing payment lookup page (view status, upload transfer proof).
- Support partial / installment payments for higher-value courses.
- WhatsApp payment notifications (piggybacks on the planned WAHA integration; fully
  optional — payment module works without it).

**Non-goals (this iteration)**
- Refund/dispute handling via Xendit API (handle manually in the Xendit Dashboard).
- Recurring/subscription billing (courses are one-time purchases).
- Multi-currency support (IDR only).
- Building custom payment-method UIs (Xendit's hosted checkout handles this).
- Accounting/ledger module (this tracks payments, not double-entry books).
- Payment gateway alternatives (e.g. Midtrans) — Xendit only for now.
- Landing-page self-service checkout (admin-initiated only; the data model supports
  adding public checkout later).

## 3. Architecture decision

**All-Go**, inside `core/`. The Go gateway is already the single public front door; the
Xendit API key must never reach the frontend or the Python AI service. This matches the
project's established boundary (Go gateway = everything except AI) and the `waha.Client`
pattern from the WhatsApp integration spec.

| Concern | Decision | Rationale |
|---|---|---|
| Xendit SDK | Raw HTTP client (`core/xendit/`) | No official Go SDK; thin client matches `waha.Client` pattern |
| Invoice creation | `POST /v2/invoices` (Invoice API v2) | Hosted checkout handles all payment-method UIs |
| Status updates | Xendit webhooks → `POST /api/webhooks/xendit` | Real-time; validated by `x-callback-token` header |
| Manual payments | Admin confirms via dashboard API | No external service needed |
| Payment tables | Go-owned, GORM `AutoMigrate` | Consistent with all other Go-owned tables |
| JSON wire format | camelCase | Matches existing Angular contract |

New code organised as:
- `core/xendit/` — thin Xendit HTTP client (invoice create/get/expire).
- `core/models/payment.go` — GORM structs (`Payment`, `PaymentItem`, `PaymentProof`).
- `core/handlers/payments.go` — CRUD + Xendit invoice creation + manual confirmation.
- `core/handlers/xendit_webhook.go` — webhook receiver.
- `core/seed/payments.go` — sample dev/demo payments.

```
                  ┌─────────────┐
  Browser ──────▶ │  Angular SPA │
                  │  (:4200)     │
                  └──────┬───────┘
                         │ Bearer JWT
                  ┌──────▼────────────────┐
                  │  Go Gateway (:8080)    │
                  │                        │
  Xendit          │  /api/admin/payments/* │──────▶ api.xendit.co
  Webhooks ──────▶│  /api/webhooks/xendit  │        POST /v2/invoices
                  │  /api/payments/:code   │
                  └──────┬────────────────┘
                         │
                  ┌──────▼───────┐     ┌──────────────┐
                  │  MySQL 8     │     │  WAHA (opt.)  │
                  │  payments,   │     │  WA notifs    │
                  │  payment_*   │     └──────────────┘
                  └──────────────┘
```

## 4. Data model

All tables are **owned by the Go gateway and added to `AutoMigrate`**. JSON is
**camelCase**. Tables follow the existing pattern in `models/models.go`.

### 4.1 `Payment` — the core payment record

| Field | Type | Notes |
|---|---|---|
| `id` | uint PK | |
| `paymentCode` | string, unique | human-readable, e.g. `PAY-20260621-0001` |
| `studentId` | uint nullable FK→students | linked enrolled student |
| `studentCrmId` | uint nullable FK→students_crm | linked CRM lead (pre-enrollment) |
| `studentName` | string | denormalized for display |
| `studentPhone` | string | for WhatsApp / contact |
| `totalAmount` | int64 (IDR) | sum of items; no decimals |
| `paidAmount` | int64 | amount actually received |
| `status` | enum `pending, paid, partial, expired, cancelled, refunded` | |
| `paymentMethod` | enum `xendit, manual_transfer, whatsapp, cash` | how the student pays |
| `paymentChannel` | string nullable | detail: `BCA`, `OVO`, `QRIS`, `Alfamart`, etc. |
| `xenditInvoiceId` | string nullable, unique | Xendit invoice ID |
| `xenditInvoiceUrl` | string nullable | Xendit hosted checkout URL |
| `xenditExternalId` | string nullable, unique | our `paymentCode` sent to Xendit |
| `expiresAt` | datetime nullable | Xendit expiry or manual deadline |
| `paidAt` | datetime nullable | when payment was confirmed/received |
| `confirmedBy` | uint nullable FK→users | manager who confirmed (manual payments) |
| `notes` | text | admin notes / customer reference |

### 4.2 `PaymentItem` — line items within a payment

Supports multi-item payments (course fee + registration fee + SIM procedure fees + quota
top-up).

| Field | Type | Notes |
|---|---|---|
| `id` | uint PK | |
| `paymentId` | uint FK→payments | |
| `itemType` | enum `course_fee, registration_fee, sim_fee, quota_topup, other` | |
| `description` | string | e.g. "Kursus Mengemudi SIM A - Mobil Manual" |
| `courseId` | uint nullable FK→courses | linked course |
| `amount` | int64 | item amount in IDR |
| `quantity` | int, default 1 | |

### 4.3 `PaymentProof` — proof-of-payment for manual transfers

| Field | Type | Notes |
|---|---|---|
| `id` | uint PK | |
| `paymentId` | uint FK→payments | |
| `fileUrl` | string | uploaded file path |
| `fileName` | string | original filename |
| `fileType` | string | MIME type |
| `uploadedAt` | datetime | |
| `uploadedBy` | string | `customer` or manager username |

## 5. Xendit client package (`core/xendit/`)

A thin, typed Go HTTP client wrapping the Xendit REST API. **No business logic** — just
HTTP calls + JSON marshalling. Pattern mirrors `core/waha/` from the WhatsApp spec.

```go
package xendit

type Client struct {
    SecretKey  string       // Basic Auth username, password empty
    BaseURL    string       // "https://api.xendit.co"
    HTTPClient *http.Client
}

// Invoice API v2
func (c *Client) CreateInvoice(req CreateInvoiceReq) (*Invoice, error)
func (c *Client) GetInvoice(id string) (*Invoice, error)
func (c *Client) ExpireInvoice(id string) (*Invoice, error)
```

Authentication: HTTP Basic Auth — `SecretKey` as username, empty password.

```go
func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
    // ...
    req.SetBasicAuth(c.SecretKey, "")
    // ...
}
```

Key request/response types:

```go
type CreateInvoiceReq struct {
    ExternalID         string          `json:"external_id"`
    Amount             int64           `json:"amount"`
    Description        string          `json:"description"`
    Currency           string          `json:"currency"`        // "IDR"
    InvoiceDuration    int             `json:"invoice_duration"` // seconds, default 86400
    Customer           *CustomerObject `json:"customer,omitempty"`
    SuccessRedirectURL string          `json:"success_redirect_url,omitempty"`
    FailureRedirectURL string          `json:"failure_redirect_url,omitempty"`
    Items              []InvoiceItem   `json:"items,omitempty"`
}

type Invoice struct {
    ID         string `json:"id"`
    ExternalID string `json:"external_id"`
    Status     string `json:"status"` // PENDING, PAID, EXPIRED, SETTLED
    Amount     int64  `json:"amount"`
    InvoiceURL string `json:"invoice_url"`
    ExpiryDate string `json:"expiry_date"`
}
```

Xendit uses the **same API URL** for sandbox and production. The environment is
determined by the key prefix: `xnd_development_*` (test) vs `xnd_production_*` (live).

## 6. Webhook receiver

### 6.1 Endpoint

New public route: `POST /api/webhooks/xendit` — no JWT middleware, validated by
the `x-callback-token` header (configured in the Xendit Dashboard and stored as
`XENDIT_CALLBACK_TOKEN` env var).

```go
func XenditWebhook(c *gin.Context) {
    // 1. Validate x-callback-token header
    if c.GetHeader("x-callback-token") != os.Getenv("XENDIT_CALLBACK_TOKEN") {
        c.JSON(401, gin.H{"error": "Unauthorized"})
        return
    }
    // 2. Parse payload
    var payload WebhookPayload
    c.ShouldBindJSON(&payload)
    // 3. Idempotency: look up by xenditInvoiceId; if already paid, return 200
    // 4. Update Payment: status, paidAmount, paymentChannel, paidAt
    // 5. Post-payment actions (if PAID):
    //    a. Auto-create/update Student record (quota allocation)
    //    b. Update StudentCrm status → active
    //    c. WhatsApp confirmation (if WAHA configured)
    // 6. Return 200
}
```

### 6.2 Webhook payload (Xendit Invoice callback)

```go
type WebhookPayload struct {
    ID             string `json:"id"`
    ExternalID     string `json:"external_id"`
    Status         string `json:"status"`          // PAID, EXPIRED
    PaidAmount     int64  `json:"paid_amount"`
    PaymentMethod  string `json:"payment_method"`   // BANK_TRANSFER, EWALLET, QR_CODE, etc.
    PaymentChannel string `json:"payment_channel"`  // BCA, OVO, QRIS, etc.
    PaidAt         string `json:"paid_at"`
}
```

## 7. Payment flows (by channel)

### 7.1 Xendit (online)

1. Manager creates payment via `POST /api/admin/payments` with `paymentMethod: "xendit"`.
2. Go handler generates `paymentCode`, calls Xendit `POST /v2/invoices`, stores
   `xenditInvoiceId` + `xenditInvoiceUrl` + `expiresAt`.
3. Manager shares the `invoiceUrl` with the student (verbally, WhatsApp, or system
   auto-sends if WAHA configured).
4. Student opens Xendit checkout → selects method → pays.
5. Xendit sends webhook → handler updates `Payment` → post-payment automation fires.

### 7.2 Manual bank transfer

1. Manager creates payment with `paymentMethod: "manual_transfer"`, status=`pending`.
2. Manager shares bank details + `paymentCode` with student:
   > *Transfer ke: BCA 1234567890 a.n. YPA Handayani*
   > *Jumlah: Rp 2.650.000 — Kode: PAY-20260621-0001*
3. Student transfers, optionally sends screenshot.
4. Manager (or student via public endpoint) uploads proof-of-transfer.
5. Manager confirms via `POST /api/admin/payments/:id/confirm` → post-payment fires.

### 7.3 WhatsApp-coordinated

Identical to manual transfer but `paymentMethod: "whatsapp"` — semantically indicates
the payment was coordinated and confirmed through WhatsApp conversation with admin.

### 7.4 Cash at outlet

1. Student pays at counter.
2. Manager creates payment with `paymentMethod: "cash"` and immediately confirms.
3. Post-payment automation fires.

## 8. Post-payment automation

When a payment transitions to `status=paid`:

- **CRM → Student enrollment:** if payment links to a `StudentCrm` record, create a
  `Student` record with quota hours from the course, update `StudentCrm.status` →
  `active`, assign instructor (chosen in payment form or a default).
- **Quota top-up:** if payment links to an existing `Student`, add quota hours.
- **WhatsApp notification** (optional, requires WAHA):
  ```
  ✅ *Pembayaran Diterima*

  Halo {name}!
  📋 Kode: {paymentCode}
  💰 Jumlah: Rp {amount}
  📝 {itemDescriptions}

  Hubungi kami untuk jadwal sesi pertama.
  📞 082191927620
  — YPA Handayani
  ```
- If WAHA is not configured, WhatsApp notifications are silently skipped (graceful
  degradation, consistent with every other optional integration in the project).

## 9. API surface (Go gateway)

Conventions match existing handlers: bare-body JSON, camelCase, `AuthMiddleware`-gated.
Manager-only for administration; public endpoints for student-facing lookup.

| Method | Path | Auth | Purpose |
|---|---|---|---|
| GET | `/api/admin/payments` | manager | list payments (paginated, filtered by status/method/date/search) |
| GET | `/api/admin/payments/:id` | manager | payment detail + items + proofs |
| POST | `/api/admin/payments` | manager | create payment (Xendit or manual) |
| PUT | `/api/admin/payments/:id` | manager | update notes / status corrections |
| POST | `/api/admin/payments/:id/confirm` | manager | confirm manual payment as paid |
| POST | `/api/admin/payments/:id/cancel` | manager | cancel/expire a pending payment |
| POST | `/api/admin/payments/:id/resend` | manager | resend Xendit link / WhatsApp |
| POST | `/api/admin/payments/:id/proof` | manager | upload proof-of-payment |
| GET | `/api/admin/payments/summary` | manager | dashboard totals (today/month, by method) |
| GET | `/api/payments/:code` | public | student lookup by payment code |
| POST | `/api/payments/:code/proof` | public | student uploads transfer proof |
| POST | `/api/webhooks/xendit` | callback-token | Xendit webhook receiver |

### 9.1 Create-payment request body

```json
{
    "studentId": 5,
    "studentName": "Budi Santoso",
    "studentPhone": "082191927620",
    "paymentMethod": "xendit",
    "items": [
        {
            "itemType": "course_fee",
            "description": "Kursus Mengemudi SIM A - Mobil Manual",
            "courseId": 3,
            "amount": 2500000
        },
        {
            "itemType": "registration_fee",
            "description": "Biaya Pendaftaran",
            "amount": 150000
        }
    ],
    "notes": "Siswa baru, pembayaran pertama"
}
```

When `paymentMethod` is `"xendit"`, the handler calls Xendit `POST /v2/invoices` and
stores the invoice URL. Otherwise it creates a `pending` manual record.

## 10. Frontend (Angular)

### 10.1 New dashboard route

`/dashboard/pembayaran` — lazy-loaded, manager-only, guarded by `authGuard` + role
check. Indonesian path name `pembayaran` (= "payment") matches the convention
(`kursus`, `sesi`, `mekanisme`).

### 10.2 Components

| Component | Purpose |
|---|---|
| `PembayaranDashboardComponent` | main page: summary cards + tab navigation |
| `PaymentSummaryCardsComponent` | revenue today/month, pending count, breakdown by method |
| `PaymentListComponent` | paginated table with filters (status, method, date range, search) |
| `PaymentDetailComponent` | full detail: items, proofs, status timeline, action buttons |
| `CreatePaymentDialogComponent` | modal: select student → add items → choose method → create |
| `ConfirmPaymentDialogComponent` | modal: confirm manual payment with notes |
| `ProofUploadComponent` | drag-and-drop proof upload (reusable) |

### 10.3 New models (`core/models/payment.model.ts`)

```typescript
export type PaymentStatus = 'pending' | 'paid' | 'partial' | 'expired' | 'cancelled' | 'refunded';
export type PaymentMethod = 'xendit' | 'manual_transfer' | 'whatsapp' | 'cash';

export interface Payment {
  id: number;
  paymentCode: string;
  studentId: number | null;
  studentName: string;
  studentPhone: string;
  totalAmount: number;
  paidAmount: number;
  status: PaymentStatus;
  paymentMethod: PaymentMethod;
  paymentChannel: string | null;
  xenditInvoiceUrl: string | null;
  expiresAt: string | null;
  paidAt: string | null;
  notes: string;
  items: PaymentItem[];
  proofs: PaymentProof[];
  createdAt: string;
}

export interface PaymentItem {
  id: number;
  itemType: string;
  description: string;
  courseId: number | null;
  amount: number;
  quantity: number;
}

export interface PaymentProof {
  id: number;
  fileUrl: string;
  fileName: string;
  uploadedAt: string;
  uploadedBy: string;
}

export interface PaymentSummary {
  totalToday: number;
  totalThisMonth: number;
  pendingCount: number;
  byMethod: { method: string; count: number; total: number }[];
}
```

### 10.4 `ApiService` additions + mock fallbacks

```typescript
// Manager
getPayments(params?): Observable<Payment[]>
getPayment(id: number): Observable<Payment>
createPayment(data): Observable<Payment>
confirmPayment(id: number, notes?: string): Observable<Payment>
cancelPayment(id: number): Observable<Payment>
resendPaymentLink(id: number): Observable<any>
uploadPaymentProof(id: number, file: File): Observable<PaymentProof>
getPaymentSummary(): Observable<PaymentSummary>

// Public
lookupPayment(code: string): Observable<Payment>
uploadCustomerProof(code: string, file: File): Observable<PaymentProof>
```

All with mock fallbacks per the existing graceful-degradation pattern.

### 10.5 i18n

New keys in `frontend/public/i18n/{id,en}.json`:

```json
{
  "payment.title": "Pembayaran",
  "payment.create": "Buat Pembayaran",
  "payment.confirm": "Konfirmasi Pembayaran",
  "payment.status.pending": "Menunggu",
  "payment.status.paid": "Lunas",
  "payment.status.partial": "Sebagian",
  "payment.status.expired": "Kedaluwarsa",
  "payment.method.xendit": "Xendit (Online)",
  "payment.method.manual_transfer": "Transfer Manual",
  "payment.method.whatsapp": "Via WhatsApp",
  "payment.method.cash": "Tunai",
  "payment.proof.upload": "Upload Bukti Transfer"
}
```

## 11. Environment variables

| Variable | Default | Required | Notes |
|---|---|---|---|
| `XENDIT_SECRET_KEY` | *(empty)* | **prod** | API secret; `xnd_development_*` for sandbox |
| `XENDIT_CALLBACK_TOKEN` | *(empty)* | **prod** | from Xendit Dashboard → Webhooks |
| `XENDIT_INVOICE_DURATION` | `86400` | no | invoice expiry in seconds (24h) |
| `XENDIT_SUCCESS_REDIRECT` | `http://localhost:4200/pembayaran/sukses` | no | post-payment redirect |
| `XENDIT_FAILURE_REDIRECT` | `http://localhost:4200/pembayaran/gagal` | no | post-payment redirect |
| `PAYMENT_CODE_PREFIX` | `PAY` | no | prefix for generated codes |

When `XENDIT_SECRET_KEY` is empty: Xendit features disabled, only manual payment methods
available. Dashboard shows an informational notice. No crash.

## 12. Integration with existing features

- **Courses → payment items:** `CreatePaymentDialog` auto-populates `price` and
  `registration_fee` from the `courses` table.
- **CRM → enrollment:** a payment linked to `studentCrmId` auto-creates a `Student`
  record on confirmation, closing the lead→active pipeline gap.
- **Students → quota:** a payment linked to `studentId` tops up
  `remaining_quota_hours`.
- **WhatsApp (WAHA):** payment confirmations logged to `WhatsAppMessageLog` with
  `context=payment`; Xendit invoice URLs can be shared via WAHA `SendText`. Fully
  optional — the payment module works without WAHA.
- **Payroll (future):** payment revenue data may inform instructor variable-pay
  reporting, but no direct integration in this iteration.

## 13. Error handling & edge cases

- **Xendit API unreachable:** payment creation falls back to `manual_transfer` with a
  notice; no crash.
- **Duplicate webhooks:** idempotent — check `xenditInvoiceId` + current status before
  processing; already-paid webhooks return 200 immediately.
- **Partial payments:** `paidAmount < totalAmount` → status `partial`; manager confirms
  remainder later or creates a second payment.
- **Expired invoices:** Xendit sends `EXPIRED` webhook → status updated; manager can
  create a new payment.
- **File upload limits:** proof-of-payment capped at 5 MB; accepted: JPG, PNG, PDF.
- **Payment code uniqueness:** `PAY-YYYYMMDD-NNNN` daily counter; UUID fallback on
  collision.
- **WAHA not configured:** WhatsApp notifications silently skipped.

## 14. Open items & research tasks

- **⚠#1 Xendit account** — confirm whether a Xendit account already exists, or include
  sign-up steps. Sandbox (test) environment is free; no payment needed to start
  development.
- **⚠#2 Bank details for manual transfer** — confirm the bank account details to
  display to customers (e.g. "BCA 1234567890 a.n. YPA Handayani"). This may become a
  `SystemSettings` entry so managers can change it from the dashboard.
- **⚠#3 Auto-enrollment behaviour** — confirm whether a confirmed payment should
  automatically create a `Student` record + assign a default instructor, or whether
  enrollment should remain a separate manual step after payment.
- **⚠#4 Installments** — confirm whether split payments (e.g. 50% now, 50% later) need
  first-class UI support, or if the current model (multiple payments per student,
  `partial` status) is sufficient.
- **⚠#5 Receipt PDF** — confirm whether a printable receipt/invoice should be generated
  after payment. Would use `maroto v2` (same as payroll payslips).
- **⚠#6 Enabled payment methods** — by default Xendit enables all activated methods.
  Confirm whether to restrict (e.g. VA + QRIS + e-wallets only, no credit cards).

## 15. Phasing (sequenced for incremental delivery)

1. **Data model + master CRUD** — `Payment`, `PaymentItem`, `PaymentProof` models,
   `AutoMigrate`, manual payment CRUD endpoints, payment code generation.
   (≈2 days) · **Status: Planned** 🔲
2. **Xendit client + invoice creation** — `core/xendit/` HTTP client, `CreateInvoice`
   call from the payment handler, store invoice URL + expiry.
   (≈1–2 days) · **Status: Planned** 🔲
3. **Webhook receiver + status updates** — `POST /api/webhooks/xendit`, callback-token
   validation, idempotent status transitions.
   (≈1 day) · **Status: Planned** 🔲
4. **Post-payment automation** — student enrollment (CRM → Student), quota allocation,
   WhatsApp notification (if WAHA available).
   (≈1–2 days) · **Status: Planned** 🔲
5. **Angular dashboard (`/dashboard/pembayaran`)** — manager payment management UI,
   summary cards, list/detail views, create/confirm dialogs, proof upload.
   (≈3–4 days) · **Status: Planned** 🔲
6. **Public payment lookup + customer proof upload** — `/pembayaran/:code` page,
   status display, proof upload for students.
   (≈1 day) · **Status: Planned** 🔲

Each phase ships and is demoable independently.
