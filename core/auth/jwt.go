package auth

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var warnSecretOnce sync.Once

// secret resolves the JWT signing key lazily on every token operation. Reading
// it here (rather than at package-init) ensures main()'s godotenv.Load() has
// already populated the environment — a package-level var would be evaluated
// before main() runs and miss a JWT_SECRET supplied only via .env.
//
// There is intentionally NO hardcoded fallback: a compiled-in default key is
// public knowledge and lets anyone forge a manager token. main() fails fast
// when JWT_SECRET is unset; this only warns to cover non-server callers (tests),
// where signing and verification still use the same (empty) key consistently.
func secret() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		warnSecretOnce.Do(func() {
			log.Println("WARNING: JWT_SECRET is not set; tokens are signed with an empty key.")
		})
	}
	return []byte(s)
}

type Claims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken creates a JWT token for a user with specified duration in hours
func GenerateToken(userID uint, role string, durationHours int) (string, error) {
	if durationHours <= 0 {
		durationHours = 24 // Default to 24 hours if not specified
	}
	expirationTime := time.Now().Add(time.Duration(durationHours) * time.Hour)
	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret())
}

// AuthMiddleware protects routes
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return secret(), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// ManagerMiddleware ensures the user has manager role
func ManagerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != "manager" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Manager access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// InstructorMiddleware ensures the user has instructor role.
func InstructorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != "instructor" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Instructor access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}
