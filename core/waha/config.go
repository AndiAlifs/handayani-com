// Package waha is a thin, typed HTTP client for the WAHA WhatsApp HTTP API.
// No business logic — just HTTP + JSON. All other packages import this.
package waha

import "os"

// Config holds the connection settings for the WAHA gateway. On this machine
// WAHA runs in Docker and the Go gateway runs on the host, so BaseURL points at
// WAHA's published port (http://localhost:3000), not a compose service name.
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

// LoadConfig reads the WAHA_* environment variables, applying defaults.
func LoadConfig() Config {
	return Config{
		BaseURL: env("WAHA_URL", "http://localhost:3000"),
		APIKey:  os.Getenv("WAHA_API_KEY"),
		Session: env("WAHA_SESSION_NAME", "default"),
		HMACKey: os.Getenv("WAHA_HMAC_KEY"),
	}
}
