package handlers

import (
	"fmt"
	"net/http"
	"time"

	"field-attendance-system/database"
	"field-attendance-system/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func GetAllRecords(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	// Get current manager
	var manager models.User
	if err := database.DB.First(&manager, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Manager tidak ditemukan"})
		return
	}

	var attendances []models.Attendance

	if manager.IsSuperAdmin {
		// Super admin can see all attendance records
		if result := database.DB.Preload("User").Order("clock_in_time DESC").Find(&attendances); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch records"})
			return
		}
	} else {
		// Regular manager: only see attendance from employees in their assigned offices
		var officeIDs []uint
		database.DB.Model(&models.ManagerOffice{}).
			Where("manager_id = ?", userID).
			Pluck("office_id", &officeIDs)

		if len(officeIDs) == 0 {
			c.JSON(http.StatusOK, gin.H{"data": []models.Attendance{}})
			return
		}

		// Get user IDs from assigned offices
		var userIDs []uint
		database.DB.Model(&models.User{}).
			Where("office_id IN ?", officeIDs).
			Pluck("id", &userIDs)

		if len(userIDs) == 0 {
			c.JSON(http.StatusOK, gin.H{"data": []models.Attendance{}})
			return
		}

		if result := database.DB.Preload("User").Where("user_id IN ?", userIDs).Order("clock_in_time DESC").Find(&attendances); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch records"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": attendances})
}

func GetAllLeaveRequests(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	// Get current manager
	var manager models.User
	if err := database.DB.First(&manager, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Manager tidak ditemukan"})
		return
	}

	var leaves []models.LeaveRequest

	if manager.IsSuperAdmin {
		// Super admin can see all leave requests
		if result := database.DB.Preload("User").Order("created_at DESC").Find(&leaves); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leave requests"})
			return
		}
	} else {
		// Regular manager: only see leave requests from employees in their assigned offices
		var officeIDs []uint
		database.DB.Model(&models.ManagerOffice{}).
			Where("manager_id = ?", userID).
			Pluck("office_id", &officeIDs)

		if len(officeIDs) == 0 {
			c.JSON(http.StatusOK, gin.H{"data": []models.LeaveRequest{}})
			return
		}

		// Get user IDs from assigned offices
		var userIDs []uint
		database.DB.Model(&models.User{}).
			Where("office_id IN ?", officeIDs).
			Pluck("id", &userIDs)

		if len(userIDs) == 0 {
			c.JSON(http.StatusOK, gin.H{"data": []models.LeaveRequest{}})
			return
		}

		if result := database.DB.Preload("User").Where("user_id IN ?", userIDs).Order("created_at DESC").Find(&leaves); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leave requests"})
			return
		}
	}

	// Create response with user data included
	type LeaveResponse struct {
		ID        uint      `json:"id"`
		UserID    uint      `json:"user_id"`
		StartDate time.Time `json:"start_date"`
		EndDate   time.Time `json:"end_date"`
		Reason    string    `json:"reason"`
		Status    string    `json:"status"`
		User      struct {
			ID       uint   `json:"id"`
			Username string `json:"username"`
			FullName string `json:"full_name"`
		} `json:"user"`
	}

	var response []LeaveResponse
	for _, leave := range leaves {
		lr := LeaveResponse{
			ID:        leave.ID,
			UserID:    leave.UserID,
			StartDate: leave.StartDate,
			EndDate:   leave.EndDate,
			Reason:    leave.Reason,
			Status:    leave.Status,
		}
		lr.User.ID = leave.User.ID
		lr.User.Username = leave.User.Username
		lr.User.FullName = leave.User.FullName
		response = append(response, lr)
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

type UpdateLeaveInput struct {
	Status string `json:"status" binding:"required,oneof=approved rejected"`
}

func UpdateLeaveStatus(c *gin.Context) {
	id := c.Param("id")
	var input UpdateLeaveInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var leave models.LeaveRequest
	if result := database.DB.First(&leave, id); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Leave request not found"})
		return
	}

	leave.Status = input.Status
	if result := database.DB.Save(&leave); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Leave status updated", "data": leave})
}

func GetAllEmployees(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	// Get current manager
	var manager models.User
	if err := database.DB.First(&manager, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Manager tidak ditemukan"})
		return
	}

	var users []models.User

	// Super admin can see all users
	if manager.IsSuperAdmin {
		if result := database.DB.Preload("Office").Find(&users); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch employees"})
			return
		}
	} else {
		// Regular manager: only see employees from their assigned offices
		var officeIDs []uint
		database.DB.Model(&models.ManagerOffice{}).
			Where("manager_id = ?", userID).
			Pluck("office_id", &officeIDs)

		if len(officeIDs) == 0 {
			// No offices assigned, return empty list
			c.JSON(http.StatusOK, gin.H{"data": []models.User{}})
			return
		}

		if result := database.DB.Preload("Office").Where("office_id IN ?", officeIDs).Find(&users); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch employees"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": users})
}

type CreateEmployeeInput struct {
	Username string `json:"username" binding:"required"`
	FullName string `json:"full_name"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required,oneof=employee manager instructor"`
	OfficeID *uint  `json:"office_id"`
}

func CreateEmployee(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var input CreateEmployeeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if username already exists
	var existingUser models.User
	if result := database.DB.Where("username = ?", input.Username).First(&existingUser); result.Error == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}

	// Get the manager to check their offices
	var manager models.User
	if err := database.DB.First(&manager, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Manager tidak ditemukan"})
		return
	}

	// Determine which office to assign the employee to
	var officeID *uint
	if input.OfficeID != nil {
		// Validate that the manager has access to this office (or is super admin)
		if manager.IsSuperAdmin {
			// Super admin can assign to any office
			var office models.OfficeLocation
			if err := database.DB.Where("id = ? AND is_active = ?", *input.OfficeID, true).First(&office).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Kantor tidak ditemukan"})
				return
			}
			officeID = input.OfficeID
		} else {
			// Regular manager must be assigned to this office
			var count int64
			database.DB.Model(&models.ManagerOffice{}).Where("manager_id = ? AND office_id = ?", userID, *input.OfficeID).Count(&count)
			if count == 0 {
				c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak dapat assign karyawan ke kantor yang tidak di-assign ke Anda"})
				return
			}
			officeID = input.OfficeID
		}
	} else {
		// Auto-assign to manager's first office
		var managerOffice models.ManagerOffice
		if err := database.DB.Where("manager_id = ?", userID).First(&managerOffice).Error; err == nil {
			officeID = &managerOffice.OfficeID
		}
	}

	// Hash password
	hashedPassword, err := hashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	user := models.User{
		Username:     input.Username,
		FullName:     input.FullName,
		PasswordHash: hashedPassword,
		Role:         input.Role,
		OfficeID:     officeID,
	}

	if result := database.DB.Create(&user); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create employee"})
		return
	}

	// Preload office for response
	database.DB.Preload("Office").First(&user, user.ID)

	c.JSON(http.StatusCreated, gin.H{"message": "Employee created successfully", "data": user})
}

type UpdateEmployeeInput struct {
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Password string `json:"password"`
	Role     string `json:"role" binding:"omitempty,oneof=employee manager instructor"`
	OfficeID *uint  `json:"office_id"`
}

func UpdateEmployee(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	id := c.Param("id")
	var input UpdateEmployeeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the manager to check permissions
	var manager models.User
	if err := database.DB.First(&manager, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Manager tidak ditemukan"})
		return
	}

	var user models.User
	if result := database.DB.First(&user, id); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}

	// Update username if provided
	if input.Username != "" {
		// Check if new username already exists
		var existingUser models.User
		if result := database.DB.Where("username = ? AND id != ?", input.Username, id).First(&existingUser); result.Error == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
			return
		}
		user.Username = input.Username
	}

	// Update full name if provided
	if input.FullName != "" {
		user.FullName = input.FullName
	}

	// Update password if provided
	if input.Password != "" {
		hashedPassword, err := hashPassword(input.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		user.PasswordHash = hashedPassword
	}

	// Update role if provided
	if input.Role != "" {
		user.Role = input.Role
	}

	// Update office_id if provided
	if input.OfficeID != nil {
		if manager.IsSuperAdmin {
			// Super admin can assign to any office
			var office models.OfficeLocation
			if err := database.DB.Where("id = ? AND is_active = ?", *input.OfficeID, true).First(&office).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Kantor tidak ditemukan"})
				return
			}
			user.OfficeID = input.OfficeID
		} else {
			// Regular manager must be assigned to this office
			var count int64
			database.DB.Model(&models.ManagerOffice{}).Where("manager_id = ? AND office_id = ?", userID, *input.OfficeID).Count(&count)
			if count == 0 {
				c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak dapat assign karyawan ke kantor yang tidak di-assign ke Anda"})
				return
			}
			user.OfficeID = input.OfficeID
		}
	}

	if result := database.DB.Save(&user); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update employee"})
		return
	}

	// Preload office for response
	database.DB.Preload("Office").First(&user, user.ID)

	c.JSON(http.StatusOK, gin.H{"message": "Employee updated successfully", "data": user})
}

func DeleteEmployee(c *gin.Context) {
	id := c.Param("id")

	var user models.User
	if result := database.DB.First(&user, id); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}

	// Prevent deleting managers/super admins through this endpoint
	if user.Role == "manager" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak dapat menghapus akun manager"})
		return
	}

	// Use transaction to delete associated records first, then the user
	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete employee"})
		return
	}

	// Delete associated attendance records
	if err := tx.Where("user_id = ?", user.ID).Delete(&models.Attendance{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete employee attendance records"})
		return
	}

	// Delete associated leave requests
	if err := tx.Where("user_id = ?", user.ID).Delete(&models.LeaveRequest{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete employee leave requests"})
		return
	}

	// Delete associated manager office assignments (in case role was changed)
	if err := tx.Where("manager_id = ?", user.ID).Delete(&models.ManagerOffice{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete employee office assignments"})
		return
	}

	// Now delete the user
	if err := tx.Delete(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete employee"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Employee deleted successfully"})
}

// Helper function to hash password (reused from auth.go)
func hashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

type EmployeeDailyStatus struct {
	UserID          uint       `json:"user_id"`
	FullName        string     `json:"full_name"`
	Username        string     `json:"username"`
	Status          string     `json:"status"` // "present_ontime", "present_late", "on_leave", "absent"
	ClockInTime     *time.Time `json:"clock_in_time,omitempty"`
	ClockOutTime    *time.Time `json:"clock_out_time,omitempty"`
	WorkHours       *float64   `json:"work_hours,omitempty"`
	WorkHoursStatus string     `json:"work_hours_status"`
	MinutesLate     int        `json:"minutes_late,omitempty"`
	LeaveReason     string     `json:"leave_reason,omitempty"`
	LeaveStatus     string     `json:"leave_status,omitempty"`
}

func getWorkHoursStatus(workHours *float64, minimumHours float64) string {
	if workHours == nil {
		return "-"
	}
	if minimumHours <= 0 {
		minimumHours = 8 // fallback default
	}
	if *workHours >= minimumHours {
		return "Terpenuhi"
	}
	kurang := minimumHours - *workHours
	if kurang == float64(int(kurang)) {
		return fmt.Sprintf("Tidak Terpenuhi (Kurang %.0f jam)", kurang)
	}
	return fmt.Sprintf("Tidak Terpenuhi (Kurang %.1f jam)", kurang)
}

func GetDailyAttendanceDashboard(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	// Get current manager
	var manager models.User
	if err := database.DB.First(&manager, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Manager tidak ditemukan"})
		return
	}

	// Get employees based on manager's permissions
	var users []models.User
	if manager.IsSuperAdmin {
		// Super admin can see all employees
		if result := database.DB.Find(&users); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch employees"})
			return
		}
	} else {
		// Regular manager: only see employees from their assigned offices
		var officeIDs []uint
		database.DB.Model(&models.ManagerOffice{}).
			Where("manager_id = ?", userID).
			Pluck("office_id", &officeIDs)

		if len(officeIDs) == 0 {
			// No offices assigned, return empty dashboard
			c.JSON(http.StatusOK, gin.H{"data": []EmployeeDailyStatus{}, "summary": map[string]int{
				"total": 0, "present_ontime": 0, "present_late": 0, "on_leave": 0, "absent": 0,
			}})
			return
		}

		if result := database.DB.Where("office_id IN ?", officeIDs).Find(&users); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch employees"})
			return
		}
	}

	// Get today's date range
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())

	// Get all today's attendance records
	var attendances []models.Attendance
	database.DB.Where("clock_in_time BETWEEN ? AND ?", startOfDay, endOfDay).Find(&attendances)

	// Create map for quick lookup
	attendanceMap := make(map[uint]models.Attendance)
	for _, att := range attendances {
		attendanceMap[att.UserID] = att
	}

	// Get all today's approved leave requests
	var leaves []models.LeaveRequest
	database.DB.Where("? BETWEEN start_date AND end_date", startOfDay).Find(&leaves)

	// Create map for leave lookup
	leaveMap := make(map[uint]models.LeaveRequest)
	for _, leave := range leaves {
		leaveMap[leave.UserID] = leave
	}

	// Build status for each employee
	var dailyStatus []EmployeeDailyStatus
	var summary = map[string]int{
		"total":          len(users),
		"present_ontime": 0,
		"present_late":   0,
		"on_leave":       0,
		"absent":         0,
	}

	for _, user := range users {
		status := EmployeeDailyStatus{
			UserID:   user.ID,
			FullName: user.FullName,
			Username: user.Username,
		}

		// Check if on leave
		if leave, hasLeave := leaveMap[user.ID]; hasLeave {
			status.Status = "on_leave"
			status.LeaveReason = leave.Reason
			status.LeaveStatus = leave.Status
			summary["on_leave"]++
		} else if att, hasAttendance := attendanceMap[user.ID]; hasAttendance {
			// Has clocked in
			if att.IsLate {
				status.Status = "present_late"
				status.MinutesLate = att.MinutesLate
				summary["present_late"]++
			} else {
				status.Status = "present_ontime"
				summary["present_ontime"]++
			}
			status.ClockInTime = &att.ClockInTime
			status.ClockOutTime = att.ClockOutTime
			status.WorkHours = att.WorkHours
			status.WorkHoursStatus = getWorkHoursStatus(att.WorkHours, manager.MinimumWorkHours)
		} else {
			// No attendance and no leave
			status.Status = "absent"
			summary["absent"]++
		}

		dailyStatus = append(dailyStatus, status)
	}

	minHours := manager.MinimumWorkHours
	if minHours <= 0 {
		minHours = 8
	}

	c.JSON(http.StatusOK, gin.H{
		"data":               dailyStatus,
		"summary":            summary,
		"minimum_work_hours": minHours,
	})
}

func GetPendingClockIns(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	// Get current manager
	var manager models.User
	if err := database.DB.First(&manager, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Manager tidak ditemukan"})
		return
	}

	var attendances []models.Attendance

	if manager.IsSuperAdmin {
		// Super admin can see all pending clock-ins
		if result := database.DB.Preload("User").Where("status = ?", "pending").Order("clock_in_time DESC").Find(&attendances); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pending clock-ins"})
			return
		}
	} else {
		// Regular manager: only see pending clock-ins from employees in their assigned offices
		var officeIDs []uint
		database.DB.Model(&models.ManagerOffice{}).
			Where("manager_id = ?", userID).
			Pluck("office_id", &officeIDs)

		if len(officeIDs) == 0 {
			c.JSON(http.StatusOK, gin.H{"data": []models.Attendance{}})
			return
		}

		// Get user IDs from assigned offices
		var userIDs []uint
		database.DB.Model(&models.User{}).
			Where("office_id IN ?", officeIDs).
			Pluck("id", &userIDs)

		if len(userIDs) == 0 {
			c.JSON(http.StatusOK, gin.H{"data": []models.Attendance{}})
			return
		}

		if result := database.DB.Preload("User").Where("status = ? AND user_id IN ?", "pending", userIDs).Order("clock_in_time DESC").Find(&attendances); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pending clock-ins"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": attendances})
}

type UpdateClockInStatusInput struct {
	Status string `json:"status" binding:"required,oneof=approved rejected"`
}

func UpdateClockInStatus(c *gin.Context) {
	id := c.Param("id")
	var input UpdateClockInStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var attendance models.Attendance
	if result := database.DB.First(&attendance, id); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Clock-in record not found"})
		return
	}

	if attendance.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only pending clock-ins can be updated"})
		return
	}

	attendance.Status = input.Status
	if result := database.DB.Save(&attendance); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update clock-in status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Clock-in status updated successfully", "data": attendance})
}
