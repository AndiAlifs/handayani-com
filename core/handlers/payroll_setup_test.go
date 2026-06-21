package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"handayani-core/auth"
	"handayani-core/database"
	"handayani-core/models"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// newPayrollTestDB spins up an in-memory sqlite DB with only the payroll tables
// (User et al. use MySQL enum types sqlite can't parse) and wires it to the
// global database.DB the handlers use.
func newPayrollTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.EmployeeProfile{},
		&models.EmployeeCompensation{},
		&models.PayComponent{},
		&models.EmployeeComponent{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database.DB = db
	return db
}

func newPayrollTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterPayrollRoutes(r)
	return r
}

// bearer returns an Authorization header value for a freshly minted JWT.
func bearer(t *testing.T, role string, userID uint) string {
	t.Helper()
	tok, err := auth.GenerateToken(userID, role, 1)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	return "Bearer " + tok
}

// do issues a request against the test router. authHeader is named to avoid
// shadowing the imported auth package.
func do(r *gin.Engine, method, path, authHeader, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
