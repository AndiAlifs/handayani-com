package handlers

// PayComponent master + per-employee component assignment (manager-only).
// Bare JSON bodies, camelCase.

import (
	"net/http"

	"handayani-core/database"
	"handayani-core/models"

	"github.com/gin-gonic/gin"
)

func ListPayComponents(c *gin.Context) {
	items := make([]models.PayComponent, 0)
	if err := database.DB.Order("id").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list components"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func CreatePayComponent(c *gin.Context) {
	var pc models.PayComponent
	if err := c.ShouldBindJSON(&pc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if pc.Code == "" || pc.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code and name are required"})
		return
	}
	if !isOneOf(pc.ComponentType, validComponentType...) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid componentType"})
		return
	}
	if pc.DefaultCalc == "" {
		pc.DefaultCalc = "fixed"
	}
	if !isOneOf(pc.DefaultCalc, validDefaultCalc...) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid defaultCalc"})
		return
	}
	pc.ID = 0
	if err := database.DB.Create(&pc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create component (code must be unique)"})
		return
	}
	c.JSON(http.StatusCreated, pc)
}

func UpdatePayComponent(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	var pc models.PayComponent
	if err := c.ShouldBindJSON(&pc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if pc.ComponentType != "" && !isOneOf(pc.ComponentType, validComponentType...) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid componentType"})
		return
	}
	pc.ID = id
	if err := database.DB.Model(&models.PayComponent{}).Where("id = ?", id).Updates(&pc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update component"})
		return
	}
	c.JSON(http.StatusOK, pc)
}

func DeletePayComponent(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	if err := database.DB.Delete(&models.PayComponent{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete component"})
		return
	}
	c.Status(http.StatusNoContent)
}

func ListEmployeeComponents(c *gin.Context) {
	userID, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	items := make([]models.EmployeeComponent, 0)
	if err := database.DB.Where("user_id = ?", userID).Order("id").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list employee components"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func AssignEmployeeComponent(c *gin.Context) {
	userID, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	var ec models.EmployeeComponent
	if err := c.ShouldBindJSON(&ec); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if ec.ComponentID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "componentId is required"})
		return
	}
	ec.ID = 0
	ec.UserID = userID
	if err := database.DB.Create(&ec).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign component"})
		return
	}
	c.JSON(http.StatusCreated, ec)
}

func DeleteEmployeeComponent(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	if err := database.DB.Delete(&models.EmployeeComponent{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete employee component"})
		return
	}
	c.Status(http.StatusNoContent)
}
