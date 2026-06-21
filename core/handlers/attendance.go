package handlers

import (
	"net/http"
	"strconv"
	"time"

	"handayani-core/database"
	"handayani-core/models"
	"handayani-core/utils"

	"github.com/gin-gonic/gin"
)

type ClockInInput struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type ClockOutInput struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func ClockIn(c *gin.Context) {
	var input ClockInInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.MustGet("userID").(uint)

	// Verify user exists
	var user models.User
	if result := database.DB.Preload("Office").First(&user, userID); result.Error != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User tidak ditemukan. Silakan login ulang."})
		return
	}

	if user.Role == "instructor" {
		clockInTime := time.Now()
		attendance := models.Attendance{
			UserID:      userID,
			ClockInTime: clockInTime,
			Latitude:    input.Latitude,
			Longitude:   input.Longitude,
			Status:      "approved",
			Distance:    0,
			IsLate:      false,
			MinutesLate: 0,
		}

		if result := database.DB.Create(&attendance); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save attendance record"})
			return
		}

		message := "Berhasil melakukan clock-in instruktur"
		if input.Latitude == 0 && input.Longitude == 0 {
			message += " - Tanpa Lokasi GPS"
		}

		c.JSON(http.StatusOK, gin.H{
			"message":         message,
			"data":            attendance,
			"distance_meters": 0,
			"status":          "approved",
			"outside_radius":  false,
			"is_late":         false,
			"minutes_late":    0,
			"office_used":     "",
		})
		return
	}

	// Find the manager who manages the employee's assigned office
	var managerID uint
	if user.OfficeID != nil {
		// Find manager via the manager_offices junction table
		var managerOffice models.ManagerOffice
		if err := database.DB.Where("office_id = ?", *user.OfficeID).First(&managerOffice).Error; err == nil {
			managerID = managerOffice.ManagerID
		}
	}

	// Fallback: if no manager found via office, use first manager
	if managerID == 0 {
		var manager models.User
		if err := database.DB.Where("role = ?", "manager").First(&manager).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Manager tidak ditemukan"})
			return
		}
		managerID = manager.ID
	}

	noLocation := input.Latitude == 0 && input.Longitude == 0

	// Check distance against ALL manager's offices (skip if no location provided)
	var closestOffice *models.OfficeLocation
	var minDistance float64 = -1
	var isWithinRadius bool = false

	if !noLocation {
		// **CRITICAL: Get ALL offices managed by the manager (up to 4)**
		var managerOffices []models.OfficeLocation
		database.DB.
			Joins("JOIN manager_offices ON manager_offices.office_id = office_locations.id").
			Where("manager_offices.manager_id = ? AND office_locations.is_active = ?", managerID, true).
			Find(&managerOffices)

		if len(managerOffices) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Lokasi kantor belum diatur. Silakan hubungi manajer Anda."})
			return
		}

		for i := range managerOffices {
			office := &managerOffices[i]
			distance := utils.CalculateDistance(
				input.Latitude,
				input.Longitude,
				office.Latitude,
				office.Longitude,
			)

			if minDistance == -1 || distance < minDistance {
				minDistance = distance
				closestOffice = office
			}

			if distance <= office.AllowedRadiusMeters {
				isWithinRadius = true
				closestOffice = office
				minDistance = distance
				break // Found a valid office, no need to check others
			}
		}
	}

	// Always approve, but track if outside radius via approvedOfficeID
	status := "approved"
	var approvedOfficeID *uint

	if isWithinRadius {
		approvedOfficeID = &closestOffice.ID
	}
	// If approvedOfficeID is nil, it means employee was outside all office radii or had no location

	// Calculate if employee is late
	isLate := false
	minutesLate := 0
	clockInTime := time.Now()

	if closestOffice != nil && closestOffice.ClockInTime != "" {
		// Parse office clock-in time (format: "HH:MM")
		officialTime, err := time.Parse("15:04", closestOffice.ClockInTime)
		if err == nil {
			// Get current time's hour and minute
			now := time.Now()
			actualTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())
			officialTimeToday := time.Date(now.Year(), now.Month(), now.Day(), officialTime.Hour(), officialTime.Minute(), 0, 0, now.Location())

			// Check if late
			if actualTime.After(officialTimeToday) {
				isLate = true
				minutesLate = int(actualTime.Sub(officialTimeToday).Minutes())
			}
		}
	}

	attendance := models.Attendance{
		UserID:           userID,
		ClockInTime:      clockInTime,
		Latitude:         input.Latitude,
		Longitude:        input.Longitude,
		Status:           status,
		Distance:         minDistance,
		IsLate:           isLate,
		MinutesLate:      minutesLate,
		ApprovedOfficeID: approvedOfficeID,
	}

	if result := database.DB.Create(&attendance); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save attendance record"})
		return
	}

	message := "Berhasil melakukan clock-in"
	if noLocation {
		message += " - Tanpa Lokasi GPS"
	} else if !isWithinRadius {
		message += " - Masuk diluar radius kantor"
	}
	if isLate {
		message += " - Terlambat " + strconv.Itoa(minutesLate) + " menit"
	}

	officeName := ""
	if closestOffice != nil {
		officeName = closestOffice.Name
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         message,
		"data":            attendance,
		"distance_meters": minDistance,
		"status":          status,
		"outside_radius":  !isWithinRadius,
		"is_late":         isLate,
		"minutes_late":    minutesLate,
		"office_used":     officeName,
	})
}

// ClockOut handles clock-out request with geolocation validation
func ClockOut(c *gin.Context) {
	var input ClockOutInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.MustGet("userID").(uint)

	// Get today's attendance record
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())

	var attendance models.Attendance
	result := database.DB.Where("user_id = ? AND clock_in_time BETWEEN ? AND ?", userID, startOfDay, endOfDay).First(&attendance)

	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Anda belum melakukan clock-in hari ini"})
		return
	}

	// Check if already clocked out
	if attendance.ClockOutTime != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Anda sudah melakukan clock-out hari ini"})
		return
	}

	// Record clock-out time and location
	clockOutTime := time.Now()

	// Calculate work hours (difference in hours as decimal)
	workDuration := clockOutTime.Sub(attendance.ClockInTime)
	workHours := workDuration.Hours()

	// Update attendance record
	attendance.ClockOutTime = &clockOutTime
	if input.Latitude != 0 || input.Longitude != 0 {
		attendance.LatitudeOut = &input.Latitude
		attendance.LongitudeOut = &input.Longitude
	}
	attendance.WorkHours = &workHours

	if err := database.DB.Save(&attendance).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan clock-out"})
		return
	}

	// Format work hours for display (e.g., "8.5 jam")
	hoursDisplay := strconv.FormatFloat(workHours, 'f', 2, 64)

	c.JSON(http.StatusOK, gin.H{
		"message":    "Berhasil melakukan clock-out",
		"data":       attendance,
		"work_hours": hoursDisplay,
	})
}

// GetTodayAttendance returns today's attendance record for the logged-in user
func GetTodayAttendance(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	// Get today's start and end time
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())

	var attendance models.Attendance
	result := database.DB.Where("user_id = ? AND clock_in_time BETWEEN ? AND ?", userID, startOfDay, endOfDay).First(&attendance)

	if result.Error != nil {
		// No attendance record for today
		c.JSON(http.StatusOK, gin.H{
			"data":    nil,
			"message": "Belum melakukan clock-in hari ini",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": attendance,
	})
}

// GetMyAttendanceHistory returns all attendance records for the logged-in employee
func GetMyAttendanceHistory(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	// Optional query parameters for filtering
	limit := c.DefaultQuery("limit", "50")
	offset := c.DefaultQuery("offset", "0")

	limitInt, _ := strconv.Atoi(limit)
	offsetInt, _ := strconv.Atoi(offset)

	var attendances []models.Attendance
	var total int64

	// Get total count
	database.DB.Model(&models.Attendance{}).Where("user_id = ?", userID).Count(&total)

	// Get records with preload of related office
	result := database.DB.
		Preload("ApprovedOffice").
		Where("user_id = ?", userID).
		Order("clock_in_time DESC").
		Limit(limitInt).
		Offset(offsetInt).
		Find(&attendances)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data absensi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   attendances,
		"total":  total,
		"limit":  limitInt,
		"offset": offsetInt,
	})
}
