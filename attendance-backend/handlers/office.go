package handlers

import (
	"net/http"

	"field-attendance-system/database"
	"field-attendance-system/models"

	"github.com/gin-gonic/gin"
)

type OfficeLocationInput struct {
	Latitude            float64 `json:"latitude" binding:"required"`
	Longitude           float64 `json:"longitude" binding:"required"`
	AllowedRadiusMeters float64 `json:"allowed_radius_meters" binding:"required,min=0"`
	Name                string  `json:"name"`
	ClockInTime         string  `json:"clock_in_time"`
}

func GetOfficeLocation(c *gin.Context) {
	var office models.OfficeLocation
	result := database.DB.First(&office)

	if result.Error != nil {
		// No office location set yet
		c.JSON(http.StatusOK, gin.H{"data": nil, "message": "No office location set"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": office})
}

func SetOfficeLocation(c *gin.Context) {
	var input OfficeLocationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if office location already exists
	var office models.OfficeLocation
	result := database.DB.First(&office)

	if result.Error != nil {
		// Create new office location
		office = models.OfficeLocation{
			Latitude:            input.Latitude,
			Longitude:           input.Longitude,
			AllowedRadiusMeters: input.AllowedRadiusMeters,
			Name:                input.Name,
			ClockInTime:         input.ClockInTime,
		}
		if err := database.DB.Create(&office).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create office location"})
			return
		}
	} else {
		// Update existing office location
		office.Latitude = input.Latitude
		office.Longitude = input.Longitude
		office.AllowedRadiusMeters = input.AllowedRadiusMeters
		office.Name = input.Name
		office.ClockInTime = input.ClockInTime
		if err := database.DB.Save(&office).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update office location"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Office location saved successfully", "data": office})
}
