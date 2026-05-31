package handlers

import (
	"net/http"
	"time"

	"field-attendance-system/database"
	"field-attendance-system/models"

	"github.com/gin-gonic/gin"
)

type LeaveInput struct {
	StartDate string `json:"start_date" binding:"required"` // Format YYYY-MM-DD
	EndDate   string `json:"end_date" binding:"required"`   // Format YYYY-MM-DD
	Reason    string `json:"reason" binding:"required"`
}

func CreateLeaveRequest(c *gin.Context) {
	var input LeaveInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.MustGet("userID").(uint)

	// Parse dates
	layout := "2006-01-02"
	start, err := time.Parse(layout, input.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format, use YYYY-MM-DD"})
		return
	}
	end, err1 := time.Parse(layout, input.EndDate)
	if err1 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format, use YYYY-MM-DD"})
		return
	}

	leave := models.LeaveRequest{
		UserID:    userID,
		StartDate: start,
		EndDate:   end,
		Reason:    input.Reason,
		Status:    "pending",
	}

	if result := database.DB.Create(&leave); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create leave request"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Leave request submitted", "data": leave})
}

// GetTodayLeave returns today's leave status for the logged-in user
func GetTodayLeave(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	// Get today's date
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var leave models.LeaveRequest
	// Find leave request where today falls between start_date and end_date
	result := database.DB.Where("user_id = ? AND ? BETWEEN start_date AND end_date", userID, today).First(&leave)

	if result.Error != nil {
		// No leave request for today
		c.JSON(http.StatusOK, gin.H{
			"data":    nil,
			"message": "Tidak sedang cuti hari ini",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": leave,
	})
}

// GetMyLeaveHistory returns all leave requests for the logged-in user
func GetMyLeaveHistory(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var leaves []models.LeaveRequest
	result := database.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&leaves)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leave history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": leaves,
	})
}
