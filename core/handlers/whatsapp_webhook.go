package handlers

// Public WAHA webhook receiver. No auth middleware — validated by HMAC.
// WAHA (container) → host gateway via host.docker.internal:8080.

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

	// Fail closed: an unconfigured HMAC key must not leave the webhook open to
	// forged, unauthenticated DB mutations. Reject until a key is configured.
	key := waha.LoadConfig().HMACKey
	if key == "" {
		log.Println("[whatsapp] WAHA_HMAC_KEY not set — rejecting webhook; configure the key to enable it")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webhook not configured"})
		return
	}
	if !whatsapp.VerifyHMAC(raw, c.GetHeader("X-Webhook-Hmac"), key) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "bad hmac"})
		return
	}

	var env whatsapp.WebhookEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	switch env.Event {
	case "session.status":
		var p whatsapp.SessionStatusPayload
		_ = json.Unmarshal(env.Payload, &p)
		if env.Session != "" && p.Status != "" {
			rec := models.WhatsAppSession{SessionName: env.Session}
			database.DB.Where(models.WhatsAppSession{SessionName: env.Session}).
				Assign(models.WhatsAppSession{Status: p.Status}).
				FirstOrCreate(&rec)
		}
	case "message.ack":
		var p whatsapp.AckPayload
		_ = json.Unmarshal(env.Payload, &p)
		if st := whatsapp.AckToStatus(p.Ack); st != "" && p.ID != "" {
			database.DB.Model(&models.WhatsAppMessageLog{}).
				Where("waha_message_id = ?", p.ID).Update("status", st)
		}
	case "message":
		var p whatsapp.MessagePayload
		_ = json.Unmarshal(env.Payload, &p)
		if p.FromMe { // ignore our own messages
			break
		}
		// Phase 1: log inbound only. Chatbot orchestration is Phase 3.
		// Dedup is enforced by the unique index on waha_message_id.
		msgID := p.ID
		database.DB.Create(&models.WhatsAppMessageLog{
			Direction: "inbound", ChatID: p.From, MessageType: "text",
			Content: p.Body, Status: "delivered", Context: "chatbot",
			WahaMessageID: &msgID,
		})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
