package models

// Knowledge-base content entities migrated from the FastAPI service.
//
// Unlike the attendance models in this package, these serialize JSON in
// camelCase (json:"programType", "courseId", ...) to match the Angular models
// and the wire format the SPA already expects. Their tables are owned by
// backend/schema.sql + seed.sql (the AI service still reads/writes them), so
// they are deliberately NOT added to AutoMigrate — GORM maps to the existing
// tables via the explicit gorm column tags below.

// Course — courses table (PRD Epic 2).
type Course struct {
	ID              uint   `gorm:"column:id;primaryKey" json:"id"`
	Category        string `gorm:"column:category" json:"category"`
	ProgramType     string `gorm:"column:program_type" json:"programType"`
	Specifics       string `gorm:"column:specifics" json:"specifics"`
	Duration        string `gorm:"column:duration" json:"duration"`
	Price           int64  `gorm:"column:price" json:"price"`
	RegistrationFee int64  `gorm:"column:registration_fee" json:"registrationFee"`
	Remarks         string `gorm:"column:remarks" json:"remarks"`
}

func (Course) TableName() string { return "courses" }

// Mechanism — mechanisms table (PRD Epic 4). SortOrder is internal (ordering
// + auto-increment on create); it never appears on the wire.
type Mechanism struct {
	ID              uint   `gorm:"column:id;primaryKey" json:"id"`
	RequirementName string `gorm:"column:requirement_name" json:"requirementName"`
	IssuingBody     string `gorm:"column:issuing_body" json:"issuingBody"`
	Cost            int64  `gorm:"column:cost" json:"cost"`
	Notes           string `gorm:"column:notes" json:"notes"`
	SortOrder       int    `gorm:"column:sort_order" json:"-"`
}

func (Mechanism) TableName() string { return "mechanisms" }

// StudentCrm — students_crm table (CRM admin tooling). TableName is mandatory:
// GORM's default pluralizer would look for "student_crms".
type StudentCrm struct {
	ID            uint    `gorm:"column:id;primaryKey" json:"id"`
	Name          string  `gorm:"column:name" json:"name"`
	Phone         string  `gorm:"column:phone" json:"phone"`
	CourseID      int     `gorm:"column:course_id" json:"courseId"`
	CourseName    string  `gorm:"column:course_name" json:"courseName"`
	Status        string  `gorm:"column:status" json:"status"`
	ProgressScore int     `gorm:"column:progress_score" json:"progressScore"`
	Notes         string  `gorm:"column:notes" json:"notes"`
	// createdAt is projected as a 'YYYY-MM-DD' string by the read query; a
	// pointer so the POST echo can emit null (matching the Python default).
	CreatedAt *string `gorm:"column:created_at" json:"createdAt"`
}

func (StudentCrm) TableName() string { return "students_crm" }

// Session — sessions table (CRM tooling + AI analysis).
//
// start_time/end_time are held as strings: the read query projects them with
// DATE_FORMAT(...,'%Y-%m-%dT%H:%i:%s') to match the Python wire format exactly
// (a time.Time would marshal RFC3339+offset and diverge).
//
// The four AI* columns are owned by the Python /analyze endpoint and are never
// written here (see handlers/sessions.go). They are json:"-" because they
// surface only through the nested aiAnalysis object on reads.
type Session struct {
	ID             uint    `gorm:"column:id;primaryKey" json:"id"`
	StudentID      int     `gorm:"column:student_id" json:"studentId"`
	StudentName    string  `gorm:"column:student_name" json:"studentName"`
	InstructorID   int     `gorm:"column:instructor_id" json:"instructorId"`
	InstructorName string  `gorm:"column:instructor_name" json:"instructorName"`
	CourseID       int     `gorm:"column:course_id" json:"courseId"`
	CourseName     string  `gorm:"column:course_name" json:"courseName"`
	StartTime      string  `gorm:"column:start_time" json:"startTime"`
	EndTime        string  `gorm:"column:end_time" json:"endTime"`
	Status         string  `gorm:"column:status" json:"status"`
	SessionNumber  int     `gorm:"column:session_number" json:"sessionNumber"`
	TotalSessions  int     `gorm:"column:total_sessions" json:"totalSessions"`
	RawNotes       *string `gorm:"column:raw_notes" json:"rawNotes"`

	AIStrengths  *string `gorm:"column:ai_strengths" json:"-"`
	AIWeaknesses *string `gorm:"column:ai_weaknesses" json:"-"`
	AINextFocus  *string `gorm:"column:ai_recommended_next_focus" json:"-"`
	AIUpsell     *string `gorm:"column:ai_upsell_recommendation" json:"-"`
}

func (Session) TableName() string { return "sessions" }
