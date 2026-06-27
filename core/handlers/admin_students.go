package handlers

import (
	"fmt"
	"math"
	"net/http"
	"time"

	"handayani-core/database"
	"handayani-core/models"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// ==================== INPUT STRUCTS ====================

type AdminCreateStudentInput struct {
	Name            string  `json:"name" binding:"required"`
	InstructorID    uint    `json:"instructor_id" binding:"required"`
	TotalQuotaHours float64 `json:"total_quota_hours" binding:"required,gt=0"`
	WhatsApp        string  `json:"whatsapp" binding:"required"`
	Gender          string  `json:"gender" binding:"required"`
	MeetingPoint    string  `json:"meeting_point"`

	InitialScheduleDate string `json:"initial_schedule_date"`
	InitialStartTime    string `json:"initial_start_time"`
	InitialEndTime      string `json:"initial_end_time"`
}

type AdminUpdateStudentInput struct {
	Name         string `json:"name"`
	WhatsApp     string `json:"whatsapp"`
	Gender       string `json:"gender"`
	MeetingPoint string `json:"meeting_point"`
	InstructorID uint   `json:"instructor_id"`
}

type AdminReassignStudentInput struct {
	InstructorID uint `json:"instructor_id" binding:"required"`
}

type AdminCreateLearningPlanInput struct {
	StudentID     uint   `json:"student_id" binding:"required"`
	ScheduledDate string `json:"scheduled_date" binding:"required"`
	StartTime     string `json:"start_time" binding:"required"`
	EndTime       string `json:"end_time" binding:"required"`
	Status        string `json:"status"`
}

type AdminBulkCreateLearningPlanInput struct {
	StudentID  uint   `json:"student_id" binding:"required"`
	DaysOfWeek []int  `json:"days_of_week" binding:"required,min=1"`
	StartTime  string `json:"start_time" binding:"required"`
	EndTime    string `json:"end_time" binding:"required"`
	FromDate   string `json:"from_date" binding:"required"`
	ToDate     string `json:"to_date" binding:"required"`
	Force      bool   `json:"force"`
}

// ==================== STUDENT HANDLERS ====================

func AdminCreateStudent(c *gin.Context) {
	var input AdminCreateStudentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Gender != "male" && input.Gender != "female" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gender harus 'male' atau 'female'"})
		return
	}

	var instructor models.User
	if err := database.DB.Where("id = ? AND role = ?", input.InstructorID, "instructor").First(&instructor).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instruktur tidak ditemukan"})
		return
	}

	student := models.Student{
		Name:                input.Name,
		InstructorID:        input.InstructorID,
		TotalQuotaHours:     roundTo2(input.TotalQuotaHours),
		RemainingQuotaHours: roundTo2(input.TotalQuotaHours),
		WhatsApp:            normalizeWhatsApp(input.WhatsApp),
		Gender:              input.Gender,
		MeetingPoint:        input.MeetingPoint,
		IsActive:            true,
	}

	if err := database.DB.Create(&student).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat data murid"})
		return
	}

	if input.InitialScheduleDate != "" && input.InitialStartTime != "" && input.InitialEndTime != "" {
		if scheduledDate, err := time.Parse("2006-01-02", input.InitialScheduleDate); err == nil {
			plan := models.LearningPlan{
				InstructorID:  input.InstructorID,
				StudentID:     student.ID,
				ScheduledDate: scheduledDate,
				StartTime:     input.InitialStartTime,
				EndTime:       input.InitialEndTime,
				Status:        "planned",
			}
			database.DB.Create(&plan)
		}
	}

	database.DB.Preload("Instructor").First(&student, student.ID)
	c.JSON(http.StatusCreated, gin.H{"message": "Murid berhasil dibuat", "data": student})
}

func AdminGetStudents(c *gin.Context) {
	instructorFilter := c.Query("instructor_id")
	isActiveFilter := c.Query("is_active")
	search := c.Query("q")

	query := database.DB.Preload("Instructor")

	if instructorFilter != "" {
		query = query.Where("instructor_id = ?", instructorFilter)
	}
	if isActiveFilter == "true" {
		query = query.Where("is_active = ?", true)
	} else if isActiveFilter == "false" {
		query = query.Where("is_active = ?", false)
	}
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR whatsapp LIKE ?", like, like)
	}

	var students []models.Student
	if err := query.Order("created_at DESC").Find(&students).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data murid"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": students})
}

func AdminUpdateStudent(c *gin.Context) {
	id := c.Param("id")

	var input AdminUpdateStudentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var student models.Student
	if err := database.DB.First(&student, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Murid tidak ditemukan"})
		return
	}

	if input.Name != "" {
		student.Name = input.Name
	}
	if input.WhatsApp != "" {
		student.WhatsApp = normalizeWhatsApp(input.WhatsApp)
	}
	if input.Gender == "male" || input.Gender == "female" {
		student.Gender = input.Gender
	}
	student.MeetingPoint = input.MeetingPoint

	if input.InstructorID != 0 {
		var instructor models.User
		if err := database.DB.Where("id = ? AND role = ?", input.InstructorID, "instructor").First(&instructor).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Instruktur tidak ditemukan"})
			return
		}
		student.InstructorID = input.InstructorID
	}

	if err := database.DB.Save(&student).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui data murid"})
		return
	}

	database.DB.Preload("Instructor").First(&student, student.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Data murid berhasil diperbarui", "data": student})
}

func AdminAdjustStudentQuota(c *gin.Context) {
	id := c.Param("id")

	var input AdjustQuotaInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var student models.Student
	if err := database.DB.First(&student, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Murid tidak ditemukan"})
		return
	}

	student.RemainingQuotaHours = roundTo2(*input.RemainingQuotaHours)
	if student.RemainingQuotaHours > student.TotalQuotaHours {
		student.TotalQuotaHours = student.RemainingQuotaHours
	}
	if student.RemainingQuotaHours > 0 && !student.IsActive {
		student.IsActive = true
	}

	if err := database.DB.Save(&student).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui kuota murid"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Kuota murid berhasil diperbarui", "data": student})
}

func AdminArchiveStudent(c *gin.Context) {
	id := c.Param("id")

	var student models.Student
	if err := database.DB.First(&student, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Murid tidak ditemukan"})
		return
	}

	student.IsActive = !student.IsActive

	if err := database.DB.Save(&student).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengarsipkan murid"})
		return
	}

	status := "diarsipkan"
	if student.IsActive {
		status = "diaktifkan kembali"
	}

	c.JSON(http.StatusOK, gin.H{"message": "Murid berhasil " + status, "data": student})
}

func AdminReassignStudent(c *gin.Context) {
	id := c.Param("id")

	var input AdminReassignStudentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var instructor models.User
	if err := database.DB.Where("id = ? AND role = ?", input.InstructorID, "instructor").First(&instructor).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instruktur tidak ditemukan"})
		return
	}

	var student models.Student
	if err := database.DB.First(&student, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Murid tidak ditemukan"})
		return
	}

	var activeSessionCount int64
	database.DB.Model(&models.StudentSession{}).
		Where("student_id = ? AND check_out_time IS NULL", student.ID).
		Count(&activeSessionCount)
	if activeSessionCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Murid masih memiliki sesi aktif, selesaikan sesi terlebih dahulu"})
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		student.InstructorID = input.InstructorID
		if err := tx.Save(&student).Error; err != nil {
			return err
		}

		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return tx.Model(&models.LearningPlan{}).
			Where("student_id = ? AND scheduled_date >= ? AND status = ?", student.ID, today, "planned").
			Update("instructor_id", input.InstructorID).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memindahkan murid"})
		return
	}

	database.DB.Preload("Instructor").First(&student, student.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Murid berhasil dipindahkan ke instruktur baru", "data": student})
}

func AdminGetStudentSessions(c *gin.Context) {
	studentID := c.Param("id")

	var student models.Student
	if err := database.DB.First(&student, studentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Murid tidak ditemukan"})
		return
	}

	var sessions []models.StudentSession
	if err := database.DB.
		Preload("Student").
		Where("student_id = ?", studentID).
		Order("check_in_time DESC").
		Find(&sessions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data sesi"})
		return
	}

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

// ==================== LEARNING PLAN HANDLERS ====================

func AdminCreateLearningPlan(c *gin.Context) {
	var input AdminCreateLearningPlanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var student models.Student
	if err := database.DB.First(&student, input.StudentID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Murid tidak ditemukan"})
		return
	}

	scheduledDate, err := time.Parse("2006-01-02", input.ScheduledDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format scheduled_date harus YYYY-MM-DD"})
		return
	}
	if _, err := time.Parse("15:04", input.StartTime); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format start_time harus HH:MM"})
		return
	}
	if _, err := time.Parse("15:04", input.EndTime); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format end_time harus HH:MM"})
		return
	}

	status := input.Status
	if status == "" {
		status = "planned"
	}
	if status != "planned" && status != "completed" && status != "cancelled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status tidak valid"})
		return
	}

	plan := models.LearningPlan{
		InstructorID:  student.InstructorID,
		StudentID:     input.StudentID,
		ScheduledDate: scheduledDate,
		StartTime:     input.StartTime,
		EndTime:       input.EndTime,
		Status:        status,
	}

	if err := database.DB.Create(&plan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat jadwal belajar"})
		return
	}

	database.DB.Preload("Student").First(&plan, plan.ID)
	c.JSON(http.StatusCreated, gin.H{"message": "Jadwal belajar berhasil dibuat", "data": plan})
}

func AdminGetLearningPlans(c *gin.Context) {
	instructorFilter := c.Query("instructor_id")
	studentFilter := c.Query("student_id")
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

	query := database.DB.Preload("Student").
		Where("scheduled_date >= ? AND scheduled_date < ?", startDate, endDate)

	if instructorFilter != "" {
		query = query.Where("instructor_id = ?", instructorFilter)
	}
	if studentFilter != "" {
		query = query.Where("student_id = ?", studentFilter)
	}

	var plans []models.LearningPlan
	if err := query.Order("scheduled_date ASC").Order("start_time ASC").Find(&plans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil jadwal belajar"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": plans})
}

func AdminUpdateLearningPlan(c *gin.Context) {
	planID := c.Param("id")

	var plan models.LearningPlan
	if err := database.DB.First(&plan, planID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Jadwal tidak ditemukan"})
		return
	}

	var input UpdateLearningPlanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.ScheduledDate != "" {
		scheduledDate, err := time.Parse("2006-01-02", input.ScheduledDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format scheduled_date harus YYYY-MM-DD"})
			return
		}
		plan.ScheduledDate = scheduledDate
	}
	if input.StartTime != "" {
		if _, err := time.Parse("15:04", input.StartTime); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format start_time harus HH:MM"})
			return
		}
		plan.StartTime = input.StartTime
	}
	if input.EndTime != "" {
		if _, err := time.Parse("15:04", input.EndTime); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format end_time harus HH:MM"})
			return
		}
		plan.EndTime = input.EndTime
	}
	if input.Status != "" {
		if input.Status != "planned" && input.Status != "completed" && input.Status != "cancelled" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Status tidak valid"})
			return
		}
		plan.Status = input.Status
	}

	if err := database.DB.Save(&plan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui jadwal"})
		return
	}

	database.DB.Preload("Student").First(&plan, plan.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Jadwal berhasil diperbarui", "data": plan})
}

func AdminDeleteLearningPlan(c *gin.Context) {
	planID := c.Param("id")

	var plan models.LearningPlan
	if err := database.DB.First(&plan, planID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Jadwal tidak ditemukan"})
		return
	}

	if err := database.DB.Delete(&plan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus jadwal"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Jadwal berhasil dihapus"})
}

func AdminBulkCreateLearningPlan(c *gin.Context) {
	var input AdminBulkCreateLearningPlanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var student models.Student
	if err := database.DB.First(&student, input.StudentID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Murid tidak ditemukan"})
		return
	}

	daySet := make(map[int]bool)
	for _, d := range input.DaysOfWeek {
		if d < 1 || d > 7 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Hari harus antara 1 (Senin) dan 7 (Minggu)"})
			return
		}
		daySet[d] = true
	}

	startT, err := time.Parse("15:04", input.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format start_time harus HH:MM"})
		return
	}
	endT, err := time.Parse("15:04", input.EndTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format end_time harus HH:MM"})
		return
	}
	sessionDurationHours := endT.Sub(startT).Hours()
	if sessionDurationHours <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jam selesai harus setelah jam mulai"})
		return
	}

	fromDate, err := time.Parse("2006-01-02", input.FromDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format from_date harus YYYY-MM-DD"})
		return
	}
	toDate, err := time.Parse("2006-01-02", input.ToDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format to_date harus YYYY-MM-DD"})
		return
	}
	if toDate.Before(fromDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "to_date harus setelah from_date"})
		return
	}
	if toDate.Sub(fromDate) > 365*2*24*time.Hour {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rentang tanggal terlalu besar (maksimal 2 tahun)"})
		return
	}

	var existing []models.LearningPlan
	database.DB.Where(
		"student_id = ? AND scheduled_date >= ? AND scheduled_date <= ?",
		input.StudentID, fromDate, toDate,
	).Find(&existing)

	existingKey := make(map[string]bool)
	for _, e := range existing {
		key := e.ScheduledDate.Format("2006-01-02") + "|" + e.StartTime
		existingKey[key] = true
	}

	var conflictDates []string
	var createDates []time.Time

	current := fromDate
	for !current.After(toDate) {
		goWeekday := int(current.Weekday())
		var ourDay int
		if goWeekday == 0 {
			ourDay = 7
		} else {
			ourDay = goWeekday
		}

		if daySet[ourDay] {
			key := current.Format("2006-01-02") + "|" + input.StartTime
			if existingKey[key] {
				conflictDates = append(conflictDates, current.Format("2006-01-02"))
			} else {
				createDates = append(createDates, current)
			}
		}
		current = current.AddDate(0, 0, 1)
	}

	if len(conflictDates) > 0 && !input.Force {
		c.JSON(http.StatusConflict, gin.H{
			"error":        fmt.Sprintf("Terdapat %d jadwal yang bentrok", len(conflictDates)),
			"conflicts":    conflictDates,
			"would_create": len(createDates),
		})
		return
	}

	// Limit sessions to what the student's remaining quota allows
	quotaLimited := 0
	if student.RemainingQuotaHours > 0 {
		maxByQuota := int(math.Floor(student.RemainingQuotaHours / sessionDurationHours))
		if maxByQuota < len(createDates) {
			quotaLimited = len(createDates) - maxByQuota
			createDates = createDates[:maxByQuota]
		}
	} else {
		quotaLimited = len(createDates)
		createDates = nil
	}

	created := 0
	for _, date := range createDates {
		plan := models.LearningPlan{
			InstructorID:  student.InstructorID,
			StudentID:     input.StudentID,
			ScheduledDate: time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location()),
			StartTime:     input.StartTime,
			EndTime:       input.EndTime,
			Status:        "planned",
		}
		if err := database.DB.Create(&plan).Error; err == nil {
			created++
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":       "Jadwal berulang berhasil dibuat",
		"created":       created,
		"skipped":       len(conflictDates),
		"quota_limited": quotaLimited,
	})
}

// ==================== INSTRUCTOR HANDLERS ====================

func AdminListInstructors(c *gin.Context) {
	var instructors []models.User
	// Preload Office so the office name surfaces (the RAG knowledge-sync renders
	// it into each instructor's KB chunk).
	if err := database.DB.Preload("Office").Where("role = ?", "instructor").Order("full_name ASC").Find(&instructors).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data instruktur"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": instructors})
}

type InstructorLoadRow struct {
	InstructorID     uint    `json:"instructor_id"`
	FullName         string  `json:"full_name"`
	Username         string  `json:"username"`
	ActiveStudents   int64   `json:"active_students"`
	TotalQuotaHours  float64 `json:"total_quota_hours"`
	RemQuotaHours    float64 `json:"remaining_quota_hours"`
	SessionsThisMonth int64  `json:"sessions_this_month"`
}

func AdminGetInstructorLoad(c *gin.Context) {
	var instructors []models.User
	if err := database.DB.Where("role = ?", "instructor").Order("full_name ASC").Find(&instructors).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data instruktur"})
		return
	}

	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthEnd := monthStart.AddDate(0, 1, 0)

	result := make([]InstructorLoadRow, 0, len(instructors))
	for _, inst := range instructors {
		row := InstructorLoadRow{
			InstructorID: inst.ID,
			FullName:     inst.FullName,
			Username:     inst.Username,
		}

		database.DB.Model(&models.Student{}).
			Where("instructor_id = ? AND is_active = ?", inst.ID, true).
			Count(&row.ActiveStudents)

		type quotaResult struct {
			TotalQuota float64
			RemQuota   float64
		}
		var qr quotaResult
		database.DB.Model(&models.Student{}).
			Select("COALESCE(SUM(total_quota_hours),0) as total_quota, COALESCE(SUM(remaining_quota_hours),0) as rem_quota").
			Where("instructor_id = ?", inst.ID).
			Scan(&qr)
		row.TotalQuotaHours = roundTo2(qr.TotalQuota)
		row.RemQuotaHours = roundTo2(qr.RemQuota)

		database.DB.Model(&models.StudentSession{}).
			Where("instructor_id = ? AND check_in_time >= ? AND check_in_time < ? AND check_out_time IS NOT NULL", inst.ID, monthStart, monthEnd).
			Count(&row.SessionsThisMonth)

		result = append(result, row)
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// ==================== EXCEL EXPORT ====================

var indonesianDays = map[string]string{
	"Monday": "Senin", "Tuesday": "Selasa", "Wednesday": "Rabu",
	"Thursday": "Kamis", "Friday": "Jumat", "Saturday": "Sabtu", "Sunday": "Minggu",
}

func AdminExportStudentRoster(c *gin.Context) {
	var students []models.Student
	if err := database.DB.Preload("Instructor").Order("instructor_id ASC, name ASC").Find(&students).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data murid"})
		return
	}

	f := excelize.NewFile()
	sheet := "Roster Murid"
	f.NewSheet(sheet)
	f.DeleteSheet("Sheet1")

	boldStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#D9E1F2"}, Pattern: 1},
	})
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#BDD7EE"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	subtotalStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Italic: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#EBF3FB"}, Pattern: 1},
	})

	cols := []string{"Nama Murid", "WhatsApp", "Gender", "Meeting Point", "Total Jam", "Sisa Jam", "Status", "Terdaftar"}
	colWidths := []float64{30, 20, 12, 25, 12, 12, 12, 22}

	row := 1

	type group struct {
		instructorName string
		students       []models.Student
	}
	groups := []group{}
	groupIdx := map[uint]int{}

	for _, s := range students {
		name := s.Instructor.FullName
		if name == "" {
			name = s.Instructor.Username
		}
		if idx, ok := groupIdx[s.InstructorID]; ok {
			groups[idx].students = append(groups[idx].students, s)
		} else {
			groupIdx[s.InstructorID] = len(groups)
			groups = append(groups, group{instructorName: name, students: []models.Student{s}})
		}
	}

	for _, g := range groups {
		// Instructor header row
		cell, _ := excelize.CoordinatesToCellName(1, row)
		f.SetCellValue(sheet, cell, "Instruktur: "+g.instructorName)
		endCell, _ := excelize.CoordinatesToCellName(len(cols), row)
		f.MergeCell(sheet, cell, endCell)
		f.SetCellStyle(sheet, cell, endCell, boldStyle)
		row++

		// Column header row
		for ci, h := range cols {
			c2, _ := excelize.CoordinatesToCellName(ci+1, row)
			f.SetCellValue(sheet, c2, h)
			f.SetCellStyle(sheet, c2, c2, headerStyle)
		}
		row++

		var sumTotal, sumRem float64
		for _, s := range g.students {
			status := "Alumni"
			if s.IsActive {
				status = "Aktif"
			}
			gender := "Laki-laki"
			if s.Gender == "female" {
				gender = "Perempuan"
			}
			dayName := indonesianDays[s.CreatedAt.Weekday().String()]
			createdStr := fmt.Sprintf("%s, %02d/%02d/%04d", dayName, s.CreatedAt.Day(), int(s.CreatedAt.Month()), s.CreatedAt.Year())

			vals := []interface{}{s.Name, s.WhatsApp, gender, s.MeetingPoint, s.TotalQuotaHours, s.RemainingQuotaHours, status, createdStr}
			for ci, v := range vals {
				c2, _ := excelize.CoordinatesToCellName(ci+1, row)
				f.SetCellValue(sheet, c2, v)
			}
			sumTotal += s.TotalQuotaHours
			sumRem += s.RemainingQuotaHours
			row++
		}

		// Subtotal row
		subtotalVals := []interface{}{"Subtotal", "", "", "", roundTo2(sumTotal), roundTo2(sumRem), "", ""}
		for ci, v := range subtotalVals {
			c2, _ := excelize.CoordinatesToCellName(ci+1, row)
			f.SetCellValue(sheet, c2, v)
			f.SetCellStyle(sheet, c2, c2, subtotalStyle)
		}
		row += 2 // blank row between groups
	}

	for ci, w := range colWidths {
		col, _ := excelize.ColumnNumberToName(ci + 1)
		f.SetColWidth(sheet, col, col, w)
	}

	filename := fmt.Sprintf("student-roster-%s.xlsx", time.Now().Format("2006-01-02"))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Cache-Control", "no-cache")

	if err := f.Write(c.Writer); err != nil {
		// Header already sent; nothing further to do
		_ = err
	}
}
