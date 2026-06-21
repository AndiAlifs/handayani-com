package handlers

// SIM mechanism steps CRUD (PRD Epic 4). Migrated from the FastAPI service.
// Public (no auth), bare JSON bodies.

import (
	"net/http"

	"field-attendance-system/database"
	"field-attendance-system/models"

	"github.com/gin-gonic/gin"
)

func ListMechanisms(c *gin.Context) {
	mechs := make([]models.Mechanism, 0)
	if err := database.DB.Order("sort_order, id").Find(&mechs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list mechanisms"})
		return
	}
	c.JSON(http.StatusOK, mechs)
}

func CreateMechanism(c *gin.Context) {
	var mech models.Mechanism
	if err := c.ShouldBindJSON(&mech); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Auto-assign the next sort_order, mirroring the Python INSERT subquery.
	var maxOrder int
	database.DB.Model(&models.Mechanism{}).Select("COALESCE(MAX(sort_order), 0)").Scan(&maxOrder)
	mech.ID = 0
	mech.SortOrder = maxOrder + 1
	if err := database.DB.Create(&mech).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create mechanism"})
		return
	}
	c.JSON(http.StatusCreated, mech)
}

func UpdateMechanism(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	var mech models.Mechanism
	if err := c.ShouldBindJSON(&mech); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mech.ID = id
	// Update only the four wire fields — never sort_order (a Save would zero it
	// and scramble list order). A map updates only the listed columns.
	if err := database.DB.Model(&models.Mechanism{}).Where("id = ?", id).Updates(map[string]interface{}{
		"requirement_name": mech.RequirementName,
		"issuing_body":     mech.IssuingBody,
		"cost":             mech.Cost,
		"notes":            mech.Notes,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update mechanism"})
		return
	}
	c.JSON(http.StatusOK, mech)
}

func DeleteMechanism(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	if err := database.DB.Delete(&models.Mechanism{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete mechanism"})
		return
	}
	c.Status(http.StatusNoContent)
}
