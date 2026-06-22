// Package whatsapp holds WhatsApp domain logic: webhook parsing, HMAC, and
// (later) chatbot orchestration + the reminder scheduler.
package whatsapp

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
)

// VerifyHMAC checks WAHA's X-Webhook-Hmac (HMAC-SHA512 over the raw body, hex).
// Comparison is constant-time. The caller must pass the RAW request body bytes,
// read before any JSON binding.
func VerifyHMAC(raw []byte, header, key string) bool {
	mac := hmac.New(sha512.New, []byte(key))
	mac.Write(raw)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(header))
}
