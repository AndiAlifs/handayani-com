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
