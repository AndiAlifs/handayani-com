# WhatsApp Integration via WAHA — Design Spec

**Date:** 2026-06-21 · **Status:** Draft, pre-approval · **Owner:** Andi Alifsyah

## 1. Context & motivation

YPA Handayani's customer communication already revolves around WhatsApp — the landing
page CTAs point to `wa.me/082191927620` and `wa.me/082193234971`, the chatbot widget
recommends users reach out via WhatsApp, and the Student model stores a `whatsapp`
field. Yet no programmatic WhatsApp integration exists: the web chatbot returns random
canned responses, session reminders require manual coordination, and the planned payroll
module lists WhatsApp delivery as an **open item** (§11 ⚠ in
`payroll-module-design.md`: *"WhatsApp gateway choice — pick the gateway (Cloud API vs
local) and its env contract"*).

[WAHA](https://waha.devlike.pro/) (WhatsApp HTTP API) is a self-hosted, Docker-based
gateway that exposes the WhatsApp Web protocol over a clean REST API — sessions, QR
pairing, sending text/image/file/voice messages, receiving webhooks on incoming
messages, and more. It runs as a single container alongside the existing stack with no
external SaaS dependency.

> **This document specifies five integration areas**, ordered by dependency and
> business impact. Each ships and demos independently.

## 2. Goals & non-goals

**Goals**
- Deploy WAHA as a new service in the Docker Compose stack, with session management
  exposed to administrators via the dashboard.
- Build a thin, reusable WAHA client package in the Go gateway (`core/waha/`) that all
  features share.
- Deliver payslip PDFs and notification messages to employees via WhatsApp (resolving
  the payroll module's open item).
- Replace the static web chatbot with an AI-powered WhatsApp chatbot driven by the
  existing RAG knowledge base + Gemini.
- Automate session reminders and instructor daily schedules via WhatsApp.
- Send AI-generated follow-up / upsell recommendations to students via WhatsApp.

**Non-goals (this iteration)**
- Multi-device / multi-number support (one paired number is sufficient).
- WhatsApp Business API (Cloud API) — WAHA uses the Web protocol; Cloud API is a
  future option if volume warrants it.
- Replacing the web chatbot widget entirely — both channels coexist; the web widget
  upgrades to call the same AI pipeline.
- End-to-end message archival / compliance logging (future, if regulated).
- Group chat management.

## 3. Architecture decision

**All WhatsApp I/O goes through the Go gateway.** The Go gateway is already the single
public front door (port 8080); WAHA is an internal service (like the AI service) that
only the gateway talks to. This is consistent with the existing `aiproxy.go` pattern.

The Go gateway gains:
- `core/waha/` — a thin HTTP client package wrapping the WAHA REST API.
- `core/handlers/whatsapp.go` — admin-facing endpoints (session/QR management, send
  test message).
- `core/whatsapp/` — domain logic: message templates, chatbot orchestration, reminder
  scheduling.
- `core/models/whatsapp.go` — GORM structs for message logs and chatbot conversations.

The FastAPI AI service gains:
- `backend/app/routers/whatsapp_ai.py` — a new endpoint that accepts an incoming
  WhatsApp message + conversation history, queries the RAG knowledge base, calls Gemini,
  and returns a structured reply. The Go gateway calls this the same way it calls
  `/api/sessions/{id}/analyze`.

```
                  ┌─────────────┐
  Browser ──────▶ │  Angular SPA │
                  │  (:4200)     │
                  └──────┬───────┘
                         │ HTTP
                  ┌──────▼───────┐     ┌─────────────┐
                  │  Go Gateway  │────▶│  FastAPI AI  │
  Incoming WA     │  (:8080)     │     │  (:8081)     │
  Webhooks ──────▶│              │     │  + /wa/chat  │
                  └──────┬───────┘     └─────────────┘
                         │ HTTP
                  ┌──────▼───────┐
                  │  WAHA        │
                  │  (:3000)     │
                  └──────────────┘
                         │
                  WhatsApp Web Protocol
```

| Concern | Decision | Rationale |
|---|---|---|
| WAHA deployment | Docker Compose service | Matches existing stack; no external infra |
| WAHA API key | Shared via env var `WAHA_API_KEY` | WAHA supports `api_key` security |
| Go ↔ WAHA auth | `X-Api-Key` header on every request | Standard WAHA auth scheme |
| Webhook delivery | WAHA → Go gateway `POST /api/webhooks/whatsapp` | Public endpoint, validated by HMAC |
| Chatbot AI | Go gateway proxies to FastAPI `/api/wa/chat` | Keeps AI in Python, consistent with analyze |
| Phone format | Indonesian international format (`628...`) | Matches existing `normalizeWhatsApp()` in `handlers/instructor.go` |

## 4. Data model

All tables are **owned by the Go gateway and added to `AutoMigrate`**, consistent with
the attendance/instructor models. JSON is **camelCase** to match the existing wire
contract.

### 4.1 `WhatsAppSession` — WAHA session state cache

Mirrors the WAHA session state in the local DB so the dashboard can render status
without polling WAHA on every page load.

| Field | Type | Notes |
|---|---|---|
| `id` | uint PK | |
| `sessionName` | string, unique | WAHA session name (e.g. `default`) |
| `status` | enum `STOPPED, STARTING, SCAN_QR_CODE, WORKING, FAILED` | mirrors WAHA `SessionInfo.status` |
| `phoneNumber` | string | paired number, e.g. `6282191927620` |
| `pairedAt` | datetime nullable | when QR was scanned successfully |
| `lastSyncedAt` | datetime | last time we synced with WAHA |

### 4.2 `WhatsAppMessageLog` — outbound message audit trail

Every message sent through the Go gateway is logged for auditability and retry.

| Field | Type | Notes |
|---|---|---|
| `id` | uint PK | |
| `direction` | enum `outbound, inbound` | |
| `chatId` | string | WAHA chat ID, e.g. `6282191927620@c.us` |
| `phoneNumber` | string | normalised `628...` format |
| `messageType` | enum `text, image, file, voice, location, contact` | |
| `content` | text | message body or file caption |
| `mediaUrl` | string nullable | URL/path for media messages |
| `wahaMessageId` | string nullable | WAHA-returned message ID |
| `status` | enum `pending, sent, delivered, read, failed` | updated via webhooks |
| `errorDetail` | string nullable | reason on `failed` |
| `context` | enum `chatbot, reminder, payslip, notification, manual, upsell` | why the message was sent |
| `contextRefId` | uint nullable | FK to the originating record (e.g. payslip ID, learning plan ID) |
| `createdAt` | datetime | |
| `updatedAt` | datetime | |

### 4.3 `ChatbotConversation` — per-contact conversation state for the AI chatbot

Tracks conversation context so the AI can reference prior messages. Conversations are
scoped to a phone number and expire after a configurable inactivity window.

| Field | Type | Notes |
|---|---|---|
| `id` | uint PK | |
| `chatId` | string, unique | WAHA chat ID |
| `phoneNumber` | string | |
| `studentName` | string nullable | extracted from lead capture or CRM match |
| `studentCrmId` | uint nullable, FK→students_crm | linked CRM record, if matched |
| `status` | enum `active, expired, escalated` | |
| `messageCount` | int | total messages in this conversation |
| `lastMessageAt` | datetime | for expiry logic |
| `createdAt` | datetime | |

### 4.4 `ChatbotMessage` — individual messages within a conversation

| Field | Type | Notes |
|---|---|---|
| `id` | uint PK | |
| `conversationId` | uint FK→chatbot_conversations | |
| `role` | enum `user, assistant, system` | for Gemini context window |
| `content` | text | |
| `citations` | text nullable | JSON array of source IDs from RAG, e.g. `["kursus-3","mekanisme-5"]` |
| `createdAt` | datetime | |

### 4.5 `NotificationSchedule` — configurable notification rules

| Field | Type | Notes |
|---|---|---|
| `id` | uint PK | |
| `notificationType` | enum `session_reminder, daily_schedule, quota_low, upsell` | |
| `enabled` | bool | toggle per type |
| `leadTimeMinutes` | int | e.g. 60 = remind 1 hour before session |
| `cronExpression` | string nullable | for recurring (e.g. daily schedule at 07:00) |
| `templateText` | text | Go template string with placeholders |
| `updatedAt` | datetime | |

Seeded with sensible defaults (see §10).

## 5. WAHA client package (`core/waha/`)

A thin, typed Go HTTP client wrapping the WAHA REST API. **No business logic** — just
HTTP calls + JSON marshalling. All other packages import this.

```go
package waha

// Client wraps the WAHA HTTP API.
type Client struct {
    BaseURL    string       // e.g. "http://waha:3000"
    APIKey     string       // sent as X-Api-Key header
    Session    string       // default: "default"
    HTTPClient *http.Client
}

// Core methods (each maps 1:1 to a WAHA endpoint):

// Session management
func (c *Client) ListSessions() ([]SessionInfo, error)
func (c *Client) CreateSession(req CreateSessionReq) (*SessionDTO, error)
func (c *Client) GetSession() (*SessionInfo, error)
func (c *Client) StartSession() (*SessionDTO, error)
func (c *Client) StopSession() (*SessionDTO, error)
func (c *Client) RestartSession() (*SessionDTO, error)
func (c *Client) DeleteSession() error
func (c *Client) LogoutSession() (*SessionDTO, error)
func (c *Client) GetMe() (*MeInfo, error)

// Auth / Pairing
func (c *Client) GetQRCode(format string) ([]byte, error)  // format: "image" | "raw"
func (c *Client) RequestCode(req RequestCodeReq) error

// Messaging
func (c *Client) SendText(req SendTextReq) (*WAMessage, error)
func (c *Client) SendImage(req SendImageReq) (*WAMessage, error)
func (c *Client) SendFile(req SendFileReq) (*WAMessage, error)
func (c *Client) SendVoice(req SendVoiceReq) (*WAMessage, error)

// Typing indicators
func (c *Client) StartTyping(chatId string) error
func (c *Client) StopTyping(chatId string) error

// Seen
func (c *Client) SendSeen(req SendSeenReq) error
```

### 5.1 Request/response structs

Derived directly from the WAHA OpenAPI spec. Key structs:

```go
type SendTextReq struct {
    ChatID  string `json:"chatId"`          // "628xxx@c.us"
    Text    string `json:"text"`
    Session string `json:"session"`
}

type SendFileReq struct {
    ChatID   string `json:"chatId"`
    File     File   `json:"file"`            // { mimetype, filename, url or data }
    Caption  string `json:"caption,omitempty"`
    Session  string `json:"session"`
}

type File struct {
    Mimetype string `json:"mimetype"`
    Filename string `json:"filename"`
    URL      string `json:"url,omitempty"`     // send from URL
    Data     string `json:"data,omitempty"`     // send base64
}

type SessionInfo struct {
    Name   string         `json:"name"`
    Status string         `json:"status"` // STOPPED | STARTING | SCAN_QR_CODE | WORKING | FAILED
    Me     *MeInfo        `json:"me,omitempty"`
}

type MeInfo struct {
    ID       string `json:"id"`
    PushName string `json:"pushName"`
}
```

### 5.2 Error handling & graceful degradation

Consistent with the existing pattern (AI proxy returns 502 when down, `ApiService`
catches errors and falls back to mocks):

- All `waha.Client` methods return `(result, error)`.
- Callers (handlers, schedulers) **never crash on WAHA errors** — they log, record
  `failed` status in `WhatsAppMessageLog`, and continue.
- The frontend `ApiService` gains mock fallbacks for all new WhatsApp endpoints.
- WAHA being down does not block any existing feature — payroll still finalizes, CRM
  still works, sessions still log.

### 5.3 Chat ID convention

WAHA uses the WhatsApp chat ID format: `{phone}@c.us` for DMs. The `normalizeWhatsApp`
function in `handlers/instructor.go` already converts Indonesian numbers to `628...`
format. The WAHA client adds `@c.us` suffix automatically:

```go
func PhoneToChatID(phone string) string {
    phone = normalizeWhatsApp(phone)
    return phone + "@c.us"
}
```

## 6. Feature 1 — WAHA Service & Admin Dashboard (foundation)

### 6.1 Docker Compose service

New `waha` service in `compose.yml`:

```yaml
waha:
  image: devlikeapro/waha:latest
  environment:
    WHATSAPP_DEFAULT_ENGINE: WEBJS
    WAHA_DASHBOARD_ENABLED: "true"
    WAHA_DASHBOARD_USERNAME: ${WAHA_DASHBOARD_USERNAME:-admin}
    WAHA_DASHBOARD_PASSWORD: ${WAHA_DASHBOARD_PASSWORD:-admin}
    WHATSAPP_API_KEY: ${WAHA_API_KEY:-}
    WHATSAPP_HOOK_URL: http://core:8080/api/webhooks/whatsapp
    WHATSAPP_HOOK_EVENTS: "message,message.ack,session.status"
    WHATSAPP_HOOK_HMAC_KEY: ${WAHA_HMAC_KEY:-waha-hmac-secret}
  ports:
    - "3000:3000"      # WAHA dashboard (development only)
  volumes:
    - waha-data:/app/.sessions
  depends_on:
    db:
      condition: service_healthy
```

New volume: `waha-data` for persistent session data.

Environment variables added to `core/.env.example`:

```env
# WhatsApp (WAHA) integration
WAHA_URL=http://localhost:3000
WAHA_API_KEY=
WAHA_HMAC_KEY=waha-hmac-secret
WAHA_SESSION_NAME=default
```

### 6.2 Go gateway — session management endpoints

New file: `core/handlers/whatsapp.go`. All endpoints are **manager-only**, mounted
under `/api/admin/whatsapp`.

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/admin/whatsapp/status` | Get current WAHA session status + paired number |
| POST | `/api/admin/whatsapp/start` | Start/create WAHA session |
| POST | `/api/admin/whatsapp/stop` | Stop WAHA session |
| POST | `/api/admin/whatsapp/restart` | Restart WAHA session |
| POST | `/api/admin/whatsapp/logout` | Logout (unpair) + restart |
| GET | `/api/admin/whatsapp/qr` | Get QR code image for pairing (proxied from WAHA) |
| POST | `/api/admin/whatsapp/send-test` | Send a test message to a given number |
| GET | `/api/admin/whatsapp/messages` | List recent message log (paginated) |
| GET | `/api/admin/whatsapp/conversations` | List chatbot conversations |
| GET | `/api/admin/whatsapp/conversations/:id` | View conversation messages |
| GET | `/api/admin/whatsapp/notifications` | Get notification schedule settings |
| PUT | `/api/admin/whatsapp/notifications/:id` | Update notification settings |

### 6.3 Webhook receiver

New public endpoint: `POST /api/webhooks/whatsapp` (no auth middleware — validated by
HMAC).

```go
func WAWebhook(c *gin.Context) {
    // 1. Validate HMAC signature from X-Webhook-Hmac header
    // 2. Parse webhook payload (event type + body)
    // 3. Dispatch by event:
    //    - "message"        → chatbot handler (§8)
    //    - "message.ack"    → update WhatsAppMessageLog status
    //    - "session.status" → update WhatsAppSession cache
}
```

HMAC validation uses the `WAHA_HMAC_KEY` env var. The key is configured on both sides
(WAHA's `WHATSAPP_HOOK_HMAC_KEY` and Go's `WAHA_HMAC_KEY`).

### 6.4 Frontend — WhatsApp management page

New dashboard route: `/dashboard/whatsapp` (Indonesian path consideration: could be
`/dashboard/whatsapp` since the brand name is universal, unlike Indonesian-translated
routes for `kursus`, `sesi`, etc.).

Manager-only page with:

- **Connection status card**: shows paired number, status badge (🟢 Connected / 🟡
  Scanning QR / 🔴 Disconnected), last synced time.
- **QR code pairing**: when status is `SCAN_QR_CODE`, displays the QR image with
  auto-refresh (poll every 5s).
- **Action buttons**: Start / Stop / Restart / Logout.
- **Send test message**: input field for phone number + message text.
- **Recent messages**: paginated table of `WhatsAppMessageLog` (direction, phone,
  content preview, status, timestamp).
- **Notification settings**: toggle cards for each notification type with editable
  templates.

New `ApiService` methods with mock fallbacks per the existing pattern.

## 7. Feature 2 — Payslip & Notification Delivery via WhatsApp

This resolves the payroll module's open item ⚠ *"WhatsApp gateway choice"* and
implements the `whatsapp` channel of the `PayslipDeliverer` interface specified in
`payroll-module-design.md` §9.

### 7.1 WhatsApp delivery adapter (`core/payroll/delivery/whatsapp.go`)

Implements the `PayslipDeliverer` interface using the `waha.Client`:

```go
type WhatsAppDeliverer struct {
    waha *waha.Client
    db   *gorm.DB
}

func (d *WhatsAppDeliverer) Deliver(payslip Payslip, employee EmployeeProfile) error {
    // 1. Check employee.WhatsApp is not empty
    // 2. Send text message with payslip summary:
    //    "📄 Slip Gaji {month} {year}\n
    //     Nama: {name}\n
    //     Gaji Bersih: Rp {netPay}\n
    //     Detail lengkap terlampir."
    // 3. Send PDF file attachment (payslip PDF)
    // 4. Log to WhatsAppMessageLog with context=payslip, contextRefId=payslip.ID
    // 5. Record to DeliveryLog with channel=whatsapp
}
```

### 7.2 Message template

```
📄 *Slip Gaji — {monthName} {year}*

Halo {employeeName},

Berikut slip gaji Anda:
💰 Gaji Bersih: *{netPay}*

Detail lengkap dapat dilihat pada file terlampir.
Jika ada pertanyaan, hubungi HRD.

— YPA Handayani
```

### 7.3 Integration with payroll run lifecycle

In the payroll run finalize step (`POST /api/payroll/runs/{id}/finalize`):

1. Generate PDFs (existing).
2. For each payslip where the employee has a `whatsapp` field:
   - Enqueue delivery via the WhatsApp adapter.
3. Delivery is **async and failure-tolerant** — a failed WhatsApp send records
   `status=failed` in `DeliveryLog` but does not block the finalize.
4. Manager can retry via `POST /api/payroll/payslips/{id}/redeliver`.

## 8. Feature 3 — AI WhatsApp Chatbot (replaces Step 4 in AI plan)

This replaces the planned "Thin Telegram bot" (Step 4 in `ai-improvements-plan.md`)
with a WhatsApp chatbot that is more aligned with the actual customer base.

### 8.1 Flow

```
Customer sends          Go Gateway receives          FastAPI AI processes
WhatsApp message  ────▶ webhook, creates/loads  ────▶ /api/wa/chat endpoint
                        ChatbotConversation           (RAG + Gemini)
                                                          │
Customer receives  ◀──── Go sends via waha.Client  ◀──── returns reply + citations
WhatsApp reply            + logs to MessageLog
```

### 8.2 Go gateway — chatbot orchestrator (`core/whatsapp/chatbot.go`)

```go
func HandleIncomingMessage(msg WebhookMessage) {
    // 1. Ignore messages from self (paired number)
    // 2. Ignore group messages (only DMs)
    // 3. Ignore non-text messages (reply with "Maaf, saya hanya bisa memproses
    //    pesan teks. Silakan ketik pertanyaan Anda.")
    // 4. Find or create ChatbotConversation for this chatId
    //    - If conversation expired (lastMessageAt > 30min), create new one
    // 5. Store incoming message as ChatbotMessage (role=user)
    // 6. Load last N messages for context (configurable, default 10)
    // 7. Send typing indicator via waha.StartTyping()
    // 8. Call FastAPI /api/wa/chat with:
    //    - message text
    //    - conversation history [{role, content}]
    //    - phone number (for CRM matching)
    // 9. Receive reply + citations
    // 10. Store reply as ChatbotMessage (role=assistant)
    // 11. Send reply via waha.SendText()
    // 12. Stop typing indicator
    // 13. Log to WhatsAppMessageLog (context=chatbot)
}
```

### 8.3 FastAPI AI endpoint (`backend/app/routers/whatsapp_ai.py`)

New router registered in `backend/app/main.py`:

```python
router = APIRouter(prefix="/api/wa", tags=["whatsapp-ai"])

class ChatRequest(BaseModel):
    message: str
    history: list[dict]  # [{role: "user"|"assistant", content: "..."}]
    phoneNumber: str | None = None

class ChatResponse(BaseModel):
    reply: str
    citations: list[str]  # ["kursus-3", "mekanisme-5"]
    leadCaptured: bool     # True if the AI extracted name+phone for CRM

@router.post("/chat", response_model=ChatResponse)
def chat(body: ChatRequest, db: Connection = Depends(get_db)):
    # 1. Fetch RAG knowledge base chunks (reuse _build_kb from rag.py)
    # 2. Build Gemini prompt:
    #    - System: "You are YPA Handayani's WhatsApp assistant..."
    #    - Context: RAG knowledge base (with source tags)
    #    - History: conversation history
    #    - User: current message
    # 3. Call Gemini (with stub fallback, consistent with ai.py pattern)
    # 4. Parse response: extract reply text + cited source IDs
    # 5. Lead capture: if the AI detects customer intent to register,
    #    extract name/phone and return leadCaptured=True
    # 6. Return ChatResponse
```

### 8.4 Gemini prompt design

```
Anda adalah asisten virtual YPA Handayani di WhatsApp. Anda membantu calon
siswa mendapatkan informasi tentang kursus mengemudi, menjahit, komputer, dan
bahasa. Jawab dalam Bahasa Indonesia, sopan dan ringkas.

ATURAN:
1. Jawab HANYA berdasarkan informasi dalam Basis Pengetahuan di bawah.
2. Jika informasi tidak tersedia, arahkan ke kontak resmi: 082191927620.
3. Sebutkan sumber dalam format [#kursus-3] jika mengutip fakta spesifik.
4. Jika calon siswa ingin mendaftar, minta nama lengkap dan nomor HP mereka.
5. Jangan menjawab pertanyaan di luar konteks YPA Handayani.
6. Maksimal 3 paragraf per jawaban.

BASIS PENGETAHUAN:
{rag_knowledge_base}

RIWAYAT PERCAKAPAN:
{conversation_history}
```

### 8.5 Lead capture → CRM integration

When the AI detects registration intent and extracts customer data:

1. FastAPI returns `leadCaptured: true` with extracted name/phone in the response.
2. Go gateway creates a new `StudentCrm` record with `status=lead`, `notes="Captured
   via WhatsApp chatbot"`.
3. Sends a confirmation message to the customer: *"Terima kasih {nama}! Data Anda
   telah kami catat. Tim kami akan segera menghubungi Anda. 😊"*
4. Optionally notifies the manager via a dashboard notification.

### 8.6 Graceful degradation

- **GEMINI_API_KEY absent**: chatbot returns deterministic canned responses (same
  pattern as `ai.py`'s `_stub` function). Responses are selected by keyword matching
  against the RAG knowledge base.
- **WAHA down**: webhook delivery fails silently; no customer-facing error. The
  dashboard shows the disconnected status.
- **FastAPI down**: Go gateway returns a fallback message: *"Maaf, sistem kami sedang
  dalam perbaikan. Silakan hubungi kami langsung di 082191927620."*

## 9. Feature 4 — Automated Session Reminders & Schedule Notifications

### 9.1 Reminder types

| Type | Trigger | Recipient | Template |
|---|---|---|---|
| **Session reminder** | `leadTimeMinutes` before `LearningPlan.scheduledDate + startTime` | Student (via `Student.WhatsApp`) | *"Halo {nama}! 🚗 Mengingatkan jadwal kursus mengemudi Anda hari ini pukul {waktu} bersama instruktur {instruktur}. Lokasi: {meetingPoint}. Sampai jumpa!"* |
| **Daily instructor schedule** | Cron: every day at 07:00 WIB | Instructor (via `User` WhatsApp, once `EmployeeProfile.whatsapp` exists via payroll module) | *"Selamat pagi {nama}! 📋 Jadwal Anda hari ini:\n{daftar_jadwal}\nSemangat mengajar!"* |
| **Low quota alert** | After `EndStudentSession` when `remainingQuotaHours ≤ 2` | Student | *"Halo {nama}! Sisa kuota kursus Anda tinggal {sisa} jam. Hubungi kami untuk menambah kuota. 📞 082191927620"* |
| **AI upsell recommendation** | After session analysis returns `upsellRecommendation != null` | Student | *"Halo {nama}! Berdasarkan evaluasi sesi terakhir, kami merekomendasikan {rekomendasi}. Hubungi kami untuk info lebih lanjut!"* |

### 9.2 Scheduler implementation (`core/whatsapp/scheduler.go`)

Uses a goroutine-based ticker (no external scheduler dependency — keeps the Go binary
self-contained, consistent with the project's minimal-dependency philosophy).

```go
func StartScheduler(wahaClient *waha.Client, db *gorm.DB) {
    // Ticker: every 1 minute
    //   1. Query LearningPlans where:
    //      - status = 'planned'
    //      - scheduledDate + startTime - leadTimeMinutes <= now
    //      - reminderSent = false
    //   2. For each match, send reminder via waha.SendText()
    //   3. Set reminderSent = true

    // Cron-like daily job (check once per tick if time matches):
    //   - At 07:00 WIB, aggregate today's LearningPlans per instructor
    //   - Send daily schedule summary to each instructor
}
```

The `LearningPlan` model already has a `ReminderSent bool` field — this is the flag
that prevents duplicate sends.

### 9.3 Integration with session analysis (upsell notifications)

In the existing `POST /api/sessions/{id}/analyze` flow:

1. AI service returns `upsellRecommendation` (existing).
2. Go gateway (post-proxy, or via a new handler wrapper) checks if the recommendation
   is non-null.
3. Looks up the student's WhatsApp number from `StudentCrm.phone` or
   `Student.WhatsApp`.
4. Sends the upsell message via `waha.SendText()`.
5. Logs with `context=upsell`.

## 10. Seed data & defaults

New seed in `core/seed/whatsapp.go`:

```go
// Default notification schedules
var defaultNotifications = []NotificationSchedule{
    {
        NotificationType: "session_reminder",
        Enabled:          true,
        LeadTimeMinutes:  60,
        TemplateText:     "Halo {{.StudentName}}! 🚗 Mengingatkan jadwal ...",
    },
    {
        NotificationType: "daily_schedule",
        Enabled:          true,
        CronExpression:   "0 7 * * *",  // 07:00 WIB daily
        TemplateText:     "Selamat pagi {{.InstructorName}}! 📋 ...",
    },
    {
        NotificationType: "quota_low",
        Enabled:          true,
        LeadTimeMinutes:  0,  // immediate, triggered by event
        TemplateText:     "Halo {{.StudentName}}! Sisa kuota ...",
    },
    {
        NotificationType: "upsell",
        Enabled:          false,  // opt-in by manager
        LeadTimeMinutes:  0,
        TemplateText:     "Halo {{.StudentName}}! Berdasarkan ...",
    },
}
```

## 11. API surface summary

### 11.1 Go gateway — new endpoints

| Method | Path | Auth | Purpose |
|---|---|---|---|
| GET | `/api/admin/whatsapp/status` | manager | WAHA session status |
| POST | `/api/admin/whatsapp/start` | manager | Start WAHA session |
| POST | `/api/admin/whatsapp/stop` | manager | Stop WAHA session |
| POST | `/api/admin/whatsapp/restart` | manager | Restart WAHA session |
| POST | `/api/admin/whatsapp/logout` | manager | Logout (unpair) |
| GET | `/api/admin/whatsapp/qr` | manager | QR code image for pairing |
| POST | `/api/admin/whatsapp/send-test` | manager | Send test message |
| GET | `/api/admin/whatsapp/messages` | manager | Message log (paginated) |
| GET | `/api/admin/whatsapp/conversations` | manager | List chatbot conversations |
| GET | `/api/admin/whatsapp/conversations/:id` | manager | Conversation detail |
| GET | `/api/admin/whatsapp/notifications` | manager | Notification settings |
| PUT | `/api/admin/whatsapp/notifications/:id` | manager | Update notification setting |
| POST | `/api/webhooks/whatsapp` | public (HMAC) | WAHA webhook receiver |

### 11.2 FastAPI AI service — new endpoint

| Method | Path | Auth | Purpose |
|---|---|---|---|
| POST | `/api/wa/chat` | internal (Go→FastAPI, token forwarded) | AI chatbot reply generation |

## 12. Frontend (Angular)

### 12.1 New dashboard route

`/dashboard/whatsapp` — lazy-loaded, manager-only, guarded by `authGuard` +
role check.

### 12.2 Components

| Component | Description |
|---|---|
| `WhatsappDashboardComponent` | Main page shell with tab navigation |
| `WaConnectionCardComponent` | Status card + QR pairing + action buttons |
| `WaMessageLogComponent` | Paginated message log table with filters |
| `WaConversationsComponent` | List of chatbot conversations with preview |
| `WaConversationDetailComponent` | Full message thread view (chat bubble UI) |
| `WaNotificationSettingsComponent` | Toggle cards + template editor for each type |
| `WaSendTestComponent` | Quick send form (phone + message) |

### 12.3 New models (`frontend/src/app/core/models/whatsapp.model.ts`)

```typescript
export interface WhatsAppStatus {
  sessionName: string;
  status: 'STOPPED' | 'STARTING' | 'SCAN_QR_CODE' | 'WORKING' | 'FAILED';
  phoneNumber: string;
  pairedAt: string | null;
  lastSyncedAt: string;
}

export interface WhatsAppMessage {
  id: number;
  direction: 'outbound' | 'inbound';
  phoneNumber: string;
  messageType: string;
  content: string;
  status: 'pending' | 'sent' | 'delivered' | 'read' | 'failed';
  context: string;
  createdAt: string;
}

export interface ChatbotConversation {
  id: number;
  phoneNumber: string;
  studentName: string | null;
  status: 'active' | 'expired' | 'escalated';
  messageCount: number;
  lastMessageAt: string;
}

export interface NotificationSchedule {
  id: number;
  notificationType: string;
  enabled: boolean;
  leadTimeMinutes: number;
  cronExpression: string | null;
  templateText: string;
}
```

### 12.4 ApiService additions

```typescript
// WhatsApp management (manager-only)
getWhatsAppStatus(): Observable<WhatsAppStatus>
startWhatsApp(): Observable<any>
stopWhatsApp(): Observable<any>
restartWhatsApp(): Observable<any>
logoutWhatsApp(): Observable<any>
getWhatsAppQR(): Observable<Blob>
sendTestMessage(phone: string, text: string): Observable<any>
getMessageLog(page: number, size: number): Observable<WhatsAppMessage[]>
getConversations(): Observable<ChatbotConversation[]>
getConversation(id: number): Observable<ChatbotMessage[]>
getNotificationSettings(): Observable<NotificationSchedule[]>
updateNotificationSetting(setting: NotificationSchedule): Observable<NotificationSchedule>
```

All with mock fallbacks per the existing pattern.

### 12.5 i18n strings

New keys added to `frontend/public/i18n/{id,en}.json`:

```json
{
  "whatsapp.title": "WhatsApp",
  "whatsapp.status": "Status Koneksi",
  "whatsapp.connected": "Terhubung",
  "whatsapp.disconnected": "Terputus",
  "whatsapp.scanning": "Menunggu Scan QR",
  "whatsapp.scan_qr": "Scan QR Code",
  "whatsapp.messages": "Riwayat Pesan",
  "whatsapp.conversations": "Percakapan Chatbot",
  "whatsapp.notifications": "Pengaturan Notifikasi",
  "whatsapp.send_test": "Kirim Pesan Tes",
  "whatsapp.session_reminder": "Pengingat Jadwal",
  "whatsapp.daily_schedule": "Jadwal Harian Instruktur",
  "whatsapp.quota_low": "Peringatan Kuota Rendah",
  "whatsapp.upsell": "Rekomendasi Kursus Tambahan"
}
```

## 13. Environment variables summary

| Variable | Service | Default | Required | Notes |
|---|---|---|---|---|
| `WAHA_URL` | core | `http://localhost:3000` | No | WAHA base URL |
| `WAHA_API_KEY` | core, waha | _(empty)_ | Recommended | Shared API key |
| `WAHA_HMAC_KEY` | core, waha | `waha-hmac-secret` | Yes (production) | Webhook signature validation |
| `WAHA_SESSION_NAME` | core | `default` | No | WAHA session identifier |
| `WAHA_DASHBOARD_USERNAME` | waha | `admin` | No | WAHA's built-in dashboard |
| `WAHA_DASHBOARD_PASSWORD` | waha | `admin` | No | WAHA's built-in dashboard |

## 14. Error handling & edge cases

- **WAHA not running**: all send operations fail gracefully; logged as `failed` in
  `WhatsAppMessageLog`; payroll finalize continues; chatbot webhook never fires.
- **Phone number not set**: skip delivery; surface in UI as "No WhatsApp number".
- **Student blocked the number**: WAHA returns error; logged; no retry (the student
  opted out).
- **Rate limiting**: WAHA/WhatsApp may throttle high-volume sends. The scheduler spaces
  messages with a configurable delay (default: 1 message per 2 seconds) to avoid
  triggering anti-spam.
- **Duplicate messages**: webhook deduplication by `wahaMessageId` — if the same
  `wahaMessageId` arrives twice, the second is ignored.
- **Conversation expiry**: conversations inactive for >30 minutes are marked `expired`;
  next message starts a fresh conversation (resets Gemini context).
- **Escalation**: if the chatbot can't answer after 2 "I don't know" responses in one
  conversation, it replies with the official phone number and marks the conversation
  `escalated`.
- **Media messages**: inbound images/audio/video are not processed (MVP); bot replies
  with a text-only notice.

## 15. Cross-references to existing planned work

| This document | References | Impact |
|---|---|---|
| Feature 2 (payslip delivery) | `payroll-module-design.md` §9, §11 ⚠ | **Resolves** the WhatsApp gateway open item |
| Feature 3 (chatbot) | `ai-improvements-plan.md` Step 4 | **Replaces** Telegram bot with WhatsApp bot |
| Feature 3 (lead capture) | `prd.md` §4.A (AI Chatbot) | **Implements** chatbot lead capture → CRM |
| Feature 4 (notifications) | `prd.md` §6.3 (n8n automation) | **Replaces** n8n with native Go scheduler |
| Feature 3 (chatbot AI) | `ai-improvements-plan.md` Step 1 | **Depends on** RAG citations (completed ✅) |
| Feature 3 (chatbot AI) | `ai-improvements-plan.md` Step 2 | **Benefits from** Langfuse tracing (optional) |

## 16. Testing

- **WAHA client unit tests** (`core/waha/client_test.go`): mock HTTP server, verify
  request/response marshalling for each method.
- **Webhook handler tests** (`core/handlers/whatsapp_test.go`): HMAC validation (valid,
  invalid, missing), event dispatch, deduplication.
- **Chatbot orchestration tests** (`core/whatsapp/chatbot_test.go`): conversation
  creation, expiry, escalation logic, CRM lead creation.
- **Scheduler tests** (`core/whatsapp/scheduler_test.go`): reminder timing, duplicate
  prevention, daily schedule aggregation.
- **FastAPI chatbot tests** (`backend/tests/test_whatsapp_ai.py`): stub Gemini
  responses, verify prompt construction, citation extraction.
- **Integration test**: end-to-end with WAHA running in Docker — send test message,
  verify receipt in message log.

## 17. Phasing (each phase ships and demos independently)

1. **WAHA setup + client + admin dashboard** — compose service, `waha/` client package,
   session management endpoints, webhook receiver skeleton, frontend WhatsApp page with
   QR pairing + status. (≈2–3 days) · **Status: Planned** 🔲
2. **Message logging + send capability** — `WhatsAppMessageLog` model, send-test
   endpoint, message log UI, graceful degradation. (≈1–2 days) · **Status: Planned** 🔲
3. **AI chatbot** — `ChatbotConversation` + `ChatbotMessage` models, chatbot
   orchestrator, FastAPI `/api/wa/chat` endpoint, conversation UI, lead capture → CRM.
   (≈3–4 days) · **Status: Planned** 🔲
4. **Payslip WhatsApp delivery** — WhatsApp delivery adapter for payroll, integration
   with run finalize, redeliver support. (≈1–2 days, **gated on payroll module Phase 5**)
   · **Status: Planned** 🔲
5. **Automated notifications** — scheduler, session reminders, daily instructor
   schedules, low-quota alerts, upsell notifications, notification settings UI. (≈2–3
   days) · **Status: Planned** 🔲

Total estimated effort: **9–14 days**.

## 18. Open items & research tasks

- **⚠#1 WAHA engine choice** — WAHA supports `WEBJS`, `NOWEB`, and `VENOM` engines.
  `WEBJS` is the most stable but resource-heavy. Evaluate `NOWEB` for lighter footprint
  if memory is constrained on the VPS.
- **⚠#2 WhatsApp Terms of Service** — using the Web protocol for automated messaging
  carries a risk of account ban. Rate-limit sends, avoid bulk marketing, and keep the
  chatbot responses helpful and non-spammy. Consider Meta's Cloud API for production if
  volume grows.
- **⚠#3 Instructor WhatsApp numbers** — the `User` model currently has no `whatsapp`
  or `email` field. The payroll module's `EmployeeProfile` adds these. Until payroll
  Phase 1 ships, daily schedule notifications to instructors are deferred (or use a
  temporary field on `User`).
- **⚠#4 Web chatbot widget upgrade** — the current `ChatBotComponent` returns random
  canned responses. It could be upgraded to call the same FastAPI `/api/wa/chat`
  endpoint (via the Go proxy) so both WhatsApp and web chatbots share the same AI brain.
  This is a natural follow-up but not in scope for this spec.
