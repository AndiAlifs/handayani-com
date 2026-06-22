package whatsapp

import "encoding/json"

// WebhookEnvelope is the common shape on every WAHA webhook POST. Decode Payload
// per Event. Note: envelope Timestamp is in milliseconds (message payload
// timestamps are in seconds).
type WebhookEnvelope struct {
	ID        string          `json:"id"`
	Timestamp int64           `json:"timestamp"`
	Event     string          `json:"event"`
	Session   string          `json:"session"`
	Engine    string          `json:"engine"`
	Payload   json.RawMessage `json:"payload"`
}

// MessagePayload is the payload for event="message".
type MessagePayload struct {
	ID       string `json:"id"`
	From     string `json:"from"` // 628xxx@c.us (DM) | ...@g.us (group)
	To       string `json:"to"`
	FromMe   bool   `json:"fromMe"`
	Body     string `json:"body"`
	HasMedia bool   `json:"hasMedia"`
}

// AckPayload is the payload for event="message.ack".
type AckPayload struct {
	ID      string `json:"id"`
	From    string `json:"from"`
	Ack     int    `json:"ack"`
	AckName string `json:"ackName"`
}

// SessionStatusPayload is the payload for event="session.status".
type SessionStatusPayload struct {
	Status string `json:"status"`
}

// AckToStatus maps a WAHA ack code to a WhatsAppMessageLog.status value.
// Unknown codes return "" (no update).
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
