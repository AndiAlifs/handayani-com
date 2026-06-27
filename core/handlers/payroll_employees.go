package handlers

// EmployeeProfile + EmployeeCompensation CRUD (manager-only, gated at the
// route). Bare JSON bodies, camelCase — matches the CRM handlers.

import (
	"net/http"
	"time"

	"handayani-core/database"
	"handayani-core/models"

	"github.com/gin-gonic/gin"
)

// updateEmployeeProfileInput uses pointer fields so a partial PUT only touches
// the keys the client actually sent: a nil pointer leaves the column untouched,
// while a present pointer (even to "" or false) is written. This is the proper
// fix for the old Select(...)-based Updates, which force-wrote every listed
// column and so blanked omitted fields / silently deactivated the employee.
type updateEmployeeProfileInput struct {
	NIK             *string    `json:"nik"`
	NPWP            *string    `json:"npwp"`
	PtkpStatus      *string    `json:"ptkpStatus"`
	EmploymentType  *string    `json:"employmentType"`
	Pph21Category   *string    `json:"pph21Category"`
	BankName        *string    `json:"bankName"`
	BankAccountNo   *string    `json:"bankAccountNo"`
	BankAccountName *string    `json:"bankAccountName"`
	BpjsKesehatanNo *string    `json:"bpjsKesehatanNo"`
	BpjsTkNo        *string    `json:"bpjsTkNo"`
	Email           *string    `json:"email"`
	WhatsApp        *string    `json:"whatsapp"`
	JoinDate        *time.Time `json:"joinDate"`
	IsActive        *bool      `json:"isActive"`
}

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
	var in updateEmployeeProfileInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Validate enums only when actually provided and non-empty.
	if in.Pph21Category != nil && *in.Pph21Category != "" && !isOneOf(*in.Pph21Category, validPph21Category...) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pph21Category"})
		return
	}
	if in.EmploymentType != nil && *in.EmploymentType != "" && !isOneOf(*in.EmploymentType, validEmploymentTyp...) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid employmentType"})
		return
	}

	// Load the existing row so omitted fields are preserved; overlay only the
	// provided keys, then Save. UserID/CreatedAt are never in the DTO, so they
	// cannot be repointed by an update.
	var p models.EmployeeProfile
	if err := database.DB.Where("id = ?", id).First(&p).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee profile not found"})
		return
	}
	if in.NIK != nil {
		p.NIK = *in.NIK
	}
	if in.NPWP != nil {
		p.NPWP = *in.NPWP
	}
	if in.PtkpStatus != nil {
		p.PtkpStatus = *in.PtkpStatus
	}
	if in.EmploymentType != nil {
		p.EmploymentType = *in.EmploymentType
	}
	if in.Pph21Category != nil {
		p.Pph21Category = *in.Pph21Category
	}
	if in.BankName != nil {
		p.BankName = *in.BankName
	}
	if in.BankAccountNo != nil {
		p.BankAccountNo = *in.BankAccountNo
	}
	if in.BankAccountName != nil {
		p.BankAccountName = *in.BankAccountName
	}
	if in.BpjsKesehatanNo != nil {
		p.BpjsKesehatanNo = *in.BpjsKesehatanNo
	}
	if in.BpjsTkNo != nil {
		p.BpjsTkNo = *in.BpjsTkNo
	}
	if in.Email != nil {
		p.Email = *in.Email
	}
	if in.WhatsApp != nil {
		p.WhatsApp = *in.WhatsApp
	}
	if in.JoinDate != nil {
		p.JoinDate = in.JoinDate
	}
	if in.IsActive != nil {
		p.IsActive = *in.IsActive
	}

	if err := database.DB.Save(&p).Error; err != nil {
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
