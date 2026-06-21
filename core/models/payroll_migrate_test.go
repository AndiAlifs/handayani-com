package models

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestPayrollModelsMigrate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&EmployeeProfile{},
		&EmployeeCompensation{},
		&PayComponent{},
		&EmployeeComponent{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, tbl := range []string{"employee_profiles", "employee_compensations", "pay_components", "employee_components"} {
		if !db.Migrator().HasTable(tbl) {
			t.Errorf("expected table %q to exist", tbl)
		}
	}
}
