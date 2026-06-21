package models

import "time"

// Payroll master-data models (PRD: internal payroll module — see
// docs/planned/payroll-module-design.md). Unlike some attendance models these
// use plain varchar string fields with app-level validation (NOT MySQL
// type:enum) and plain indexed uint FK columns with no DB-level foreign-key
// constraints, so the models migrate cleanly under the pure-Go sqlite driver
// used in tests. Money is int64 rupiah; dates are *time.Time (type:date).

// EmployeeProfile — payroll-administrative layer, 1:1 with User (UserID is a
// plain unique-indexed column; no DB-level FK constraint).
type EmployeeProfile struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	UserID          uint       `gorm:"uniqueIndex;not null" json:"userId"`
	NIK             string     `gorm:"type:varchar(32)" json:"nik"`
	NPWP            string     `gorm:"type:varchar(32)" json:"npwp"` // empty => +20% PPh 21 (Phase 2)
	PtkpStatus      string     `gorm:"type:varchar(8)" json:"ptkpStatus"`      // TK/0 … K/3
	EmploymentType  string     `gorm:"type:varchar(16)" json:"employmentType"` // permanent|contract|freelance
	Pph21Category   string     `gorm:"type:varchar(24)" json:"pph21Category"`  // pegawai_tetap|bukan_pegawai|pegawai_tidak_tetap
	BankName        string     `gorm:"type:varchar(64)" json:"bankName"`
	BankAccountNo   string     `gorm:"type:varchar(40)" json:"bankAccountNo"`
	BankAccountName string     `gorm:"type:varchar(128)" json:"bankAccountName"`
	BpjsKesehatanNo string     `gorm:"type:varchar(32)" json:"bpjsKesehatanNo"`
	BpjsTkNo        string     `gorm:"type:varchar(32)" json:"bpjsTkNo"`
	Email           string     `gorm:"type:varchar(128)" json:"email"`
	WhatsApp        string     `gorm:"type:varchar(24)" json:"whatsapp"`
	JoinDate        *time.Time `gorm:"type:date" json:"joinDate"`
	IsActive        bool       `gorm:"default:true" json:"isActive"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

// EmployeeCompensation — pay basis for an employee (effective-dated).
type EmployeeCompensation struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	UserID        uint       `gorm:"index;not null" json:"userId"`
	PayBasis      string     `gorm:"type:varchar(16)" json:"payBasis"` // monthly_fixed|per_session|per_hour
	BaseSalary    int64      `json:"baseSalary"`                       // rupiah, for monthly_fixed
	Rate          int64      `json:"rate"`                             // rupiah per session/hour
	EffectiveFrom *time.Time `gorm:"type:date" json:"effectiveFrom"`
	EffectiveTo   *time.Time `gorm:"type:date" json:"effectiveTo"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// PayComponent — master list of earning/deduction types.
type PayComponent struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	Code          string `gorm:"type:varchar(32);uniqueIndex" json:"code"`
	Name          string `gorm:"type:varchar(128)" json:"name"`
	ComponentType string `gorm:"type:varchar(12)" json:"componentType"` // earning|deduction
	Taxable       bool   `json:"taxable"`
	IsBpjsBase    bool   `json:"isBpjsBase"`
	DefaultCalc   string `gorm:"type:varchar(12)" json:"defaultCalc"` // fixed|manual
}

// EmployeeComponent — a recurring component assigned to an employee.
type EmployeeComponent struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	UserID        uint       `gorm:"index;not null" json:"userId"`
	ComponentID   uint       `gorm:"index;not null" json:"componentId"`
	Amount        int64      `json:"amount"` // rupiah
	EffectiveFrom *time.Time `gorm:"type:date" json:"effectiveFrom"`
	EffectiveTo   *time.Time `gorm:"type:date" json:"effectiveTo"`
}
