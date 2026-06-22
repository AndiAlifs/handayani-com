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

// SendResult tolerates both WEBJS/WPP (id._serialized) and NOWEB/GOWS (key.id)
// response shapes — WAHA does not normalize this across engines.
type SendResult struct {
	ID struct {
		Serialized string `json:"_serialized"`
	} `json:"id"`
	Key struct {
		ID string `json:"id"`
	} `json:"key"`
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
func PhoneToChatID(phone string) string {
	return NormalizePhone(phone) + "@c.us"
}

// NormalizePhone mirrors handlers.normalizeWhatsApp (kept here to avoid an
// import cycle — waha must not import handlers).
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
