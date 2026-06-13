package models

import (
	"time"

	"gorm.io/gorm"
)

// InstructorBase is a drop-in replacement for gorm.Model that serialises ID as lowercase "id".
type InstructorBase struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Student represents the learner managed by the instructor.
type Student struct {
	InstructorBase
	Name                string  `json:"name"`
	InstructorID        uint    `json:"instructor_id"`
	Instructor          User    `gorm:"foreignKey:InstructorID" json:"instructor,omitempty"`
	TotalQuotaHours     float64 `json:"total_quota_hours"`
	RemainingQuotaHours float64 `json:"remaining_quota_hours"`
	WhatsApp            string  `json:"whatsapp"`
	Gender              string  `gorm:"type:enum('male','female');default:'male'" json:"gender"`
	MeetingPoint        string  `json:"meeting_point"`
	IsActive            bool    `gorm:"default:true" json:"is_active"`
}

// StudentSession tracks learning session time and quota deduction.
type StudentSession struct {
	InstructorBase
	StudentID     uint       `json:"student_id"`
	InstructorID  uint       `json:"instructor_id"`
	CheckInTime   time.Time  `json:"check_in_time"`
	CheckOutTime  *time.Time `json:"check_out_time"`
	DeductedHours float64    `json:"deducted_hours"`
	Latitude      float64    `json:"latitude"`
	Longitude     float64    `json:"longitude"`
	Notes         string     `gorm:"type:text" json:"notes"`
	Student       Student    `gorm:"foreignKey:StudentID" json:"student"`
}

// LearningPlan is the schedule created by an instructor.
type LearningPlan struct {
	InstructorBase
	InstructorID  uint      `json:"instructor_id"`
	StudentID     uint      `json:"student_id"`
	Student       Student   `gorm:"foreignKey:StudentID" json:"student"`
	ScheduledDate time.Time `json:"scheduled_date"`
	StartTime     string    `json:"start_time"`
	EndTime       string    `json:"end_time"`
	Status        string    `gorm:"type:enum('planned','completed','cancelled');default:'planned'" json:"status"`
	ReminderSent  bool      `gorm:"default:false" json:"reminder_sent"`
}
