# WhatsApp/WAHA Phase 1 (Foundation) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up the WAHA foundation — config + Docker service, a typed Go WAHA client, the DB models, the HMAC-validated webhook receiver, manager-only admin endpoints, and a manager dashboard page — so an admin can pair a number and send a test message.

**Architecture:** All WhatsApp I/O goes through the Go gateway (consistent with `aiproxy.go`). WAHA + MySQL run as Docker containers; the Go/FastAPI/Angular apps run on the host. Go→WAHA = `http://localhost:3000`; WAHA→Go webhook = `http://host.docker.internal:8080`. New tables are Go-owned (AutoMigrate), camelCase JSON. Everything degrades gracefully — WAHA down never breaks existing features.

**Tech Stack:** Go 1.26 (Gin v1.11, GORM v1.31, golang-jwt/v5), Angular 18 (standalone components, signals), MySQL 8, Docker Compose, WAHA (`devlikeapro/waha`, WEBJS engine).

## Global Constraints

- New DB tables: Go-owned, added to `AutoMigrate` in `core/main.go`; **camelCase** JSON tags (match `knowledge.go` + Angular models), NOT the snake_case `instructor.go` style. The new `User.WhatsApp` column is the exception — snake_case `json:"whatsapp"` to match `models.go`.
- Phone normalization: reuse `normalizeWhatsApp()` (`core/handlers/instructor.go`) → `628…`; chat ID = `<phone>@c.us`.
- WAHA auth: header `X-Api-Key: <WAHA_API_KEY>` on every WAHA call.
- Webhook HMAC: HMAC-**SHA512** over the **raw body bytes**, hex (lowercase), header `X-Webhook-Hmac`; key = `WAHA_HMAC_KEY`. If the key is empty (dev), skip verification but log a warning.
- Handlers return **bare JSON bodies** (no `gin.H{"data":...}` envelope), like `courses.go`/`crm.go`.
- Admin endpoints under `/api/admin/whatsapp`, gated `AuthMiddleware()` + `ManagerMiddleware()`. Webhook `POST /api/webhooks/whatsapp` is public (HMAC-validated).
- Graceful degradation: every `waha.Client` call returns `(result, error)`; callers log + record `failed`, never crash. Frontend `ApiService` methods get `catchError` mock fallbacks.
- WAHA send response shape is engine-dependent — normalize id as `id._serialized` (WEBJS/WPP) else `key.id` (NOWEB/GOWS).
- Env vars (read via `os.Getenv`, no config pkg): `WAHA_URL` (default `http://localhost:3000`), `WAHA_API_KEY`, `WAHA_HMAC_KEY`, `WAHA_SESSION_NAME` (default `default`).

**Verification reality:** Everything below is verifiable with `go build`/`go vet`/`go test` and `ng build`/`ng test` **without** a live WhatsApp pairing. Actually scanning the QR and sending to a real number is a manual step the user performs (runbook §22.4 of the spec).

---

## Task 1: WAHA config + Docker Compose service + env

**Files:**
- Create: `core/waha/config.go`
- Test: `core/waha/config_test.go`
- Modify: `compose.yml` (add `waha` service + `waha-data` volume)
- Modify: `core/.env.example` (add WAHA block)

**Interfaces:**
- Produces: `waha.Config{ BaseURL, APIKey, Session, HMACKey string }`; `waha.LoadConfig() Config` (reads env with defaults).

- [ ] **Step 1: Write the failing test** — `core/waha/config_test.go`

```go
package waha

import (
	"os"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	os.Unsetenv("WAHA_URL")
	os.Unsetenv("WAHA_SESSION_NAME")
	c := LoadConfig()
	if c.BaseURL != "http://localhost:3000" {
		t.Fatalf("BaseURL default = %q", c.BaseURL)
	}
	if c.Session != "default" {
		t.Fatalf("Session default = %q", c.Session)
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	os.Setenv("WAHA_URL", "http://waha-host:3000")
	os.Setenv("WAHA_SESSION_NAME", "sales")
	defer func() { os.Unsetenv("WAHA_URL"); os.Unsetenv("WAHA_SESSION_NAME") }()
	c := LoadConfig()
	if c.BaseURL != "http://waha-host:3000" || c.Session != "sales" {
		t.Fatalf("got %+v", c)
	}
}
```

- [ ] **Step 2: Run test, verify it fails** — `cd core && go test ./waha/` → FAIL (package/func undefined).

- [ ] **Step 3: Implement** — `core/waha/config.go`

```go
// Package waha is a thin, typed HTTP client for the WAHA WhatsApp HTTP API.
// No business logic — just HTTP + JSON. All other packages import this.
package waha

import "os"

type Config struct {
	BaseURL string // Go (host) → WAHA published port, e.g. http://localhost:3000
	APIKey  string // sent as X-Api-Key
	Session string // WAHA session name, default "default"
	HMACKey string // webhook signature key (WAHA_HMAC_KEY)
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func LoadConfig() Config {
	return Config{
		BaseURL: env("WAHA_URL", "http://localhost:3000"),
		APIKey:  os.Getenv("WAHA_API_KEY"),
		Session: env("WAHA_SESSION_NAME", "default"),
		HMACKey: os.Getenv("WAHA_HMAC_KEY"),
	}
}
```

- [ ] **Step 4: Run test, verify pass** — `cd core && go test ./waha/` → PASS.

- [ ] **Step 5: Add `waha` service to `compose.yml`** (insert before the `web:` service; add `waha-data` under `volumes:`):

```yaml
  waha:
    image: devlikeapro/waha:latest
    environment:
      WHATSAPP_DEFAULT_ENGINE: WEBJS
      WAHA_DASHBOARD_ENABLED: "true"
      WAHA_DASHBOARD_USERNAME: ${WAHA_DASHBOARD_USERNAME:-admin}
      WAHA_DASHBOARD_PASSWORD: ${WAHA_DASHBOARD_PASSWORD:-admin}
      WAHA_API_KEY: ${WAHA_API_KEY:-}
      # Container → host: the Go gateway runs on the host, not in compose.
      WHATSAPP_HOOK_URL: http://host.docker.internal:8080/api/webhooks/whatsapp
      WHATSAPP_HOOK_EVENTS: "message,message.ack,session.status"
      WHATSAPP_HOOK_HMAC_KEY: ${WAHA_HMAC_KEY:-waha-hmac-secret}
    ports:
      - "3000:3000"
    volumes:
      - waha-data:/app/.sessions
    extra_hosts:
      - "host.docker.internal:host-gateway"
```

And under the top-level `volumes:` add `waha-data:`.

- [ ] **Step 6: Add env block to `core/.env.example`:**

```env
# WhatsApp (WAHA) integration
WAHA_URL=http://localhost:3000
WAHA_API_KEY=
WAHA_HMAC_KEY=waha-hmac-secret
WAHA_SESSION_NAME=default
```

- [ ] **Step 7: Verify compose parses** — `docker compose config >/dev/null && echo OK` (if docker available; else skip).

- [ ] **Step 8: Commit** — `git add core/waha/ compose.yml core/.env.example && git commit -m "feat(whatsapp): WAHA config loader + compose service"`

---

## Task 2: `core/waha` typed client

**Files:**
- Create: `core/waha/types.go`, `core/waha/client.go`
- Test: `core/waha/client_test.go`

**Interfaces:**
- Consumes: `Config` (Task 1).
- Produces:
  - `New(c Config) *Client`
  - `(*Client) GetSession() (*SessionInfo, error)`
  - `(*Client) StartSession() (*SessionInfo, error)`, `StopSession`, `RestartSession`, `LogoutSession` (same signature)
  - `(*Client) GetMe() (*MeInfo, error)`
  - `(*Client) GetQR() ([]byte, string, error)` (bytes, contentType)
  - `(*Client) SendText(chatID, text string) (*SendResult, error)`
  - `(*SendResult) MessageID() string` (normalizes `id._serialized` vs `key.id`)
  - `PhoneToChatID(phone string) string`
  - Types: `SessionInfo{Name,Status string; Me *MeInfo}`, `MeInfo{ID,PushName string}`, `SendResult`.

- [ ] **Step 1: Write the failing test** — `core/waha/client_test.go`

```go
package waha

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testClient(srv *httptest.Server) *Client {
	return New(Config{BaseURL: srv.URL, APIKey: "k", Session: "default"})
}

func TestGetSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "k" {
			t.Errorf("missing api key header")
		}
		if r.URL.Path != "/api/sessions/default" {
			t.Errorf("path = %q", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"name": "default", "status": "WORKING",
			"me": map[string]string{"id": "628111@c.us", "pushName": "Biz"},
		})
	}))
	defer srv.Close()
	s, err := testClient(srv).GetSession()
	if err != nil || s.Status != "WORKING" || s.Me.PushName != "Biz" {
		t.Fatalf("got %+v err %v", s, err)
	}
}

func TestSendTextNormalizesID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sendText" {
			t.Errorf("path = %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req SendTextReq
		json.Unmarshal(body, &req)
		if req.ChatID != "628999@c.us" || req.Session != "default" {
			t.Errorf("req = %+v", req)
		}
		// WEBJS shape: nested id._serialized
		json.NewEncoder(w).Encode(map[string]any{"id": map[string]string{"_serialized": "MSG_1"}})
	}))
	defer srv.Close()
	res, err := testClient(srv).SendText("628999@c.us", "hi")
	if err != nil || res.MessageID() != "MSG_1" {
		t.Fatalf("got %+v err %v", res, err)
	}
}

func TestSendTextNowebID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"key": map[string]string{"id": "K_2"}})
	}))
	defer srv.Close()
	res, _ := testClient(srv).SendText("628@c.us", "hi")
	if res.MessageID() != "K_2" {
		t.Fatalf("noweb id = %q", res.MessageID())
	}
}

func TestPhoneToChatID(t *testing.T) {
	if got := PhoneToChatID("0812-345"); got != "62812345@c.us" {
		t.Fatalf("chatID = %q", got)
	}
}

func TestErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusInternalServerError)
	}))
	defer srv.Close()
	if _, err := testClient(srv).GetSession(); err == nil {
		t.Fatal("expected error on 500")
	}
}
```

- [ ] **Step 2: Run, verify fail** — `cd core && go test ./waha/` → FAIL.

- [ ] **Step 3: Implement `core/waha/types.go`**

```go
package waha

import "strings"

type MeInfo struct {
	ID       string `json:"id"`
	PushName string `json:"pushName"`
}

type SessionInfo struct {
	Name   string  `json:"name"`
	Status string  `json:"status"` // STOPPED|STARTING|SCAN_QR_CODE|WORKING|FAILED
	Me     *MeInfo `json:"me,omitempty"`
}

type SendTextReq struct {
	ChatID  string `json:"chatId"`
	Text    string `json:"text"`
	Session string `json:"session"`
}

// SendResult tolerates both WEBJS/WPP (id._serialized) and NOWEB/GOWS (key.id).
type SendResult struct {
	ID  struct{ Serialized string `json:"_serialized"` } `json:"id"`
	Key struct{ ID string `json:"id"` }                  `json:"key"`
}

func (r *SendResult) MessageID() string {
	if r == nil {
		return ""
	}
	if r.ID.Serialized != "" {
		return r.ID.Serialized
	}
	return r.Key.ID
}

// PhoneToChatID normalizes an Indonesian number and appends WAHA's DM suffix.
// normalizeWhatsApp lives in package handlers; we inline the same rules here to
// avoid an import cycle (waha must not import handlers).
func PhoneToChatID(phone string) string {
	return NormalizePhone(phone) + "@c.us"
}

func NormalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.NewReplacer(" ", "", "-", "", "+", "").Replace(phone)
	if strings.HasPrefix(phone, "0") {
		phone = "62" + phone[1:]
	}
	if !strings.HasPrefix(phone, "62") {
		phone = "62" + phone
	}
	return phone
}
```

- [ ] **Step 4: Implement `core/waha/client.go`**

```go
package waha

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	BaseURL string
	APIKey  string
	Session string
	HTTP    *http.Client
}

func New(c Config) *Client {
	return &Client{
		BaseURL: c.BaseURL,
		APIKey:  c.APIKey,
		Session: c.Session,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) do(method, path string, body any, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, rdr)
	if err != nil {
		return err
	}
	if c.APIKey != "" {
		req.Header.Set("X-Api-Key", c.APIKey)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("waha %s %s: %d %s", method, path, resp.StatusCode, string(b))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (c *Client) GetSession() (*SessionInfo, error) {
	var s SessionInfo
	err := c.do(http.MethodGet, "/api/sessions/"+c.Session, nil, &s)
	return &s, err
}

func (c *Client) sessionAction(action string) (*SessionInfo, error) {
	var s SessionInfo
	err := c.do(http.MethodPost, "/api/sessions/"+c.Session+"/"+action, nil, &s)
	return &s, err
}

func (c *Client) StartSession() (*SessionInfo, error)   { return c.sessionAction("start") }
func (c *Client) StopSession() (*SessionInfo, error)     { return c.sessionAction("stop") }
func (c *Client) RestartSession() (*SessionInfo, error)  { return c.sessionAction("restart") }
func (c *Client) LogoutSession() (*SessionInfo, error)   { return c.sessionAction("logout") }

func (c *Client) GetMe() (*MeInfo, error) {
	var m MeInfo
	err := c.do(http.MethodGet, "/api/sessions/"+c.Session+"/me", nil, &m)
	return &m, err
}

// GetQR returns the QR image bytes and its content-type for the pairing screen.
func (c *Client) GetQR() ([]byte, string, error) {
	u := fmt.Sprintf("%s/api/%s/auth/qr?format=image", c.BaseURL, url.PathEscape(c.Session))
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	if c.APIKey != "" {
		req.Header.Set("X-Api-Key", c.APIKey)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("waha qr: %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/png"
	}
	return b, ct, err
}

func (c *Client) SendText(chatID, text string) (*SendResult, error) {
	var res SendResult
	err := c.do(http.MethodPost, "/api/sendText",
		SendTextReq{ChatID: chatID, Text: text, Session: c.Session}, &res)
	return &res, err
}
```

- [ ] **Step 5: Run, verify pass** — `cd core && go test ./waha/ -v` → all PASS.

- [ ] **Step 6: Commit** — `git add core/waha/ && git commit -m "feat(whatsapp): typed WAHA client (session, qr, sendText)"`

---

## Task 3: GORM models + AutoMigrate + User.WhatsApp

**Files:**
- Create: `core/models/whatsapp.go`
- Modify: `core/models/models.go` (add `WhatsApp` to `User`)
- Modify: `core/main.go` (AutoMigrate the 5 new models)
- Test: `core/models/whatsapp_test.go` (sqlite in-memory AutoMigrate smoke test using `glebarez/sqlite`, already a dep)

**Interfaces:**
- Produces: `models.WhatsAppSession`, `models.WhatsAppMessageLog`, `models.ChatbotConversation`, `models.ChatbotMessage`, `models.NotificationSchedule` (fields exactly per spec §21).

- [ ] **Step 1: Write the failing test** — `core/models/whatsapp_test.go`

```go
package models

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestWhatsAppAutoMigrate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&WhatsAppSession{}, &WhatsAppMessageLog{},
		&ChatbotConversation{}, &ChatbotMessage{}, &NotificationSchedule{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	log := WhatsAppMessageLog{ChatID: "628@c.us", Direction: "outbound", Status: "pending", Context: "manual"}
	if err := db.Create(&log).Error; err != nil {
		t.Fatalf("create: %v", err)
	}
	if log.ID == 0 {
		t.Fatal("ID not assigned")
	}
}
```

- [ ] **Step 2: Run, verify fail** — `cd core && go test ./models/` → FAIL.

- [ ] **Step 3: Implement `core/models/whatsapp.go`** — paste the five structs verbatim from spec §21 (`WhatsAppSession`, `WhatsAppMessageLog`, `ChatbotConversation`, `ChatbotMessage`, `NotificationSchedule`), `package models`, `import "time"`.

- [ ] **Step 4: Add `WhatsApp` to `User`** in `core/models/models.go` — after the `IsSuperAdmin` line, add:

```go
	WhatsApp         string    `gorm:"type:varchar(24)" json:"whatsapp"` // nullable; instructor notifications
```

- [ ] **Step 5: Register in AutoMigrate** — in `core/main.go`, append to the `AutoMigrate(...)` list (after `&models.EmployeeComponent{},`):

```go
		&models.WhatsAppSession{},
		&models.WhatsAppMessageLog{},
		&models.ChatbotConversation{},
		&models.ChatbotMessage{},
		&models.NotificationSchedule{},
```

- [ ] **Step 6: Run, verify pass + build** — `cd core && go test ./models/ && go build ./...` → PASS / no errors.

- [ ] **Step 7: Commit** — `git add core/models/ core/main.go && git commit -m "feat(whatsapp): GORM models + User.whatsapp + automigrate"`

---

## Task 4: Webhook HMAC verification + receiver

**Files:**
- Create: `core/whatsapp/hmac.go`, `core/whatsapp/webhook.go`
- Create: `core/handlers/whatsapp_webhook.go`
- Test: `core/whatsapp/hmac_test.go`

**Interfaces:**
- Produces: `whatsapp.VerifyHMAC(raw []byte, header, key string) bool`; `whatsapp.WebhookEnvelope`, `AckPayload`, `SessionStatusPayload`, `MessagePayload` (per spec §19.1); `handlers.WAWebhook(c *gin.Context)`.
- Consumes: `models.*` (Task 3), `database.DB`.

- [ ] **Step 1: Write the failing test** — `core/whatsapp/hmac_test.go` (uses the docs' test vector)

```go
package whatsapp

import "testing"

func TestVerifyHMACDocsVector(t *testing.T) {
	body := []byte(`{"event":"message","session":"default","engine":"WEBJS"}`)
	const sig = "208f8a55dde9e05519e898b10b89bf0d0b3b0fdf11fdbf09b6b90476301b98d8097c462b2b17a6ce93b6b47a136cf2e78a33a63f6752c2c1631777076153fa89"
	if !VerifyHMAC(body, sig, "my-secret-key") {
		t.Fatal("valid signature rejected")
	}
	if VerifyHMAC(body, "deadbeef", "my-secret-key") {
		t.Fatal("invalid signature accepted")
	}
	if VerifyHMAC(body, sig, "wrong-key") {
		t.Fatal("wrong key accepted")
	}
}
```

- [ ] **Step 2: Run, verify fail** — `cd core && go test ./whatsapp/` → FAIL.

- [ ] **Step 3: Implement `core/whatsapp/hmac.go`**

```go
// Package whatsapp holds WhatsApp domain logic: webhook parsing, HMAC, and
// (later) chatbot orchestration + the reminder scheduler.
package whatsapp

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
)

// VerifyHMAC checks WAHA's X-Webhook-Hmac (HMAC-SHA512 over the raw body, hex).
func VerifyHMAC(raw []byte, header, key string) bool {
	mac := hmac.New(sha512.New, []byte(key))
	mac.Write(raw)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(header))
}
```

- [ ] **Step 4: Run, verify pass** — `cd core && go test ./whatsapp/` → PASS.

- [ ] **Step 5: Implement `core/whatsapp/webhook.go`** (envelope + payload types per spec §19.1)

```go
package whatsapp

import "encoding/json"

type WebhookEnvelope struct {
	ID        string          `json:"id"`
	Timestamp int64           `json:"timestamp"` // ms
	Event     string          `json:"event"`
	Session   string          `json:"session"`
	Engine    string          `json:"engine"`
	Payload   json.RawMessage `json:"payload"`
}

type MessagePayload struct {
	ID       string `json:"id"`
	From     string `json:"from"`
	To       string `json:"to"`
	FromMe   bool   `json:"fromMe"`
	Body     string `json:"body"`
	HasMedia bool   `json:"hasMedia"`
}

type AckPayload struct {
	ID      string `json:"id"`
	From    string `json:"from"`
	Ack     int    `json:"ack"`
	AckName string `json:"ackName"`
}

type SessionStatusPayload struct {
	Status string `json:"status"`
}

// AckToStatus maps WAHA ack codes to WhatsAppMessageLog.status.
func AckToStatus(ack int) string {
	switch ack {
	case -1:
		return "failed"
	case 0:
		return "pending"
	case 1:
		return "sent"
	case 2:
		return "delivered"
	case 3, 4:
		return "read"
	}
	return ""
}
```

- [ ] **Step 6: Implement `core/handlers/whatsapp_webhook.go`** (public, HMAC-validated; dispatch ack→status, session.status→cache; message→log inbound, chatbot deferred to Phase 3)

```go
package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"handayani-core/database"
	"handayani-core/models"
	"handayani-core/waha"
	"handayani-core/whatsapp"

	"github.com/gin-gonic/gin"
)

func WAWebhook(c *gin.Context) {
	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20)) // 1 MB cap
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(raw))

	key := waha.LoadConfig().HMACKey
	if key != "" {
		if !whatsapp.VerifyHMAC(raw, c.GetHeader("X-Webhook-Hmac"), key) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "bad hmac"})
			return
		}
	} else {
		log.Println("[whatsapp] WAHA_HMAC_KEY empty — webhook signature NOT verified")
	}

	var env whatsapp.WebhookEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	switch env.Event {
	case "session.status":
		var p whatsapp.SessionStatusPayload
		json.Unmarshal(env.Payload, &p)
		database.DB.Where(models.WhatsAppSession{SessionName: env.Session}).
			Assign(models.WhatsAppSession{Status: p.Status}).
			FirstOrCreate(&models.WhatsAppSession{})
	case "message.ack":
		var p whatsapp.AckPayload
		json.Unmarshal(env.Payload, &p)
		if st := whatsapp.AckToStatus(p.Ack); st != "" {
			database.DB.Model(&models.WhatsAppMessageLog{}).
				Where("waha_message_id = ?", p.ID).Update("status", st)
		}
	case "message":
		var p whatsapp.MessagePayload
		json.Unmarshal(env.Payload, &p)
		if p.FromMe { // ignore our own
			break
		}
		// Phase 1: log inbound only; chatbot orchestration is Phase 3.
		database.DB.Create(&models.WhatsAppMessageLog{
			Direction: "inbound", ChatID: p.From, MessageType: "text",
			Content: p.Body, Status: "delivered", Context: "chatbot",
			WahaMessageID: &p.ID,
		})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
```

- [ ] **Step 7: Build + test** — `cd core && go build ./... && go test ./whatsapp/` → OK.

- [ ] **Step 8: Commit** — `git add core/whatsapp/ core/handlers/whatsapp_webhook.go && git commit -m "feat(whatsapp): HMAC-validated webhook receiver"`

---

## Task 5: Admin handlers + route wiring

**Files:**
- Create: `core/handlers/whatsapp.go`
- Modify: `core/main.go` (init `waha.Client`, register routes)
- Test: `core/handlers/whatsapp_test.go` (send-test logs a row; mock WAHA via httptest)

**Interfaces:**
- Produces handlers: `GetWhatsAppStatus`, `StartWhatsApp`, `StopWhatsApp`, `RestartWhatsApp`, `LogoutWhatsApp`, `GetWhatsAppQR`, `SendTestMessage`, `GetMessageLog`.
- Consumes: `waha.Client` (Task 2), `models.*` (Task 3), `database.DB`.

- [ ] **Step 1: Implement `core/handlers/whatsapp.go`** — a package-level client `var WAClient *waha.Client` set from `main`, plus:

```go
package handlers

import (
	"net/http"
	"time"

	"handayani-core/database"
	"handayani-core/models"
	"handayani-core/waha"

	"github.com/gin-gonic/gin"
)

// WAClient is initialized once in main() from waha.LoadConfig().
var WAClient *waha.Client

func syncSession(s *waha.SessionInfo) models.WhatsAppSession {
	rec := models.WhatsAppSession{SessionName: WAClient.Session}
	phone := ""
	if s.Me != nil {
		phone = s.Me.ID
	}
	database.DB.Where(models.WhatsAppSession{SessionName: WAClient.Session}).
		Assign(models.WhatsAppSession{Status: s.Status, PhoneNumber: phone, LastSyncedAt: time.Now()}).
		FirstOrCreate(&rec)
	return rec
}

func GetWhatsAppStatus(c *gin.Context) {
	s, err := WAClient.GetSession()
	if err != nil {
		// graceful degradation: return last-known cache, or STOPPED
		var rec models.WhatsAppSession
		database.DB.Where("session_name = ?", WAClient.Session).First(&rec)
		if rec.Status == "" {
			rec = models.WhatsAppSession{SessionName: WAClient.Session, Status: "STOPPED"}
		}
		c.JSON(http.StatusOK, rec)
		return
	}
	c.JSON(http.StatusOK, syncSession(s))
}

func waSessionAction(c *gin.Context, fn func() (*waha.SessionInfo, error)) {
	s, err := fn()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "WAHA unavailable"})
		return
	}
	c.JSON(http.StatusOK, syncSession(s))
}

func StartWhatsApp(c *gin.Context)   { waSessionAction(c, WAClient.StartSession) }
func StopWhatsApp(c *gin.Context)    { waSessionAction(c, WAClient.StopSession) }
func RestartWhatsApp(c *gin.Context) { waSessionAction(c, WAClient.RestartSession) }
func LogoutWhatsApp(c *gin.Context)  { waSessionAction(c, WAClient.LogoutSession) }

func GetWhatsAppQR(c *gin.Context) {
	b, ct, err := WAClient.GetQR()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "QR unavailable"})
		return
	}
	c.Data(http.StatusOK, ct, b)
}

type sendTestReq struct {
	Phone string `json:"phone"`
	Text  string `json:"text"`
}

func SendTestMessage(c *gin.Context) {
	var req sendTestReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Phone == "" || req.Text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone and text required"})
		return
	}
	phone := waha.NormalizePhone(req.Phone)
	chatID := phone + "@c.us"
	logRec := models.WhatsAppMessageLog{
		Direction: "outbound", ChatID: chatID, PhoneNumber: phone,
		MessageType: "text", Content: req.Text, Status: "pending", Context: "manual",
	}
	database.DB.Create(&logRec)

	res, err := WAClient.SendText(chatID, req.Text)
	if err != nil {
		msg := err.Error()
		database.DB.Model(&logRec).Updates(models.WhatsAppMessageLog{Status: "failed", ErrorDetail: &msg})
		c.JSON(http.StatusBadGateway, gin.H{"error": "send failed", "id": logRec.ID})
		return
	}
	id := res.MessageID()
	database.DB.Model(&logRec).Updates(models.WhatsAppMessageLog{Status: "sent", WahaMessageID: &id})
	c.JSON(http.StatusOK, logRec)
}

func GetMessageLog(c *gin.Context) {
	var logs []models.WhatsAppMessageLog
	q := database.DB.Order("id desc").Limit(50)
	q.Find(&logs)
	c.JSON(http.StatusOK, logs)
}
```

- [ ] **Step 2: Wire routes + client init in `core/main.go`** — after the CRM group, before `r.Run`:

```go
	// WhatsApp (WAHA) — manager-only admin + public webhook.
	handlers.WAClient = waha.New(waha.LoadConfig())
	r.POST("/api/webhooks/whatsapp", handlers.WAWebhook) // public, HMAC-validated
	wa := r.Group("/api/admin/whatsapp")
	wa.Use(auth.AuthMiddleware(), auth.ManagerMiddleware())
	{
		wa.GET("/status", handlers.GetWhatsAppStatus)
		wa.POST("/start", handlers.StartWhatsApp)
		wa.POST("/stop", handlers.StopWhatsApp)
		wa.POST("/restart", handlers.RestartWhatsApp)
		wa.POST("/logout", handlers.LogoutWhatsApp)
		wa.GET("/qr", handlers.GetWhatsAppQR)
		wa.POST("/send-test", handlers.SendTestMessage)
		wa.GET("/messages", handlers.GetMessageLog)
	}
```

Add `"handayani-core/waha"` to the import block.

- [ ] **Step 3: Write the handler test** — `core/handlers/whatsapp_test.go`: spin an httptest WAHA returning a WEBJS send result, point `WAClient` at it + an in-memory sqlite `database.DB`, `POST /send-test`, assert 200 + a `sent` row. (Mirror `auth.service.spec` style — `httptest.NewServer`, `gin.New()`, `database.DB = <sqlite>`.)

- [ ] **Step 4: Build + vet + test** — `cd core && go build ./... && go vet ./... && go test ./...` → all OK.

- [ ] **Step 5: Commit** — `git add core/handlers/whatsapp.go core/main.go core/handlers/whatsapp_test.go && git commit -m "feat(whatsapp): manager admin endpoints + routes"`

---

## Task 6: Frontend model + ApiService + mocks + managerGuard

**Files:**
- Create: `frontend/src/app/core/models/whatsapp.model.ts`
- Create: `frontend/src/app/core/guards/manager.guard.ts`
- Modify: `frontend/src/app/core/services/api.service.ts`
- Modify: `frontend/src/app/core/services/mock-data.ts`

**Interfaces:**
- Produces: `WhatsAppStatus`, `WhatsAppMessage` interfaces (per spec §12.3); `managerGuard: CanActivateFn`; ApiService methods `getWhatsAppStatus()`, `startWhatsApp()`, `stopWhatsApp()`, `restartWhatsApp()`, `logoutWhatsApp()`, `getWhatsAppQR()`, `sendTestMessage(phone,text)`, `getMessageLog()`.

- [ ] **Step 1: Create `whatsapp.model.ts`** — `WhatsAppStatus` + `WhatsAppMessage` interfaces verbatim from spec §12.3.

- [ ] **Step 2: Create `manager.guard.ts`**

```ts
import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from '../services/auth.service';

export const managerGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);
  if (auth.isManager()) return true;
  router.navigate(['/dashboard']);
  return false;
};
```

- [ ] **Step 3: Add mocks to `mock-data.ts`**

```ts
export const MOCK_WHATSAPP_STATUS: WhatsAppStatus = {
  sessionName: 'default', status: 'STOPPED', phoneNumber: '',
  pairedAt: null, lastSyncedAt: new Date().toISOString(),
};
export const MOCK_WHATSAPP_MESSAGES: WhatsAppMessage[] = [];
```
(import the types at top: `import { WhatsAppStatus, WhatsAppMessage } from '../models/whatsapp.model';` and re-export via the existing `export type` block.)

- [ ] **Step 4: Add ApiService methods** — following the `catchError`+mock pattern; QR uses `responseType:'blob'` (no fallback, like `AttendanceService`):

```ts
getWhatsAppStatus(): Observable<WhatsAppStatus> {
  return this.http.get<WhatsAppStatus>(`${this.baseUrl}/api/admin/whatsapp/status`).pipe(
    catchError(() => of(MOCK_WHATSAPP_STATUS)));
}
startWhatsApp(): Observable<WhatsAppStatus> {
  return this.http.post<WhatsAppStatus>(`${this.baseUrl}/api/admin/whatsapp/start`, {}).pipe(
    catchError(() => of(MOCK_WHATSAPP_STATUS)));
}
stopWhatsApp(): Observable<WhatsAppStatus> {
  return this.http.post<WhatsAppStatus>(`${this.baseUrl}/api/admin/whatsapp/stop`, {}).pipe(
    catchError(() => of(MOCK_WHATSAPP_STATUS)));
}
restartWhatsApp(): Observable<WhatsAppStatus> {
  return this.http.post<WhatsAppStatus>(`${this.baseUrl}/api/admin/whatsapp/restart`, {}).pipe(
    catchError(() => of(MOCK_WHATSAPP_STATUS)));
}
logoutWhatsApp(): Observable<WhatsAppStatus> {
  return this.http.post<WhatsAppStatus>(`${this.baseUrl}/api/admin/whatsapp/logout`, {}).pipe(
    catchError(() => of(MOCK_WHATSAPP_STATUS)));
}
getWhatsAppQR(): Observable<Blob> {
  return this.http.get(`${this.baseUrl}/api/admin/whatsapp/qr`, { responseType: 'blob' });
}
sendTestMessage(phone: string, text: string): Observable<WhatsAppMessage | null> {
  return this.http.post<WhatsAppMessage>(`${this.baseUrl}/api/admin/whatsapp/send-test`, { phone, text }).pipe(
    catchError(() => of(null)));
}
getMessageLog(): Observable<WhatsAppMessage[]> {
  return this.http.get<WhatsAppMessage[]>(`${this.baseUrl}/api/admin/whatsapp/messages`).pipe(
    catchError(() => of(MOCK_WHATSAPP_MESSAGES)));
}
```
(import the new mocks + types at top of `api.service.ts`.)

- [ ] **Step 5: Build** — `cd frontend && npm run build` → success.

- [ ] **Step 6: Commit** — `git add frontend/src/app/core && git commit -m "feat(whatsapp): frontend model, api methods, mocks, manager guard"`

---

## Task 7: Frontend WhatsApp dashboard page + route + nav

**Files:**
- Create: `frontend/src/app/dashboard/whatsapp/whatsapp.component.ts` (+ `.html`, `.scss`)
- Modify: `frontend/src/app/app.routes.ts` (add child route w/ `[authGuard, managerGuard]`)
- Modify: `frontend/src/app/dashboard/dashboard-layout/dashboard-layout.component.ts` (nav item) + `.html` (svg case)

**Interfaces:**
- Consumes: ApiService methods + `WhatsAppStatus`/`WhatsAppMessage` (Task 6), `managerGuard` (Task 6).

- [ ] **Step 1: Create `whatsapp.component.ts`** — standalone component (signals) with: status card (badge from `status()`: WORKING→🟢 Terhubung, SCAN_QR_CODE→🟡 Scan QR, else 🔴 Terputus), action buttons (start/stop/restart/logout calling ApiService then refreshing status), QR `<img>` bound to an object URL from `getWhatsAppQR()` that polls every 5s while `status()==='SCAN_QR_CODE'`, a send-test form (phone+text → `sendTestMessage` → refresh log), and a message-log table from `getMessageLog()`. Use `OnInit` to load status+log; `OnDestroy` to clear the poll interval and revoke object URLs. Hardcoded Indonesian labels (consistent with existing dashboard).

- [ ] **Step 2: Add the route** in `app.routes.ts` (inside `dashboard` `children`):

```ts
{
  path: 'whatsapp',
  canActivate: [managerGuard],
  loadComponent: () => import('./dashboard/whatsapp/whatsapp.component').then(m => m.WhatsappComponent)
},
```
(import `managerGuard` at top.)

- [ ] **Step 3: Add nav item** in `dashboard-layout.component.ts` `navItems`:

```ts
{ label: 'WhatsApp', icon: 'whatsapp', route: '/dashboard/whatsapp', roles: ['manager'] },
```
and add a `*ngSwitchCase="'whatsapp'"` inline SVG in `dashboard-layout.component.html`.

- [ ] **Step 4: Build + test** — `cd frontend && npm run build && npm test -- --watch=false --browsers=ChromeHeadless` (skip browser test if no Chrome) → build OK.

- [ ] **Step 5: Commit** — `git add frontend/src/app && git commit -m "feat(whatsapp): manager dashboard page + route + nav"`

---

## Phase 1 acceptance (per spec §22.2.1)

- `go build ./... && go vet ./... && go test ./...` green; `npm run build` green.
- `docker compose up db waha` healthy; `GET /api/admin/whatsapp/status` returns live status (manager token); QR renders; scanning flips status to `WORKING` (**manual**).
- Webhook validates the §19.2 test vector (accept valid, 401 invalid/missing) — covered by `hmac_test.go`.
- Send-test delivers and logs `pending→sent`; WAHA stopped ⇒ `failed`, gateway stays up.

## Out of scope (later phases, own plans)

Phase 2 message-log pagination/filters · Phase 3 chatbot (`ChatbotConversation/Message`, FastAPI `/api/wa/chat`, lead capture) · Phase 4 payslip delivery · Phase 5 scheduler + notifications + settings UI.
