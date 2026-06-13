package handlers

import (
	"field-attendance-system/database"
	"field-attendance-system/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetAllOffices - Get all offices (super admin) or manager's assigned offices
func GetAllOffices(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	var offices []models.OfficeLocation

	if user.IsSuperAdmin {
		// Super admin sees all offices
		database.DB.Where("is_active = ?", true).Find(&offices)
	} else if user.Role == "manager" {
		// Regular manager sees only assigned offices
		database.DB.
			Joins("JOIN manager_offices ON manager_offices.office_id = office_locations.id").
			Where("manager_offices.manager_id = ? AND office_locations.is_active = ?", userID, true).
			Find(&offices)
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": offices})
}

// CreateOffice - Any manager can create offices
func CreateOffice(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	// Any manager can create offices (not just super admin)
	if user.Role != "manager" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Hanya manager yang dapat membuat kantor"})
		return
	}

	var input struct {
		Name                string  `json:"name" binding:"required"`
		Address             string  `json:"address"`
		Latitude            float64 `json:"latitude" binding:"required"`
		Longitude           float64 `json:"longitude" binding:"required"`
		AllowedRadiusMeters float64 `json:"allowed_radius_meters" binding:"required,gt=0"`
		ClockInTime         string  `json:"clock_in_time" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if manager already has 4 offices (to prevent exceeding limit)
	var currentOfficeCount int64
	database.DB.Model(&models.ManagerOffice{}).Where("manager_id = ?", userID).Count(&currentOfficeCount)

	if currentOfficeCount >= 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Anda sudah memiliki 4 kantor (maksimal). Hapus salah satu untuk menambah yang baru."})
		return
	}

	office := models.OfficeLocation{
		Name:                input.Name,
		Address:             input.Address,
		Latitude:            input.Latitude,
		Longitude:           input.Longitude,
		AllowedRadiusMeters: input.AllowedRadiusMeters,
		ClockInTime:         input.ClockInTime,
		IsActive:            true,
	}

	if err := database.DB.Create(&office).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat kantor"})
		return
	}

	// Automatically assign the office to the manager who created it
	assignment := models.ManagerOffice{
		ManagerID: userID,
		OfficeID:  office.ID,
	}

	if err := database.DB.Create(&assignment).Error; err != nil {
		// If assignment fails, delete the office to maintain consistency
		database.DB.Delete(&office)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal assign kantor ke manager"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Kantor berhasil dibuat dan di-assign", "data": office})
}

// UpdateOffice - Manager can update if assigned, super admin can update any
func UpdateOffice(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	officeID, _ := strconv.Atoi(c.Param("id"))

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	// Check permission
	if !user.IsSuperAdmin {
		var count int64
		database.DB.Model(&models.ManagerOffice{}).
			Where("manager_id = ? AND office_id = ?", userID, officeID).
			Count(&count)

		if count == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki akses ke kantor ini"})
			return
		}
	}

	var office models.OfficeLocation
	if err := database.DB.First(&office, officeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Kantor tidak ditemukan"})
		return
	}

	var input struct {
		Name                *string  `json:"name"`
		Address             *string  `json:"address"`
		Latitude            *float64 `json:"latitude"`
		Longitude           *float64 `json:"longitude"`
		AllowedRadiusMeters *float64 `json:"allowed_radius_meters"`
		ClockInTime         *string  `json:"clock_in_time"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	if input.Name != nil {
		office.Name = *input.Name
	}
	if input.Address != nil {
		office.Address = *input.Address
	}
	if input.Latitude != nil {
		office.Latitude = *input.Latitude
	}
	if input.Longitude != nil {
		office.Longitude = *input.Longitude
	}
	if input.AllowedRadiusMeters != nil {
		office.AllowedRadiusMeters = *input.AllowedRadiusMeters
	}
	if input.ClockInTime != nil {
		office.ClockInTime = *input.ClockInTime
	}

	if err := database.DB.Save(&office).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate kantor"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Kantor berhasil diupdate", "data": office})
}

// AssignOfficeToManager - Super admin assigns office to manager (max 4)
func AssignOfficeToManager(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	if !user.IsSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Hanya super admin yang dapat assign kantor"})
		return
	}

	var input struct {
		ManagerID uint `json:"manager_id" binding:"required"`
		OfficeID  uint `json:"office_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check manager exists and is a manager
	var manager models.User
	if err := database.DB.First(&manager, input.ManagerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Manager tidak ditemukan"})
		return
	}

	if manager.Role != "manager" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User bukan manager"})
		return
	}

	// Check office exists
	var office models.OfficeLocation
	if err := database.DB.First(&office, input.OfficeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Kantor tidak ditemukan"})
		return
	}

	// Check if already assigned
	var existingAssignment models.ManagerOffice
	if err := database.DB.Where("manager_id = ? AND office_id = ?", input.ManagerID, input.OfficeID).
		First(&existingAssignment).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Manager sudah di-assign ke kantor ini"})
		return
	}

	// Check manager doesn't have more than 4 offices
	var count int64
	database.DB.Model(&models.ManagerOffice{}).Where("manager_id = ?", input.ManagerID).Count(&count)

	if count >= 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Manager sudah memiliki 4 kantor (maksimal)"})
		return
	}

	// Create assignment
	assignment := models.ManagerOffice{
		ManagerID: input.ManagerID,
		OfficeID:  input.OfficeID,
	}

	if err := database.DB.Create(&assignment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal assign kantor"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Kantor berhasil di-assign ke manager"})
}

// UnassignOfficeFromManager - Super admin removes office from manager (must keep at least 1)
func UnassignOfficeFromManager(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	if !user.IsSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Hanya super admin yang dapat unassign kantor"})
		return
	}

	var input struct {
		ManagerID uint `json:"manager_id" binding:"required"`
		OfficeID  uint `json:"office_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check manager has more than 1 office
	var count int64
	database.DB.Model(&models.ManagerOffice{}).Where("manager_id = ?", input.ManagerID).Count(&count)

	if count <= 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Manager harus memiliki minimal 1 kantor"})
		return
	}

	// Delete assignment
	result := database.DB.Where("manager_id = ? AND office_id = ?", input.ManagerID, input.OfficeID).
		Delete(&models.ManagerOffice{})

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Assignment tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Kantor berhasil di-unassign dari manager"})
}

// GetManagerOffices - Get offices assigned to a manager
func GetManagerOffices(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var offices []models.OfficeLocation
	database.DB.
		Joins("JOIN manager_offices ON manager_offices.office_id = office_locations.id").
		Where("manager_offices.manager_id = ? AND office_locations.is_active = ?", userID, true).
		Find(&offices)

	c.JSON(http.StatusOK, gin.H{"data": offices, "count": len(offices)})
}

// DeleteOffice - Soft delete office (sets is_active = false)
func DeleteOffice(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	officeID, _ := strconv.Atoi(c.Param("id"))

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	// Check permission - must be assigned or super admin
	if !user.IsSuperAdmin {
		var count int64
		database.DB.Model(&models.ManagerOffice{}).
			Where("manager_id = ? AND office_id = ?", userID, officeID).
			Count(&count)

		if count == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki akses ke kantor ini"})
			return
		}
	}

	var office models.OfficeLocation
	if err := database.DB.First(&office, officeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Kantor tidak ditemukan"})
		return
	}

	// Check if office has active attendance records
	var attendanceCount int64
	database.DB.Model(&models.Attendance{}).
		Where("approved_office_id = ?", officeID).
		Count(&attendanceCount)

	if attendanceCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak dapat menghapus kantor yang memiliki riwayat absensi"})
		return
	}

	// Soft delete: set is_active to false
	office.IsActive = false
	if err := database.DB.Save(&office).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus kantor"})
		return
	}

	// Remove all manager assignments
	database.DB.Where("office_id = ?", officeID).Delete(&models.ManagerOffice{})

	c.JSON(http.StatusOK, gin.H{"message": "Kantor berhasil dihapus"})
}

// GetEmployeeOffices - Shows offices managed by employee's manager
func GetEmployeeOffices(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
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
			c.JSON(http.StatusNotFound, gin.H{"error": "Manager tidak ditemukan"})
			return
		}
		managerID = manager.ID
	}

	// Get all offices managed by the manager
	var offices []models.OfficeLocation
	database.DB.
		Joins("JOIN manager_offices ON manager_offices.office_id = office_locations.id").
		Where("manager_offices.manager_id = ? AND office_locations.is_active = ?", managerID, true).
		Find(&offices)

	c.JSON(http.StatusOK, gin.H{"data": offices, "count": len(offices)})
}
