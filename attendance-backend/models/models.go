package models

import (
	"time"
)

// User represents a system user (Employee, Manager, or Instructor)
type User struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	Username         string    `gorm:"unique;not null" json:"username"`
	FullName         string    `gorm:"type:varchar(255)" json:"full_name"`
	PasswordHash     string    `gorm:"not null" json:"-"`
	Role             string    `gorm:"type:enum('employee','manager','instructor');default:'employee'" json:"role"`
	OfficeID         *uint     `json:"office_id,omitempty"` // Employee's primary office
	IsSuperAdmin     bool      `gorm:"default:false" json:"is_super_admin"`
	MinimumWorkHours float64   `gorm:"default:8.0" json:"minimum_work_hours"` // Manager specific setting
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	// Relations
	Office *OfficeLocation `gorm:"foreignKey:OfficeID" json:"office,omitempty"`
}

// Attendance represents a clock-in record
type Attendance struct {
	ID               uint            `gorm:"primaryKey" json:"id"`
	UserID           uint            `gorm:"not null" json:"user_id"`
	ClockInTime      time.Time       `gorm:"not null" json:"clock_in_time"`
	ClockOutTime     *time.Time      `json:"clock_out_time,omitempty"`
	Latitude         float64         `gorm:"type:decimal(10,8)" json:"latitude"`
	Longitude        float64         `gorm:"type:decimal(11,8)" json:"longitude"`
	LatitudeOut      *float64        `gorm:"type:decimal(10,8)" json:"latitude_out,omitempty"`
	LongitudeOut     *float64        `gorm:"type:decimal(11,8)" json:"longitude_out,omitempty"`
	Status           string          `gorm:"type:enum('approved','pending','rejected');default:'approved'" json:"status"`
	Distance         float64         `gorm:"type:decimal(10,2)" json:"distance"`
	WorkHours        *float64        `gorm:"type:decimal(10,2)" json:"work_hours,omitempty"`
	IsLate           bool            `gorm:"default:false" json:"is_late"`
	MinutesLate      int             `gorm:"default:0" json:"minutes_late"`
	ApprovedOfficeID *uint           `json:"approved_office_id,omitempty"` // Which office validated this
	User             User            `gorm:"foreignKey:UserID" json:"user"`
	ApprovedOffice   *OfficeLocation `gorm:"foreignKey:ApprovedOfficeID" json:"approved_office,omitempty"`
}

// LeaveRequest represents a leave application
type LeaveRequest struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	StartDate time.Time `gorm:"type:date;not null" json:"start_date"`
	EndDate   time.Time `gorm:"type:date;not null" json:"end_date"`
	Reason    string    `gorm:"type:text" json:"reason"`
	Status    string    `gorm:"type:enum('pending','approved','rejected');default:'pending'" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      User      `gorm:"foreignKey:UserID" json:"-"`
}

// OfficeLocation represents the office coordinates and allowed radius for clock-in
type OfficeLocation struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	Name                string    `gorm:"type:varchar(100);not null" json:"name"`
	Address             string    `gorm:"type:varchar(255)" json:"address"`
	Latitude            float64   `gorm:"type:decimal(10,8);not null" json:"latitude"`
	Longitude           float64   `gorm:"type:decimal(11,8);not null" json:"longitude"`
	AllowedRadiusMeters float64   `gorm:"type:decimal(10,2);not null" json:"allowed_radius_meters"`
	ClockInTime         string    `gorm:"type:varchar(5);not null" json:"clock_in_time"`
	IsActive            bool      `gorm:"default:true" json:"is_active"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// ManagerOffice junction table - links managers to 1-4 office locations
type ManagerOffice struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ManagerID  uint      `gorm:"not null" json:"manager_id"`
	OfficeID   uint      `gorm:"not null" json:"office_id"`
	AssignedAt time.Time `gorm:"autoCreateTime" json:"assigned_at"`

	// Relations
	Manager *User           `gorm:"foreignKey:ManagerID" json:"manager,omitempty"`
	Office  *OfficeLocation `gorm:"foreignKey:OfficeID" json:"office,omitempty"`
}

// SystemSettings stores global system configuration
type SystemSettings struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	SettingKey   string    `gorm:"type:varchar(50);unique;not null" json:"setting_key"`
	SettingValue string    `gorm:"type:varchar(255);not null" json:"setting_value"`
	Description  string    `gorm:"type:varchar(255)" json:"description"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Common setting keys
const (
	SettingSessionDurationHours = "session_duration_hours" // Default session duration in hours
	SettingQuotaPresetOptions   = "quota_preset_options"   // Comma-separated quota hour presets (e.g., "8,10")
)
