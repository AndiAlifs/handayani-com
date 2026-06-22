package models

import "time"

// WhatsApp/WAHA integration models — Go-owned, AutoMigrated, camelCase JSON to
// match the knowledge.go content models and the Angular wire contract.

// WhatsAppSession mirrors the WAHA session state locally so the dashboard can
// render status without polling WAHA on every page load.
type WhatsAppSession struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	SessionName  string     `gorm:"uniqueIndex;size:64" json:"sessionName"`
	Status       string     `gorm:"size:16" json:"status"` // STOPPED|STARTING|SCAN_QR_CODE|WORKING|FAILED
	PhoneNumber  string     `gorm:"size:24" json:"phoneNumber"`
	PairedAt     *time.Time `json:"pairedAt"`
	LastSyncedAt time.Time  `json:"lastSyncedAt"`
}

// WhatsAppMessageLog is the outbound/inbound message audit trail. wahaMessageId
// is uniquely indexed for webhook dedup + ack correlation.
type WhatsAppMessageLog struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Direction     string    `gorm:"size:8;index" json:"direction"` // outbound|inbound
	ChatID        string    `gorm:"size:64;index" json:"chatId"`   // 628xxx@c.us
	PhoneNumber   string    `gorm:"size:24" json:"phoneNumber"`
	MessageType   string    `gorm:"size:16" json:"messageType"` // text|image|file|voice|...
	Content       string    `gorm:"type:text" json:"content"`
	MediaURL      *string   `gorm:"size:512" json:"mediaUrl"`
	WahaMessageID *string   `gorm:"uniqueIndex;size:128" json:"wahaMessageId"`
	Status        string    `gorm:"size:12;index" json:"status"`  // pending|sent|delivered|read|failed
	ErrorDetail   *string   `gorm:"size:512" json:"errorDetail"`
	Context       string    `gorm:"size:16;index" json:"context"` // chatbot|reminder|payslip|notification|manual|upsell
	ContextRefID  *uint     `json:"contextRefId"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// ChatbotConversation holds per-contact AI conversation state.
type ChatbotConversation struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ChatID        string    `gorm:"uniqueIndex;size:64" json:"chatId"`
	PhoneNumber   string    `gorm:"size:24" json:"phoneNumber"`
	StudentName   *string   `gorm:"size:255" json:"studentName"`
	StudentCrmID  *uint     `json:"studentCrmId"` // soft FK → students_crm (cross-DB-owned; no hard constraint)
	Status        string    `gorm:"size:12;index" json:"status"` // active|expired|escalated
	MessageCount  int       `json:"messageCount"`
	LastMessageAt time.Time `gorm:"index" json:"lastMessageAt"`
	CreatedAt     time.Time `json:"createdAt"`
}

// ChatbotMessage is a single message within a conversation.
type ChatbotMessage struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ConversationID uint      `gorm:"index" json:"conversationId"`
	Role           string    `gorm:"size:10" json:"role"` // user|assistant|system
	Content        string    `gorm:"type:text" json:"content"`
	Citations      *string   `gorm:"type:text" json:"citations"` // JSON array, e.g. ["kursus-3"]
	CreatedAt      time.Time `json:"createdAt"`
}

// NotificationSchedule is the per-type configurable notification rule.
type NotificationSchedule struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	NotificationType string    `gorm:"uniqueIndex;size:24" json:"notificationType"` // session_reminder|daily_schedule|quota_low|upsell
	Enabled          bool      `json:"enabled"`
	LeadTimeMinutes  int       `json:"leadTimeMinutes"`
	CronExpression   *string   `gorm:"size:32" json:"cronExpression"`
	TemplateText     string    `gorm:"type:text" json:"templateText"`
	UpdatedAt        time.Time `json:"updatedAt"`
}
