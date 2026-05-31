package handlers

import (
	"net/http"

	"field-attendance-system/auth"
	"field-attendance-system/database"
	"field-attendance-system/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type RegisterInput struct {
	Username string `json:"username" binding:"required"`
	FullName string `json:"full_name"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"omitempty,oneof=employee manager instructor"` // optional, defaults to employee if empty.
}

type LoginInput struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	RememberMe bool   `json:"remember_me"` // If true, session lasts 7 days
}

func Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Default role if not provided or restricted logic could go here.
	// For simplicity, we trust the input role or default to employee in DB structure,
	// but here we set it explicitly if needed.
	role := input.Role
	if role == "" {
		role = "employee"
	}

	user := models.User{
		Username:     input.Username,
		FullName:     input.FullName,
		PasswordHash: string(hashedPassword),
		Role:         role,
	}

	if result := database.DB.Create(&user); result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username already exists or invalid data"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
}

func Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if result := database.DB.Where("username = ?", input.Username).First(&user); result.Error != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Determine session duration
	var durationHours int
	if input.RememberMe {
		durationHours = 168 // 7 days (remember me)
	} else {
		durationHours = GetSessionDurationHours() // Use system setting
	}

	token, err := auth.GenerateToken(user.ID, user.Role, durationHours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token, "role": user.Role, "full_name": user.FullName, "username": user.Username})
}

func CreateUser(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Force role to employee for now, or respect input if needed.
	// The requirement is "Add Employee", so we default to employee.
	user := models.User{
		Username:     input.Username,
		FullName:     input.FullName,
		PasswordHash: string(hashedPassword),
		Role:         "employee",
	}

	if result := database.DB.Create(&user); result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username already exists or invalid data"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Employee created successfully", "user": user})
}
