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
