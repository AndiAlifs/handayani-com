package handlers

import (
	"math"
	"net/http"
	"strings"
	"time"

	"handayani-core/database"
	"handayani-core/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdjustQuotaInput struct {
	RemainingQuotaHours float64 `json:"remaining_quota_hours" binding:"required,gte=0"`
}

type UpdateLearningPlanInput struct {
	ScheduledDate string `json:"scheduled_date"`
	StartTime     string `json:"start_time"`
	EndTime       string `json:"end_time"`
	Status        string `json:"status"`
}

type StartStudentSessionInput struct {
	StudentID uint    `json:"student_id" binding:"required"`
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
}

type EndStudentSessionInput struct {
	SessionID          *uint      `json:"session_id"`
	StudentID          *uint      `json:"student_id"`
	Notes              string     `json:"notes"`
	CustomCheckInTime  *time.Time `json:"custom_check_in_time"`
	CustomCheckOutTime *time.Time `json:"custom_check_out_time"`
}

// ==================== STUDENT READ ====================

func GetStudents(c *gin.Context) {
	instructorID := c.MustGet("userID").(uint)
	activeFilter := c.DefaultQuery("active", "all")

	query := database.DB.Where("instructor_id = ?", instructorID)

	if activeFilter == "true" {
		query = query.Where("is_active = ?", true)
	} else if activeFilter == "false" {
		query = query.Where("is_active = ?", false)
	}

	var students []models.Student
	if err := query.Order("created_at DESC").Find(&students).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data murid"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": students})
}

// ==================== STUDENT SESSIONS ====================

func GetStudentSessions(c *gin.Context) {
	instructorID := c.MustGet("userID").(uint)
	studentID := c.Param("id")

	// Verify student belongs to this instructor
	var student models.Student
	if err := database.DB.Where("id = ? AND instructor_id = ?", studentID, instructorID).First(&student).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Murid tidak ditemukan"})
		return
	}

	var sessions []models.StudentSession
	if err := database.DB.
		Preload("Student").
		Where("student_id = ? AND instructor_id = ?", studentID, instructorID).
		Order("check_in_time DESC").
		Find(&sessions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data sesi"})
		return
	}

	// Compute summary
	var totalSessions int
	var totalHours float64
	for _, s := range sessions {
		if s.CheckOutTime != nil {
			totalSessions++
			totalHours += s.DeductedHours
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":           sessions,
		"student":        student,
		"total_sessions": totalSessions,
		"total_hours":    roundTo2(totalHours),
	})
}

func StartStudentSession(c *gin.Context) {
	instructorID := c.MustGet("userID").(uint)

	var input StartStudentSessionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var student models.Student
	if err := database.DB.Where("id = ? AND instructor_id = ?", input.StudentID, instructorID).First(&student).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Murid tidak valid untuk instruktur ini"})
		return
	}

	if !student.IsActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Murid sudah tidak aktif"})
		return
	}

	if student.RemainingQuotaHours <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kuota murid sudah habis"})
		return
	}

	var activeCount int64
	database.DB.Model(&models.StudentSession{}).
		Where("instructor_id = ? AND check_out_time IS NULL", instructorID).
		Count(&activeCount)
	if activeCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Masih ada sesi aktif, selesaikan terlebih dahulu"})
		return
	}

	session := models.StudentSession{
		StudentID:    input.StudentID,
		InstructorID: instructorID,
		CheckInTime:  time.Now(),
		Latitude:     input.Latitude,
		Longitude:    input.Longitude,
	}

	if err := database.DB.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai sesi murid"})
		return
	}

	// Auto-mark matching learning plan as completed
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.AddDate(0, 0, 1)
	database.DB.Model(&models.LearningPlan{}).
		Where("instructor_id = ? AND student_id = ? AND scheduled_date >= ? AND scheduled_date < ? AND status = ?",
			instructorID, input.StudentID, today, tomorrow, "planned").
		Update("status", "completed")

	database.DB.Preload("Student").First(&session, session.ID)
	c.JSON(http.StatusCreated, gin.H{"message": "Sesi murid dimulai", "data": session})
}

func EndStudentSession(c *gin.Context) {
	instructorID := c.MustGet("userID").(uint)

	var input EndStudentSessionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var session models.StudentSession
	sessionQuery := database.DB.Where("instructor_id = ? AND check_out_time IS NULL", instructorID)
	if input.SessionID != nil {
		sessionQuery = sessionQuery.Where("id = ?", *input.SessionID)
	}
	if input.StudentID != nil {
		sessionQuery = sessionQuery.Where("student_id = ?", *input.StudentID)
	}
	if err := sessionQuery.Order("check_in_time DESC").First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sesi aktif tidak ditemukan"})
		return
	}

	checkOut := time.Now()
	if input.CustomCheckOutTime != nil {
		checkOut = *input.CustomCheckOutTime
	}

	checkIn := session.CheckInTime
	if input.CustomCheckInTime != nil {
		checkIn = *input.CustomCheckInTime
	}

	deducted := roundTo2(checkOut.Sub(checkIn).Hours())
	if deducted < 0 {
		deducted = 0
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if input.CustomCheckInTime != nil {
			session.CheckInTime = checkIn
		}
		session.CheckOutTime = &checkOut
		session.DeductedHours = deducted
		session.Notes = input.Notes
		if err := tx.Save(&session).Error; err != nil {
			return err
		}

		var student models.Student
		if err := tx.Where("id = ? AND instructor_id = ?", session.StudentID, instructorID).First(&student).Error; err != nil {
			return err
		}

		remaining := student.RemainingQuotaHours - deducted
		if remaining < 0 {
			remaining = 0
		}
		student.RemainingQuotaHours = roundTo2(remaining)

		// Auto-archive if quota is depleted
		if student.RemainingQuotaHours <= 0 {
			student.IsActive = false
		}

		return tx.Save(&student).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyelesaikan sesi murid"})
		return
	}

	database.DB.Preload("Student").First(&session, session.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Sesi murid selesai", "data": session})
}

func GetActiveStudentSession(c *gin.Context) {
	instructorID := c.MustGet("userID").(uint)

	var session models.StudentSession
	err := database.DB.
		Preload("Student").
		Where("instructor_id = ? AND check_out_time IS NULL", instructorID).
		Order("check_in_time DESC").
		First(&session).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{"data": nil})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil sesi aktif"})
		return
	}

	activeHours := roundTo2(time.Since(session.CheckInTime).Hours())
	c.JSON(http.StatusOK, gin.H{"data": session, "active_hours": activeHours})
}

// ==================== LEARNING PLANS ====================

func GetLearningPlans(c *gin.Context) {
	instructorID := c.MustGet("userID").(uint)
	period := c.DefaultQuery("period", "month")

	now := time.Now()
	var startDate time.Time
	var endDate time.Time

	if period == "week" {
		weekdayOffset := int(now.Weekday())
		if weekdayOffset == 0 {
			weekdayOffset = 7
		}
		startDate = time.Date(now.Year(), now.Month(), now.Day()-weekdayOffset+1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 0, 7)
	} else {
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 1, 0)
	}

	if q := c.Query("start_date"); q != "" {
		if parsed, err := time.Parse("2006-01-02", q); err == nil {
			startDate = parsed
		}
	}
	if q := c.Query("end_date"); q != "" {
		if parsed, err := time.Parse("2006-01-02", q); err == nil {
			endDate = parsed.Add(24 * time.Hour)
		}
	}

	var plans []models.LearningPlan
	if err := database.DB.
		Preload("Student").
		Where("instructor_id = ? AND scheduled_date >= ? AND scheduled_date < ?", instructorID, startDate, endDate).
		Order("scheduled_date ASC").
		Order("start_time ASC").
		Find(&plans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil jadwal belajar"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": plans})
}

type InstructorUpdatePlanInput struct {
	Status string `json:"status" binding:"required"`
}

func UpdateLearningPlan(c *gin.Context) {
	instructorID := c.MustGet("userID").(uint)
	planID := c.Param("id")

	var plan models.LearningPlan
	if err := database.DB.Where("id = ? AND instructor_id = ?", planID, instructorID).First(&plan).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Jadwal tidak ditemukan"})
		return
	}

	var input InstructorUpdatePlanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Status != "cancelled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instruktur hanya dapat mengubah status ke 'cancelled'. Status 'selesai' ditetapkan otomatis melalui sesi aktif"})
		return
	}
	plan.Status = input.Status

	if err := database.DB.Save(&plan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui jadwal"})
		return
	}

	database.DB.Preload("Student").First(&plan, plan.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Status jadwal berhasil diperbarui", "data": plan})
}

// ==================== QUOTA PRESETS ====================

func GetQuotaPresets(c *gin.Context) {
	var setting models.SystemSettings
	if err := database.DB.Where("setting_key = ?", models.SettingQuotaPresetOptions).First(&setting).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"presets": []string{"8", "10"}})
		return
	}

	presets := strings.Split(setting.SettingValue, ",")
	for i := range presets {
		presets[i] = strings.TrimSpace(presets[i])
	}

	c.JSON(http.StatusOK, gin.H{"presets": presets})
}

// ==================== HELPERS ====================

func roundTo2(value float64) float64 {
	return math.Round(value*100) / 100
}

// normalizeWhatsApp converts local Indonesian phone numbers to international format.
func normalizeWhatsApp(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "+", "")

	if strings.HasPrefix(phone, "0") {
		phone = "62" + phone[1:]
	}

	if !strings.HasPrefix(phone, "62") {
		phone = "62" + phone
	}

	return phone
}
