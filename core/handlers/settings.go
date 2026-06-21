package handlers

import (
	"net/http"
	"strconv"

	"handayani-core/database"
	"handayani-core/models"

	"github.com/gin-gonic/gin"
)

// GetSessionDuration returns the current session duration setting
func GetSessionDuration(c *gin.Context) {
	var setting models.SystemSettings
	result := database.DB.Where("setting_key = ?", models.SettingSessionDurationHours).First(&setting)

	if result.Error != nil {
		// Return default value if not found
		c.JSON(http.StatusOK, gin.H{
			"setting_key":   models.SettingSessionDurationHours,
			"setting_value": "24",
			"description":   "Durasi sesi login default (jam)",
		})
		return
	}

	c.JSON(http.StatusOK, setting)
}

// UpdateSessionDuration updates the session duration setting (manager only)
func UpdateSessionDuration(c *gin.Context) {
	var input struct {
		DurationHours int `json:"duration_hours" binding:"required,min=1,max=168"` // 1 hour to 7 days
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Durasi harus antara 1-168 jam (1 jam - 7 hari)",
			"detail": err.Error(),
		})
		return
	}

	// Check if user is super admin
	userID := c.MustGet("userID").(uint)
	var user models.User
	database.DB.First(&user, userID)

	if !user.IsSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Hanya super admin yang dapat mengubah pengaturan ini"})
		return
	}

	var setting models.SystemSettings
	result := database.DB.Where("setting_key = ?", models.SettingSessionDurationHours).First(&setting)

	if result.Error != nil {
		// Create new setting
		setting = models.SystemSettings{
			SettingKey:   models.SettingSessionDurationHours,
			SettingValue: strconv.Itoa(input.DurationHours),
			Description:  "Durasi sesi login default (jam)",
		}
		database.DB.Create(&setting)
	} else {
		// Update existing setting
		setting.SettingValue = strconv.Itoa(input.DurationHours)
		database.DB.Save(&setting)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Durasi sesi berhasil diperbarui",
		"duration_hours": input.DurationHours,
	})
}

// GetSystemSettings returns all system settings (for manager dashboard)
func GetSystemSettings(c *gin.Context) {
	var settings []models.SystemSettings
	database.DB.Find(&settings)

	// Create a map for easier frontend consumption
	settingsMap := make(map[string]string)
	for _, s := range settings {
		settingsMap[s.SettingKey] = s.SettingValue
	}

	// Add defaults if not set
	if _, ok := settingsMap[models.SettingSessionDurationHours]; !ok {
		settingsMap[models.SettingSessionDurationHours] = "24"
	}
	if _, ok := settingsMap[models.SettingQuotaPresetOptions]; !ok {
		settingsMap[models.SettingQuotaPresetOptions] = "8,10"
	}

	c.JSON(http.StatusOK, gin.H{"settings": settingsMap})
}

// GetQuotaPresetsSetting returns the quota presets for the admin dashboard
func GetQuotaPresetsSetting(c *gin.Context) {
	var setting models.SystemSettings
	if err := database.DB.Where("setting_key = ?", models.SettingQuotaPresetOptions).First(&setting).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"setting_key":   models.SettingQuotaPresetOptions,
			"setting_value": "8,10",
			"description":   "Opsi preset kuota murid (jam, dipisahkan koma)",
		})
		return
	}
	c.JSON(http.StatusOK, setting)
}

// UpdateQuotaPresets updates the quota preset options (manager/super admin only)
func UpdateQuotaPresets(c *gin.Context) {
	var input struct {
		Presets string `json:"presets" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format preset tidak valid"})
		return
	}

	userID := c.MustGet("userID").(uint)
	var user models.User
	database.DB.First(&user, userID)

	if !user.IsSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Hanya super admin yang dapat mengubah pengaturan ini"})
		return
	}

	var setting models.SystemSettings
	result := database.DB.Where("setting_key = ?", models.SettingQuotaPresetOptions).First(&setting)

	if result.Error != nil {
		setting = models.SystemSettings{
			SettingKey:   models.SettingQuotaPresetOptions,
			SettingValue: input.Presets,
			Description:  "Opsi preset kuota murid (jam, dipisahkan koma)",
		}
		database.DB.Create(&setting)
	} else {
		setting.SettingValue = input.Presets
		database.DB.Save(&setting)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Preset kuota berhasil diperbarui",
		"presets": input.Presets,
	})
}

// Helper function to get session duration from database
func GetSessionDurationHours() int {
	var setting models.SystemSettings
	result := database.DB.Where("setting_key = ?", models.SettingSessionDurationHours).First(&setting)

	if result.Error != nil {
		return 24 // Default 24 hours
	}

	hours, err := strconv.Atoi(setting.SettingValue)
	if err != nil || hours <= 0 {
		return 24
	}

	return hours
}

// GetMinimumWorkHours returns the current manager's minimum work hours setting
func GetMinimumWorkHours(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User tidak ditemukan"})
		return
	}

	minHours := user.MinimumWorkHours
	if minHours <= 0 {
		minHours = 8 // default
	}

	c.JSON(http.StatusOK, gin.H{
		"minimum_work_hours": minHours,
	})
}

// UpdateMinimumWorkHours updates the manager's minimum work hours setting
func UpdateMinimumWorkHours(c *gin.Context) {
	var input struct {
		MinimumWorkHours float64 `json:"minimum_work_hours" binding:"required,min=1,max=24"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Jam kerja minimum harus antara 1-24 jam",
			"detail": err.Error(),
		})
		return
	}

	userID := c.MustGet("userID").(uint)
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User tidak ditemukan"})
		return
	}

	user.MinimumWorkHours = input.MinimumWorkHours
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan pengaturan jam kerja minimum"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":            "Jam kerja minimum berhasil diperbarui",
		"minimum_work_hours": user.MinimumWorkHours,
	})
}
