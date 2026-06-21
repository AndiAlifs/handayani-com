package handlers

// Course catalog CRUD (PRD Epic 2). Migrated from the FastAPI service.
// Public (no auth), bare JSON bodies — matches the Angular ApiService contract,
// not this service's gin.H{"data":...} envelope convention.

import (
	"net/http"

	"field-attendance-system/database"
	"field-attendance-system/models"

	"github.com/gin-gonic/gin"
)

func ListCourses(c *gin.Context) {
	courses := make([]models.Course, 0)
	if err := database.DB.Order("category, id").Find(&courses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list courses"})
		return
	}
	c.JSON(http.StatusOK, courses)
}

func CreateCourse(c *gin.Context) {
	var course models.Course
	if err := c.ShouldBindJSON(&course); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	course.ID = 0
	if err := database.DB.Create(&course).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create course"})
		return
	}
	c.JSON(http.StatusCreated, course)
}

func UpdateCourse(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	var course models.Course
	if err := c.ShouldBindJSON(&course); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	course.ID = id
	if err := database.DB.Save(&course).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update course"})
		return
	}
	c.JSON(http.StatusOK, course)
}

func DeleteCourse(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	if err := database.DB.Delete(&models.Course{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete course"})
		return
	}
	c.Status(http.StatusNoContent)
}
