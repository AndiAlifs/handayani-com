# WhatsApp Integration via WAHA — Design Spec

**Date:** 2026-06-21 (rev. 2026-06-22) · **Status:** Draft, pre-approval · **Owner:** Andi Alifsyah

> **Revision 2026-06-22:** pinned to a **single-machine** deployment — WAHA + MySQL run
> as Docker containers, the Go/FastAPI/Angular apps run as host processes. All
> service-to-service URLs in this doc reflect that split (see §3).

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

**Deployment topology (this machine).** WAHA and MySQL run as Docker containers; the
Go gateway, FastAPI AI service, and Angular dev server run as **bare host processes**
(`go run` / `uvicorn` / `ng serve`) — the same split the existing `core/.env.example`
already assumes (`DB_HOST=127.0.0.1`). This single fact drives every URL below:

- **Go → WAHA** uses `http://localhost:3000` — the host reaches WAHA's *published*
  container port. NOT the compose service name `http://waha:3000`, which only resolves
  *inside* the Docker network.
- **WAHA → Go** (webhook) uses `http://host.docker.internal:8080` — the container
  reaches the host-side gateway. NOT `http://core:8080`; there is no `core` container in
  this topology.

| Concern | Decision | Rationale |
|---|---|---|
| WAHA deployment | Docker container; apps run on the host | Same machine: WAHA + MySQL are containers, the Go/FastAPI/Angular apps are host processes |
| Go → WAHA URL | `http://localhost:3000` (`WAHA_URL`) | Host process → WAHA's published container port |
| WAHA API key | Shared via env var `WAHA_API_KEY` | WAHA supports `api_key` security |
| Go ↔ WAHA auth | `X-Api-Key` header on every request | Standard WAHA auth scheme |
| Webhook delivery | WAHA → `http://host.docker.internal:8080/api/webhooks/whatsapp` | Container → host; Go must listen on `0.0.0.0:8080` (Gin default) so the container can reach it; HMAC-validated |
| Chatbot AI | Go gateway proxies to FastAPI `/api/wa/chat` via `AI_SERVICE_URL` (host: `http://localhost:8081`) | Keeps AI in Python, consistent with analyze |
| Phone format | Indonesian international format (`628...`) | Matches existing `normalizeWhatsApp()` in `handlers/instructor.go` |

## 4. Data model

All tables are **owned by the Go gateway and added to `AutoMigrate`** (verified: the
gateway already AutoMigrates `User`, `Student`, `LearningPlan`, … in `core/main.go`; the
`knowledge.go` content tables are the only ones it deliberately does NOT migrate).

> **Casing note (corrected after verification).** The wire format is *not* uniformly
> camelCase. The `knowledge.go` content models (`Course`, `Mechanism`, `StudentCrm`,
> `Session`) and the Angular models are camelCase — but the `instructor.go` / `models.go`
> models (`Student`, `LearningPlan`, `Attendance`, `User`) serialize as **snake_case**
> (`remaining_quota_hours`, `reminder_sent`, `scheduled_date`). The new WhatsApp tables
> below use **camelCase** to match the Angular models in §12.3, so do NOT copy the
> snake_case `instructor.go` tag style (full structs in §21). When a feature *reads*
> `Student` / `LearningPlan`, use their existing snake_case field names (`whatsapp`,
> `remaining_quota_hours`, `reminder_sent`).

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
    BaseURL    string       // host process → published port, e.g. "http://localhost:3000"
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

New `waha` service added to `compose.yml`. In this topology the app services (`core`,
`ai`, `web`) are **not** run from compose — they run on the host — so you bring up only
the infra containers:

```sh
docker compose up db waha      # MySQL + WAHA in Docker; run core/ai/web on the host
```

The service definition:

```yaml
waha:
  image: devlikeapro/waha:latest
  environment:
    WHATSAPP_DEFAULT_ENGINE: WEBJS
    WAHA_DASHBOARD_ENABLED: "true"
    WAHA_DASHBOARD_USERNAME: ${WAHA_DASHBOARD_USERNAME:-admin}
    WAHA_DASHBOARD_PASSWORD: ${WAHA_DASHBOARD_PASSWORD:-admin}
    # Per WAHA docs the primary env var is WAHA_API_KEY (WHATSAPP_API_KEY is a legacy
    # alias). Accepts a hashed form too: WAHA_API_KEY=sha512:<hash>.
    WAHA_API_KEY: ${WAHA_API_KEY:-}
    # Container → host: the Go gateway runs on the host, not in compose.
    WHATSAPP_HOOK_URL: http://host.docker.internal:8080/api/webhooks/whatsapp
    WHATSAPP_HOOK_EVENTS: "message,message.ack,session.status"
    WHATSAPP_HOOK_HMAC_KEY: ${WAHA_HMAC_KEY:-waha-hmac-secret}
  ports:
    # Required (not just for the dashboard): the host-side Go gateway calls WAHA's
    # REST API on this published port via WAHA_URL=http://localhost:3000.
    - "3000:3000"
  volumes:
    - waha-data:/app/.sessions
  # Auto-provided by Docker Desktop (Windows/macOS); needed explicitly on native Linux.
  extra_hosts:
    - "host.docker.internal:host-gateway"
```

New volume: `waha-data` for persistent session data. (No `depends_on: db` — WAHA keeps
its session state in `waha-data` and has no dependency on MySQL.)

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

> **Data sourcing (corrected).** `LearningPlan` (`core/models/instructor.go`) has **no
> `meetingPoint` and no instructor relation** — only `InstructorID`, `StudentID`, and a
> `Student` relation. So the session-reminder template must resolve `{meetingPoint}` from
> **`Student.MeetingPoint`** and `{instruktur}` by looking up **`User.FullName` via
> `LearningPlan.InstructorID`**; `{nama}`/`{waktu}` come from the plan + its `Student`.
> The reminder reads `Student.WhatsApp` (field `whatsapp`); quota alerts read
> `Student.RemainingQuotaHours` (`remaining_quota_hours`).

> **⚠ Proactive-messaging ban risk — read before building Feature 4.** WAHA's official
> "How to Avoid Blocking" guidance is **reactive-only**: *"never initiate a conversation …
> only reply to messages you receive."* Every notification in this table is
> **proactive/unsolicited** — exactly the highest-ban-risk pattern on the unofficial Web
> protocol. This doesn't kill the feature, but it sets the rules: opt-in per type,
> transactional framing, 30–60 s random spacing, and per-contact caps (§20.3), with a
> migration path to the WhatsApp Cloud API if volume grows. The chatbot (Feature 3) is
> reactive and low-risk.

### 9.2 Scheduler implementation (`core/whatsapp/scheduler.go`)

Uses a goroutine-based ticker (no external scheduler dependency — keeps the Go binary
self-contained, consistent with the project's minimal-dependency philosophy).

> **This is net-new infrastructure (verified).** There is currently **no background
> goroutine, ticker, or cron anywhere in `core/`** — the gateway is purely synchronous
> request handling plus a one-time startup migrate+seed. This scheduler is the first
> long-running worker, so build it deliberately (§20.4): start it once from `main()`
> after `seed.RunAll()`, run it in a **single process only** (the `ReminderSent` flag is
> not safe against two concurrent schedulers without row-level locking), pin the clock to
> **`Asia/Jakarta` (WIB)** explicitly rather than trusting the host TZ, and tolerate
> restarts (a window that elapsed while the process was down should still fire on the next
> tick, not be silently skipped).

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

The `LearningPlan` model already has a `ReminderSent bool` field (`json:"reminder_sent"`,
verified at `core/models/instructor.go`) — this is the flag that prevents duplicate
sends. **It is currently unused** (nothing reads or writes it today), so the scheduler
owns its full lifecycle. To kill the double-send race, flip it inside the same write that
records the send and gate on the affected-row count: `UPDATE learning_plans SET
reminder_sent = true WHERE id = ? AND reminder_sent = false` — only send when 1 row was
updated.

### 9.3 Integration with session analysis (upsell notifications)

**Implementation reality (corrected).** Today `POST /api/sessions/{id}/analyze` is a
**transparent reverse-proxy** — `sessions.POST("/:id/analyze", handlers.ProxyToAI)` in
`core/main.go` streams the Python response straight back, so the Go gateway never sees the
body and cannot "post-proxy inspect" anything. Two designs:

- **(A) Wrapping handler (recommended).** Replace the transparent proxy on this one route
  with a handler that calls the AI service itself (reusing `AI_SERVICE_URL`), reads the
  JSON, streams it back unchanged, and — only on success — fires the upsell asynchronously.
  AI stays in Python; the side-effect lives in Go.
- **(B) Python callback.** FastAPI calls back into the gateway after persisting. Rejected:
  adds a Python→Go dependency and a second auth hop for no real gain.

Flow for (A), after a successful analyze returning a non-null `upsellRecommendation`:

1. Resolve the student's WhatsApp number. `Session.studentId` is an int → map to a
   `Student` (which has `WhatsApp`); `StudentCrm` has `phone` but no link to `Session`, so
   `Student.WhatsApp` is the reliable source.
2. Check the `upsell` `NotificationSchedule` is `enabled` (ships **disabled** by default,
   §10) — if not, skip.
3. Send via `waha.SendText()`, through the §20.3 rate limiter.
4. Log with `context=upsell`, `contextRefId=session.ID`.

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

`/dashboard/whatsapp` — lazy-loaded, manager-only.

> **No role guard exists yet (verified).** The frontend has only `authGuard`
> (authenticated-or-not, `core/guards/auth.guard.ts`); there is no manager-only
> `CanActivateFn`. Role-gating today is *UI-only* — `DashboardLayoutComponent` filters
> nav items via `NavItem.roles` + `visibleNavItems`, and the backend enforces manager-only
> with `ManagerMiddleware`. For this sensitive page add a small `managerGuard`
> (`inject(AuthService).isManager()` else redirect) and stack it:
> `canActivate: [authGuard, managerGuard]`. The `/api/admin/...` endpoints stay the
> authoritative gate regardless. Note also: dashboard nav labels are **hardcoded strings**
> today, and `AuthService` already exposes `isManager()`.

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
| `WAHA_URL` | core | `http://localhost:3000` | No | Go (host) → WAHA's published container port. NOT `http://waha:3000` (compose-DNS, resolves only inside Docker) |
| `WAHA_API_KEY` | core, waha | _(empty)_ | Recommended | Shared API key |
| `WAHA_HMAC_KEY` | core, waha | `waha-hmac-secret` | Yes (production) | Webhook signature validation |
| `WAHA_SESSION_NAME` | core | `default` | No | WAHA session identifier |
| `WHATSAPP_HOOK_URL` | waha | `http://host.docker.internal:8080/api/webhooks/whatsapp` | Yes | WAHA (container) → Go gateway (host). NOT `http://core:8080` |
| `WAHA_DASHBOARD_USERNAME` | waha | `admin` | No | WAHA's built-in dashboard |
| `WAHA_DASHBOARD_PASSWORD` | waha | `admin` | No | WAHA's built-in dashboard |

## 14. Error handling & edge cases

- **WAHA not running**: all send operations fail gracefully; logged as `failed` in
  `WhatsAppMessageLog`; payroll finalize continues; chatbot webhook never fires.
- **Phone number not set**: skip delivery; surface in UI as "No WhatsApp number".
- **Student blocked the number**: WAHA returns error; logged; no retry (the student
  opted out).
- **Rate limiting (corrected to WAHA's own guidance)**: 1 message / 2 s is **far too
  aggressive** for the unofficial Web protocol. WAHA's "How to Avoid Blocking" prescribes:
  be **reactive-only** where possible, wait a **random 30–60 s between messages** (not a
  fixed interval), cap at **~4 messages per contact per hour**, take breaks between
  batches, and never send 24/7 — being flagged as spam 5–10 times can ban the number. The
  sender must implement a randomized 30–60 s queue + per-contact hourly cap (§20.3), not a
  tight 2 s loop.
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

## 18. Decisions on open items

> Resolved against the WAHA docs + the current codebase on 2026-06-22. #5 is the
> same-machine networking caveat from the revision note; the rest are now decisions.

- **#1 WAHA engine → `WEBJS` (default); revisit `NOWEB` only if RAM-bound.** WAHA ships
  **four** engines (`WEBJS`, `WPP`, `NOWEB`, `GOWS`) — the old `VENOM` is gone. `WEBJS` is
  the default and most feature-complete but runs headless Chromium (heaviest; budget
  ~1 GB for the container). `NOWEB`/`GOWS` are browserless and far lighter, but `NOWEB`
  needs a Store enabled for chat/contact history and `GOWS` is newest/least proven.
  **Decision:** start on `WEBJS` for stability on this single dedicated machine; switch to
  `NOWEB` only under memory pressure — and re-test if you do, because **webhook and
  send-response shapes differ by engine** (§19.4).
- **#2 ToS / ban risk → reactive chatbot ships; proactive sends are opt-in + rate-limited
  + flagged for Cloud-API migration.** The single biggest external risk. The chatbot
  (Feature 3) is reactive and low-risk. The proactive notifications (Feature 4) and
  payslip/upsell sends violate WAHA's "only reply, never initiate" guidance, so they ship
  **off by default**, gated per type via `NotificationSchedule`, sent transactionally with
  30–60 s random spacing and per-contact caps (§20.3). If proactive volume grows, migrate
  those flows to the **WhatsApp Cloud API** and keep WAHA for the reactive chatbot.
- **#3 Instructor WhatsApp numbers → add a nullable `WhatsApp` to `User` now.** Verified:
  `User` (`core/models/models.go`) has **no `whatsapp`, `phone`, or `email` field at all.**
  Rather than block Feature 4's daily-instructor-schedule on payroll's `EmployeeProfile`,
  add one nullable `WhatsApp string` column to `User` in this work (AutoMigrate-friendly;
  reconcile with `EmployeeProfile.whatsapp` when payroll lands). Until populated, the
  daily-schedule job skips numberless instructors and logs it.
- **#4 Web chatbot widget upgrade → confirmed follow-up, out of scope.** Upgrade the
  static `ChatBotComponent` to call `/api/wa/chat` (via the Go proxy) so web and WhatsApp
  share one AI brain. Natural next step after Feature 3; not in this spec.
- **#5 Same-machine networking (Windows host)** — with WAHA in Docker and the Go gateway
  on the host, the webhook crosses the container→host boundary via `host.docker.internal`.
  Three things must hold: (1) the Go gateway listens on `0.0.0.0:8080` (Gin's default
  `r.Run(":" + port)` does — do **not** bind `127.0.0.1` only, or the container can't reach
  it; verified at `core/main.go`); (2) Windows Firewall may prompt to allow the inbound
  connection to the Go process the first time WAHA posts a webhook — allow it; (3) the WAHA
  container needs working outbound DNS/internet to reach WhatsApp's servers — this machine's
  Docker DNS is already pinned to the router in `daemon.json` (public UDP:53 is dropped on
  this network), so keep that fix in place.

---

## 19. Verified WAHA API contract (implementation reference)

Sourced from the WAHA docs (`waha.devlike.pro`) on 2026-06-22. WAHA uses **date-based
versioning**; the granular session API below landed in **2024.9**. Pin your image
(`devlikeapro/waha:<date>` rather than `:latest`) and validate against that version's
Swagger at `http://localhost:3000/swagger`.

### 19.1 Webhook envelope + payload structs

Every webhook is a `POST` with a common envelope; decode `payload` per `event`. Note the
**timestamp scale trap**: the envelope `timestamp` is **milliseconds**, but the `message`
payload `timestamp` is **seconds**.

```go
type WebhookEnvelope struct {
    ID        string          `json:"id"`        // ULID, unique per event
    Timestamp int64           `json:"timestamp"` // MILLISECONDS
    Event     string          `json:"event"`     // message | message.ack | session.status
    Session   string          `json:"session"`
    Me        *MeInfo         `json:"me"`         // paired account
    Engine    string          `json:"engine"`     // WEBJS | NOWEB | ...
    Payload   json.RawMessage `json:"payload"`    // decode by Event
}

// event = "message"
type MessagePayload struct {
    ID        string `json:"id"`
    Timestamp int64  `json:"timestamp"` // SECONDS (≠ envelope ms)
    From      string `json:"from"`      // "628xxx@c.us" (DM) | "...@g.us" (group)
    To        string `json:"to"`
    FromMe    bool   `json:"fromMe"`    // true ⇒ sent by us — ignore
    Body      string `json:"body"`
    HasMedia  bool   `json:"hasMedia"`
    Source    string `json:"source"`    // "app" | "api"
    // notifyName/pushName are engine-dependent and live under _data — don't rely on a
    // guaranteed top-level field; fall back to me.pushName.
}

// event = "message.ack"
type AckPayload struct {
    ID      string `json:"id"`
    From    string `json:"from"`
    FromMe  bool   `json:"fromMe"`
    Ack     int    `json:"ack"`     // -1..4
    AckName string `json:"ackName"` // ERROR|PENDING|SERVER|DEVICE|READ|PLAYED
}

// event = "session.status"
type SessionStatusPayload struct {
    Status string `json:"status"` // STOPPED|STARTING|SCAN_QR_CODE|WORKING|FAILED
}
```

**Routing rules** (for `chatbot.go`, §8.2): ignore `payload.FromMe == true` (self); ignore
chat IDs ending `@g.us` (groups — only `@c.us` DMs); non-text → canned "text only" reply.

**`message.ack` → `WhatsAppMessageLog.status` mapping:**

| ack | ackName | log status |
|----|---------|-----------|
| -1 | ERROR | `failed` |
| 0 | PENDING | `pending` |
| 1 | SERVER | `sent` |
| 2 | DEVICE | `delivered` |
| 3 | READ | `read` |
| 4 | PLAYED | `read` |

### 19.2 Webhook HMAC verification (Core feature — free)

WAHA signs each webhook with **HMAC-SHA512 over the raw request body**, hex-encoded
(lowercase), in header **`X-Webhook-Hmac`** (algorithm echoed in
`X-Webhook-Hmac-Algorithm: sha512`). The key is `WHATSAPP_HOOK_HMAC_KEY`. WAHA also sends
`X-Webhook-Request-Id` and `X-Webhook-Timestamp` (ms).

**Critical:** hash the **raw bytes**, before Gin binds JSON. In a Gin handler, read then
restore the body:

```go
func verifyHMAC(c *gin.Context, key string) ([]byte, bool) {
    raw, _ := io.ReadAll(c.Request.Body)
    c.Request.Body = io.NopCloser(bytes.NewReader(raw)) // restore for later binding
    mac := hmac.New(sha512.New, []byte(key))
    mac.Write(raw)
    expected := hex.EncodeToString(mac.Sum(nil))
    got := c.GetHeader("X-Webhook-Hmac")
    return raw, hmac.Equal([]byte(expected), []byte(got)) // constant-time
}
```

**Test vector from the docs** (use it in `whatsapp_test.go`):
body `{"event":"message","session":"default","engine":"WEBJS"}`, key `my-secret-key` →
`X-Webhook-Hmac` = `208f8a55dde9e05519e898b10b89bf0d0b3b0fdf11fdbf09b6b90476301b98d8097c462b2b17a6ce93b6b47a136cf2e78a33a63f6752c2c1631777076153fa89`.

If `WAHA_HMAC_KEY` is empty (dev), skip verification but log a warning — never silently
accept unsigned webhooks in production.

### 19.3 Endpoint paths (note the convention split)

Session **management** moved under `/api/sessions/{session}/...` in 2024.9; **auth/QR** and
**sending** kept the older `/api/{session}/...` or "session-in-body" forms. This split is a
real source of bugs — don't assume one prefix.

| Group | Method · Path | Body / notes |
|---|---|---|
| List sessions | `GET /api/sessions` | `?all=true` includes stopped |
| Create | `POST /api/sessions` | `{ name, config }` |
| Get info | `GET /api/sessions/{session}` | status + me |
| Get me | `GET /api/sessions/{session}/me` | paired number |
| Start / Stop / Restart / Logout | `POST /api/sessions/{session}/{start\|stop\|restart\|logout}` | |
| Delete | `DELETE /api/sessions/{session}` | |
| QR (image) | `GET /api/{session}/auth/qr?format=image` | `format=raw` → `{value}`; `Accept: application/json` → `{mimetype,data(base64)}` |
| Pairing code | `POST /api/{session}/auth/request-code` | `{ phoneNumber }`; not always available — keep QR fallback |
| Send text | `POST /api/sendText` | `{ session, chatId, text }` |
| Send image/file/voice | `POST /api/send{Image\|File\|Voice}` | `{ session, chatId, file:{mimetype,filename,url\|data}, caption? }` |
| Typing / seen | `POST /api/{startTyping\|stopTyping\|sendSeen}` | `{ session, chatId }` |

Auth on every call: header `X-Api-Key: <WAHA_API_KEY>`.

### 19.4 Send response is engine-dependent — normalize it

There is **no single normalized send response**; the shape depends on the engine, and WAHA
explicitly warns to re-test on engine change:

- **WEBJS / WPP**: id is nested → `payload.id._serialized`.
- **NOWEB / GOWS**: Baileys-style → `payload.key.id`.

The `waha.Client` send methods must normalize defensively rather than hardcode one path:

```go
// pick whichever the active engine populated
func messageID(r SendResult) string {
    if r.ID.Serialized != "" { return r.ID.Serialized } // WEBJS/WPP
    return r.Key.ID                                       // NOWEB/GOWS
}
```

Store the result in `WhatsAppMessageLog.wahaMessageId` for later `message.ack` correlation.

---

## 20. Security & hardening

### 20.1 The public webhook is the main attack surface

`POST /api/webhooks/whatsapp` has **no auth middleware** — it must defend itself:

1. **HMAC first** (§19.2), constant-time, before any parsing/DB work; reject 401 on
   mismatch. Reject if the header is absent in production.
2. **Body-size cap** — wrap with `http.MaxBytesReader` (e.g. 1 MB) so a malicious large
   body can't exhaust memory before HMAC runs.
3. **Replay window** — reject events whose `X-Webhook-Timestamp` is older than a few
   minutes; combined with HMAC this blocks captured-payload replay.
4. **Idempotency / dedup** — a unique index on `WhatsAppMessageLog.wahaMessageId` (§21)
   makes duplicate `message`/`message.ack` deliveries no-ops (insert-or-ignore). WAHA can
   redeliver, so this is required, not optional.
5. **Fast ACK** — return `200` quickly and do chatbot/Gemini work asynchronously; WAHA
   retries on slow/failed webhooks, which would otherwise duplicate AI calls.

### 20.2 Lead capture is unauthenticated user input → CRM (treat as hostile)

Feature 3 turns arbitrary WhatsApp text into a `StudentCrm` row (§8.5). Risks: CRM spam,
junk/garbage records, and stored-content injection (a crafted "name" later rendered in the
dashboard). Mitigations:

- **Validate before insert** — phone must `normalizeWhatsApp()` to a plausible `628…`;
  cap name length; strip control chars; never trust an AI-"extracted" field verbatim.
- **Rate-limit lead creation per `chatId`** — at most one new `StudentCrm` per number per
  conversation; subsequent intents update the existing row, don't multiply it.
- **Mark provenance** — `status=lead`, `notes="Captured via WhatsApp chatbot"` so managers
  can triage; consider a `pending_review` substate rather than dropping straight into the
  active funnel.
- **Frontend is the renderer** — Angular escapes by default; keep it that way (no
  `[innerHTML]` on chatbot-sourced fields).

### 20.3 Proactive messaging & the rate limiter (ban-avoidance)

Per #2 and §9, all proactive sends share **one serialized sender** with WAHA-compliant
pacing — not the original "1 msg / 2 s":

- A single outbound queue (buffered channel) drained by one goroutine; **random 30–60 s**
  gap between sends (jittered, not fixed).
- **Per-contact cap** ≈ 4 messages/hour (track in-memory or off `WhatsAppMessageLog`).
- **Transactional only** — reminders/payslips/quota alerts are service messages, not
  marketing; the `upsell` type ships disabled and is the most ban-prone — keep it opt-in.
- **Skip on disconnect** — if `WhatsAppSession.status != WORKING`, don't even enqueue;
  log `failed` with reason `session_not_connected`.

### 20.4 Scheduler concurrency, time, and restarts

- **Single instance.** The ticker runs in exactly one process. If the gateway is ever
  scaled out, only one may run the scheduler (env flag), else duplicate sends. The
  `ReminderSent` compare-and-set (§9.2) is the last line of defence.
- **Timezone.** Resolve `time.LoadLocation("Asia/Jakarta")` once; compute "07:00 WIB" and
  all `scheduledDate + startTime` math in that location. Do not rely on the host TZ (the
  container is irrelevant here — the scheduler runs in the host Go process).
- **Catch-up, not skip.** Query is `status='planned' AND reminder_sent=false AND
  (scheduledDate+startTime - leadTime) <= now` — a window missed during downtime still
  matches on the next tick. Guard against firing reminders for plans already long past
  (e.g. only if `scheduledDate+startTime >= now - grace`).
- **Daily-schedule idempotency.** The 07:00 job can fire on multiple ticks within the
  minute; record a per-day-per-instructor sent marker (or a dedicated flag) so it sends
  once.

### 20.5 Message delivery: status lifecycle & retry

`WhatsAppMessageLog.status`: `pending → sent → delivered → read`, or `→ failed`. `sent` is
set on a successful `POST /api/sendText`; `delivered`/`read` come from `message.ack`
webhooks (§19.1). `failed` (send error, or ack `ERROR`) carries `errorDetail`. Retry is
**bounded and manual-friendly**: payslip/notification sends expose a redeliver action
(§7.3); the chatbot does **not** retry (a missed reply is better than a duplicate). No
infinite auto-retry — it compounds ban risk.

### 20.6 Secrets

`WAHA_API_KEY` and `WAHA_HMAC_KEY` live in `core/.env` (host) and the WAHA container env;
both are read via `os.Getenv` (the project has no secrets manager). Keep them out of git
(`.env` is already gitignored), set non-empty values in production, and prefer the hashed
`WAHA_API_KEY=sha512:<hash>` form WAHA supports.

---

## 21. Data model — copy-paste GORM structs

Go-owned tables, added to the `AutoMigrate` list in `core/main.go`. **camelCase JSON**
(matching `knowledge.go` + the Angular models), idiomatic GORM (columns auto-derive to
snake_case; only indexes/constraints are tagged). Place in `core/models/whatsapp.go`.

```go
package models

import "time"

type WhatsAppSession struct {
    ID           uint       `gorm:"primaryKey" json:"id"`
    SessionName  string     `gorm:"uniqueIndex;size:64" json:"sessionName"`
    Status       string     `gorm:"size:16" json:"status"` // STOPPED|STARTING|SCAN_QR_CODE|WORKING|FAILED
    PhoneNumber  string     `gorm:"size:24" json:"phoneNumber"`
    PairedAt     *time.Time `json:"pairedAt"`
    LastSyncedAt time.Time  `json:"lastSyncedAt"`
}

type WhatsAppMessageLog struct {
    ID            uint      `gorm:"primaryKey" json:"id"`
    Direction     string    `gorm:"size:8;index" json:"direction"`            // outbound|inbound
    ChatID        string    `gorm:"size:64;index" json:"chatId"`              // 628xxx@c.us
    PhoneNumber   string    `gorm:"size:24" json:"phoneNumber"`
    MessageType   string    `gorm:"size:16" json:"messageType"`               // text|image|file|voice|...
    Content       string    `gorm:"type:text" json:"content"`
    MediaURL      *string   `gorm:"size:512" json:"mediaUrl"`
    WahaMessageID *string   `gorm:"uniqueIndex;size:128" json:"wahaMessageId"` // dedup + ack correlation
    Status        string    `gorm:"size:12;index" json:"status"`              // pending|sent|delivered|read|failed
    ErrorDetail   *string   `gorm:"size:512" json:"errorDetail"`
    Context       string    `gorm:"size:16;index" json:"context"`             // chatbot|reminder|payslip|notification|manual|upsell
    ContextRefID  *uint     `json:"contextRefId"`
    CreatedAt     time.Time `json:"createdAt"`
    UpdatedAt     time.Time `json:"updatedAt"`
}

type ChatbotConversation struct {
    ID            uint      `gorm:"primaryKey" json:"id"`
    ChatID        string    `gorm:"uniqueIndex;size:64" json:"chatId"`
    PhoneNumber   string    `gorm:"size:24" json:"phoneNumber"`
    StudentName   *string   `gorm:"size:255" json:"studentName"`
    StudentCrmID  *uint     `json:"studentCrmId"` // soft FK → students_crm (cross-DB-owned table; no hard constraint)
    Status        string    `gorm:"size:12;index" json:"status"` // active|expired|escalated
    MessageCount  int       `json:"messageCount"`
    LastMessageAt time.Time `gorm:"index" json:"lastMessageAt"`  // expiry scans
    CreatedAt     time.Time `json:"createdAt"`
}

type ChatbotMessage struct {
    ID             uint      `gorm:"primaryKey" json:"id"`
    ConversationID uint      `gorm:"index" json:"conversationId"`
    Role           string    `gorm:"size:10" json:"role"` // user|assistant|system
    Content        string    `gorm:"type:text" json:"content"`
    Citations      *string   `gorm:"type:text" json:"citations"` // JSON array, e.g. ["kursus-3"]
    CreatedAt      time.Time `json:"createdAt"`
}

type NotificationSchedule struct {
    ID               uint      `gorm:"primaryKey" json:"id"`
    NotificationType string    `gorm:"uniqueIndex;size:24" json:"notificationType"` // session_reminder|daily_schedule|quota_low|upsell
    Enabled          bool      `json:"enabled"`
    LeadTimeMinutes  int       `json:"leadTimeMinutes"`
    CronExpression   *string   `gorm:"size:32" json:"cronExpression"`
    TemplateText     string    `gorm:"type:text" json:"templateText"`
    UpdatedAt        time.Time `json:"updatedAt"`
}
```

Plus the #3 decision — one nullable column on the existing `User` (`core/models/models.go`,
**snake_case** to match that file's convention):

```go
WhatsApp string `gorm:"type:varchar(24)" json:"whatsapp"` // nullable; instructor notifications
```

`StudentCrmID` is a **soft FK** — `students_crm` is owned by `backend/schema.sql`, so don't
add a hard GORM constraint that AutoMigrate would try to enforce cross-ownership.

---

## 22. Operational rigor

### 22.1 Observability

- **Structured logs** on every send/receive: `event`, `chatId` (mask all but last 4),
  `context`, `status`, `wahaMessageId`, latency. Never log message bodies at info level
  (PII).
- **Counters** (even if just logged periodically): sent/failed by `context`, chatbot
  replies, leads captured, current `WhatsAppSession.status`.
- **Health surfacing** — `/api/admin/whatsapp/status` already returns session state; treat
  `status=FAILED` or a stale `lastSyncedAt` (> a few minutes) as the disconnect alert the
  dashboard badge renders. A periodic sync goroutine refreshes the cache from WAHA.

### 22.2 Acceptance criteria (per phase, gate before moving on)

1. **WAHA + admin** — `docker compose up db waha` healthy; `/api/admin/whatsapp/status`
   returns live WAHA status; QR renders and a real scan flips status to `WORKING`; webhook
   receiver validates the §19.2 test vector (accept valid, 401 invalid/missing).
2. **Logging + send** — send-test delivers a real message; row in `WhatsAppMessageLog`
   transitions `pending→sent`, then `delivered`/`read` on ack; WAHA stopped ⇒ `failed` with
   `errorDetail`, gateway stays up.
3. **AI chatbot** — inbound DM gets an AI reply with citations; group/self/non-text ignored
   correctly; conversation expires after 30 min; lead-capture creates exactly one validated
   `StudentCrm`; FastAPI down ⇒ fallback message.
4. **Payslip delivery** — finalize sends text + PDF to employees with a number; missing
   number skipped + surfaced; failed send doesn't block finalize; redeliver works.
5. **Notifications** — reminder fires once at `leadTime` and flips `reminder_sent` (no
   duplicate on the next tick); 07:00 WIB daily schedule sends once per instructor; quota
   alert on `≤2h`; all gated by `NotificationSchedule.enabled`; sender honours 30–60 s
   spacing.

### 22.3 Migration & rollback

- **Migration is additive** — six AutoMigrate entries (five new tables + `User.WhatsApp`).
  No data backfill. The `knowledge.go` content tables are untouched (still owned by
  `backend/schema.sql`). The soft FK to `students_crm` adds no DB-level constraint.
- **Seeding is idempotent** — follow the existing `core/seed` pattern (existence-check
  before insert, as every current seeder does); the four `NotificationSchedule` rows (§10)
  upsert by `notificationType` so re-running `seed.RunAll()` is safe.
- **Rollback** — drop the five tables and the `User.whatsapp` column; remove the scheduler
  goroutine start and the webhook/admin routes. Because everything degrades gracefully
  (§5.2), a partial rollback (e.g. disabling the scheduler only) leaves the rest working.

### 22.4 Run/deploy runbook (this machine)

1. `docker compose up db waha` — MySQL + WAHA as containers (see §6.1).
2. Set `core/.env`: `WAHA_URL=http://localhost:3000`, `WAHA_API_KEY=…`,
   `WAHA_HMAC_KEY=…` (match the container's `WHATSAPP_HOOK_HMAC_KEY`), `WAHA_SESSION_NAME=default`.
3. Run apps on the host: `cd core && go run .` · `cd backend && uvicorn app.main:app --port 8081` · `cd frontend && ng serve`.
4. Open `/dashboard/whatsapp`, Start the session, scan the QR with the business phone →
   status `WORKING`.
5. Verify the inbound path: message the paired number; confirm WAHA reached the host
   webhook (allow the Windows Firewall prompt the first time). If nothing arrives, check
   `host.docker.internal` resolves from the container and Go is on `0.0.0.0:8080` (§18 #5).
6. Send a test message from the dashboard; confirm the `WhatsAppMessageLog` row and acks.
