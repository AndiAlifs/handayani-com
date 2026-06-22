package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"handayani-core/database"
	"handayani-core/models"
	"handayani-core/waha"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupWADB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.WhatsAppMessageLog{}, &models.WhatsAppSession{}); err != nil {
		t.Fatal(err)
	}
	database.DB = db
}

func TestSendTestMessageLogsSent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupWADB(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"id": map[string]string{"_serialized": "MSG_OK"}})
	}))
	defer srv.Close()
	WAClient = waha.New(waha.Config{BaseURL: srv.URL, Session: "default"})

	r := gin.New()
	r.POST("/send-test", SendTestMessage)
	req := httptest.NewRequest(http.MethodPost, "/send-test",
		strings.NewReader(`{"phone":"0812-345","text":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body %s", w.Code, w.Body.String())
	}
	var got models.WhatsAppMessageLog
	if err := database.DB.Order("id desc").First(&got).Error; err != nil {
		t.Fatalf("no log row: %v", err)
	}
	if got.Status != "sent" || got.PhoneNumber != "62812345" || got.WahaMessageID == nil || *got.WahaMessageID != "MSG_OK" {
		t.Fatalf("log = %+v", got)
	}
}

func TestSendTestMessageFailsGracefully(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupWADB(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusInternalServerError)
	}))
	defer srv.Close()
	WAClient = waha.New(waha.Config{BaseURL: srv.URL, Session: "default"})

	r := gin.New()
	r.POST("/send-test", SendTestMessage)
	req := httptest.NewRequest(http.MethodPost, "/send-test",
		strings.NewReader(`{"phone":"628999","text":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", w.Code)
	}
	var got models.WhatsAppMessageLog
	database.DB.Order("id desc").First(&got)
	if got.Status != "failed" || got.ErrorDetail == nil {
		t.Fatalf("expected failed log, got %+v", got)
	}
}
