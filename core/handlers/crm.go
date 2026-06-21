package handlers

// CRM students CRUD (admin tooling). Manager-only (gated at the route).
// Migrated from the FastAPI service. Bare JSON bodies.

import (
	"net/http"

	"handayani-core/database"
	"handayani-core/models"

	"github.com/gin-gonic/gin"
)

func ListStudentsCrm(c *gin.Context) {
	students := make([]models.StudentCrm, 0)
	// Project created_at as a date string to match the Python wire format.
	if err := database.DB.Model(&models.StudentCrm{}).
		Select("id, name, phone, course_id, course_name, status, progress_score, notes, " +
			"DATE_FORMAT(created_at, '%Y-%m-%d') AS created_at").
		Order("created_at DESC, id DESC").
		Find(&students).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list students"})
		return
	}
	c.JSON(http.StatusOK, students)
}

func CreateStudentCrm(c *gin.Context) {
	var student models.StudentCrm
	if err := c.ShouldBindJSON(&student); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if student.Status == "" {
		student.Status = "lead" // mirror the Pydantic default
	}
	student.ID = 0
	student.CreatedAt = nil // let the DB default fill created_at
	// Select only the writable columns so GORM doesn't try to insert created_at.
	if err := database.DB.Select(
		"name", "phone", "course_id", "course_name", "status", "progress_score", "notes",
	).Create(&student).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create student"})
		return
	}
	c.JSON(http.StatusCreated, student)
}

func UpdateStudentCrm(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	var student models.StudentCrm
	if err := c.ShouldBindJSON(&student); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	student.ID = id
	// Update the 7 writable columns; never touch created_at.
	if err := database.DB.Model(&models.StudentCrm{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name":           student.Name,
		"phone":          student.Phone,
		"course_id":      student.CourseID,
		"course_name":    student.CourseName,
		"status":         student.Status,
		"progress_score": student.ProgressScore,
		"notes":          student.Notes,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update student"})
		return
	}
	c.JSON(http.StatusOK, student)
}

func DeleteStudentCrm(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	if err := database.DB.Delete(&models.StudentCrm{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete student"})
		return
	}
	c.Status(http.StatusNoContent)
}
