package handlers

// Manager-only WhatsApp admin endpoints (mounted under /api/admin/whatsapp).
// All WAHA calls degrade gracefully — WAHA being down never crashes the gateway.

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

// syncSession upserts the local WhatsAppSession cache from a live WAHA response.
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
		// Graceful degradation: return last-known cache, or a STOPPED placeholder.
		var rec models.WhatsAppSession
		database.DB.Where("session_name = ?", WAClient.Session).First(&rec)
		if rec.SessionName == "" {
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
		database.DB.Model(&logRec).Updates(map[string]interface{}{"status": "failed", "error_detail": &msg})
		c.JSON(http.StatusBadGateway, gin.H{"error": "send failed", "id": logRec.ID})
		return
	}
	id := res.MessageID()
	logRec.Status = "sent"
	logRec.WahaMessageID = &id
	database.DB.Model(&logRec).Updates(map[string]interface{}{"status": "sent", "waha_message_id": &id})
	c.JSON(http.StatusOK, logRec)
}

func GetMessageLog(c *gin.Context) {
	logs := make([]models.WhatsAppMessageLog, 0)
	database.DB.Order("id desc").Limit(50).Find(&logs)
	c.JSON(http.StatusOK, logs)
}
