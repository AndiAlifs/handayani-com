package handlers

// Training sessions CRUD (admin tooling). Migrated from the FastAPI service.
// GET is any-authenticated; writes are manager-only (gated at the route).
//
// The four ai_* columns are OWNED by the Python /analyze endpoint. This service
// never writes them: create leaves them NULL, update touches only the 12
// non-AI columns. On read they surface as a nested aiAnalysis object (or null),
// matching the Angular Session interface and the old Pydantic wire shape.

import (
	"encoding/json"
	"net/http"

	"handayani-core/database"
	"handayani-core/models"

	"github.com/gin-gonic/gin"
)

type aiAnalysis struct {
	Strengths            []string `json:"strengths"`
	Weaknesses           []string `json:"weaknesses"`
	RecommendedNextFocus string   `json:"recommendedNextFocus"`
	UpsellRecommendation *string  `json:"upsellRecommendation"`
}

type sessionResponse struct {
	ID             uint        `json:"id"`
	StudentID      int         `json:"studentId"`
	StudentName    string      `json:"studentName"`
	InstructorID   int         `json:"instructorId"`
	InstructorName string      `json:"instructorName"`
	CourseID       int         `json:"courseId"`
	CourseName     string      `json:"courseName"`
	StartTime      string      `json:"startTime"`
	EndTime        string      `json:"endTime"`
	Status         string      `json:"status"`
	SessionNumber  int         `json:"sessionNumber"`
	TotalSessions  int         `json:"totalSessions"`
	RawNotes       *string     `json:"rawNotes"`
	AiAnalysis     *aiAnalysis `json:"aiAnalysis"`
}

// sessionInput captures a write payload. aiAnalysis is accepted only so the
// create/update response can echo it back (as the Python endpoints did); it is
// never persisted here.
type sessionInput struct {
	StudentID      int         `json:"studentId"`
	StudentName    string      `json:"studentName"`
	InstructorID   int         `json:"instructorId"`
	InstructorName string      `json:"instructorName"`
	CourseID       int         `json:"courseId"`
	CourseName     string      `json:"courseName"`
	StartTime      string      `json:"startTime"`
	EndTime        string      `json:"endTime"`
	Status         string      `json:"status"`
	SessionNumber  int         `json:"sessionNumber"`
	TotalSessions  int         `json:"totalSessions"`
	RawNotes       *string     `json:"rawNotes"`
	AiAnalysis     *aiAnalysis `json:"aiAnalysis"`
}

// parseJSONStrings unmarshals a JSON array column into a []string, always
// returning a non-nil slice (so empty serializes as [] not null), mirroring
// the Python _as_list helper.
func parseJSONStrings(v *string) []string {
	out := []string{}
	if v == nil || *v == "" {
		return out
	}
	_ = json.Unmarshal([]byte(*v), &out)
	if out == nil {
		out = []string{}
	}
	return out
}

// toSessionResponse builds the wire object, including the nested aiAnalysis
// exactly as _row_to_session did: present iff the analysis has real content;
// strengths/weaknesses always arrays; upsell passed through (null stays null).
// Note the emptiness test uses the *parsed* arrays — a stored empty array is
// the JSON string "[]", which is non-empty as a raw string, so testing the raw
// column would wrongly treat an empty analysis as present.
func toSessionResponse(s models.Session) sessionResponse {
	var analysis *aiAnalysis
	focus := ""
	if s.AINextFocus != nil {
		focus = *s.AINextFocus
	}
	strengths := parseJSONStrings(s.AIStrengths)
	weaknesses := parseJSONStrings(s.AIWeaknesses)
	if focus != "" || len(strengths) > 0 || len(weaknesses) > 0 {
		analysis = &aiAnalysis{
			Strengths:            strengths,
			Weaknesses:           weaknesses,
			RecommendedNextFocus: focus,
			UpsellRecommendation: s.AIUpsell,
		}
	}
	return sessionResponse{
		ID:             s.ID,
		StudentID:      s.StudentID,
		StudentName:    s.StudentName,
		InstructorID:   s.InstructorID,
		InstructorName: s.InstructorName,
		CourseID:       s.CourseID,
		CourseName:     s.CourseName,
		StartTime:      s.StartTime,
		EndTime:        s.EndTime,
		Status:         s.Status,
		SessionNumber:  s.SessionNumber,
		TotalSessions:  s.TotalSessions,
		RawNotes:       s.RawNotes,
		AiAnalysis:     analysis,
	}
}

// inputToResponse echoes a write payload back as the response (matching the
// Python create/update, which returned the bound object without re-reading).
func inputToResponse(id uint, in sessionInput) sessionResponse {
	return sessionResponse{
		ID:             id,
		StudentID:      in.StudentID,
		StudentName:    in.StudentName,
		InstructorID:   in.InstructorID,
		InstructorName: in.InstructorName,
		CourseID:       in.CourseID,
		CourseName:     in.CourseName,
		StartTime:      in.StartTime,
		EndTime:        in.EndTime,
		Status:         in.Status,
		SessionNumber:  in.SessionNumber,
		TotalSessions:  in.TotalSessions,
		RawNotes:       in.RawNotes,
		AiAnalysis:     in.AiAnalysis,
	}
}

// applyDefaults mirrors the Pydantic field defaults for omitted values.
func (in *sessionInput) applyDefaults() {
	if in.Status == "" {
		in.Status = "scheduled"
	}
	if in.SessionNumber == 0 {
		in.SessionNumber = 1
	}
	if in.TotalSessions == 0 {
		in.TotalSessions = 10
	}
}

var sessionWriteColumns = []string{
	"student_id", "student_name", "instructor_id", "instructor_name",
	"course_id", "course_name", "start_time", "end_time", "status",
	"session_number", "total_sessions", "raw_notes",
}

func ListSessions(c *gin.Context) {
	var rows []models.Session
	if err := database.DB.Model(&models.Session{}).
		Select("id, student_id, student_name, instructor_id, instructor_name, course_id, course_name, " +
			"DATE_FORMAT(start_time, '%Y-%m-%dT%H:%i:%s') AS start_time, " +
			"DATE_FORMAT(end_time, '%Y-%m-%dT%H:%i:%s') AS end_time, " +
			"status, session_number, total_sessions, raw_notes, " +
			"ai_strengths, ai_weaknesses, ai_recommended_next_focus, ai_upsell_recommendation").
		Order("start_time").
		Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list sessions"})
		return
	}
	out := make([]sessionResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, toSessionResponse(r))
	}
	c.JSON(http.StatusOK, out)
}

func CreateSession(c *gin.Context) {
	var in sessionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	in.applyDefaults()
	row := models.Session{
		StudentID:      in.StudentID,
		StudentName:    in.StudentName,
		InstructorID:   in.InstructorID,
		InstructorName: in.InstructorName,
		CourseID:       in.CourseID,
		CourseName:     in.CourseName,
		StartTime:      in.StartTime,
		EndTime:        in.EndTime,
		Status:         in.Status,
		SessionNumber:  in.SessionNumber,
		TotalSessions:  in.TotalSessions,
		RawNotes:       in.RawNotes,
	}
	// Insert only the non-AI columns; ai_* default to NULL.
	if err := database.DB.Select(sessionWriteColumns).Create(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}
	c.JSON(http.StatusCreated, inputToResponse(row.ID, in))
}

func UpdateSession(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	var in sessionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	in.applyDefaults()
	// Update only the 12 non-AI columns — the ai_* columns are owned by the
	// Python /analyze endpoint and must be preserved.
	if err := database.DB.Model(&models.Session{}).Where("id = ?", id).Updates(map[string]interface{}{
		"student_id":      in.StudentID,
		"student_name":    in.StudentName,
		"instructor_id":   in.InstructorID,
		"instructor_name": in.InstructorName,
		"course_id":       in.CourseID,
		"course_name":     in.CourseName,
		"start_time":      in.StartTime,
		"end_time":        in.EndTime,
		"status":          in.Status,
		"session_number":  in.SessionNumber,
		"total_sessions":  in.TotalSessions,
		"raw_notes":       in.RawNotes,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session"})
		return
	}
	c.JSON(http.StatusOK, inputToResponse(id, in))
}

func DeleteSession(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	if err := database.DB.Delete(&models.Session{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete session"})
		return
	}
	c.Status(http.StatusNoContent)
}
