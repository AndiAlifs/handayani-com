package handlers

// EmployeeProfile + EmployeeCompensation CRUD (manager-only, gated at the
// route). Bare JSON bodies, camelCase — matches the CRM handlers.

import (
	"net/http"

	"handayani-core/database"
	"handayani-core/models"

	"github.com/gin-gonic/gin"
)

func ListEmployeeProfiles(c *gin.Context) {
	profiles := make([]models.EmployeeProfile, 0)
	if err := database.DB.Order("id").Find(&profiles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list employee profiles"})
		return
	}
	c.JSON(http.StatusOK, profiles)
}

func CreateEmployeeProfile(c *gin.Context) {
	var p models.EmployeeProfile
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if p.Pph21Category == "" {
		p.Pph21Category = "pegawai_tetap"
	}
	if !isOneOf(p.Pph21Category, validPph21Category...) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pph21Category"})
		return
	}
	if p.EmploymentType != "" && !isOneOf(p.EmploymentType, validEmploymentTyp...) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid employmentType"})
		return
	}
	p.ID = 0
	if err := database.DB.Create(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create employee profile"})
		return
	}
	c.JSON(http.StatusCreated, p)
}

func UpdateEmployeeProfile(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	var p models.EmployeeProfile
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if p.Pph21Category != "" && !isOneOf(p.Pph21Category, validPph21Category...) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pph21Category"})
		return
	}
	if p.EmploymentType != "" && !isOneOf(p.EmploymentType, validEmploymentTyp...) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid employmentType"})
		return
	}
	p.ID = id
	// Select the editable columns explicitly so a false IsActive (deactivation)
	// and cleared text fields are persisted — GORM's struct-based Updates skips
	// zero-valued fields. UserID is intentionally omitted (it is the 1:1 identity
	// link and must not be repointed via a profile update).
	if err := database.DB.Model(&models.EmployeeProfile{}).Where("id = ?", id).
		Select("NIK", "NPWP", "PtkpStatus", "EmploymentType", "Pph21Category",
			"BankName", "BankAccountNo", "BankAccountName", "BpjsKesehatanNo",
			"BpjsTkNo", "Email", "WhatsApp", "JoinDate", "IsActive").
		Updates(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update employee profile"})
		return
	}
	c.JSON(http.StatusOK, p)
}

func DeleteEmployeeProfile(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	if err := database.DB.Delete(&models.EmployeeProfile{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete employee profile"})
		return
	}
	c.Status(http.StatusNoContent)
}

func ListEmployeeCompensations(c *gin.Context) {
	userID, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	comps := make([]models.EmployeeCompensation, 0)
	if err := database.DB.Where("user_id = ?", userID).Order("effective_from DESC, id DESC").
		Find(&comps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list compensation"})
		return
	}
	c.JSON(http.StatusOK, comps)
}

func UpsertEmployeeCompensation(c *gin.Context) {
	userID, err := parseUintParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	var comp models.EmployeeCompensation
	if err := c.ShouldBindJSON(&comp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !isOneOf(comp.PayBasis, validPayBasis...) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payBasis"})
		return
	}
	comp.ID = 0
	comp.UserID = userID
	if err := database.DB.Create(&comp).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create compensation"})
		return
	}
	c.JSON(http.StatusCreated, comp)
}
