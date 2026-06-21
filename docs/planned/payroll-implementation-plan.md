# Payroll Module — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Companion spec:** [payroll-module-design.md](payroll-module-design.md) (read it first).

**Goal:** Build the YPA Handayani internal payroll module, all-Go in the `core/` service, shipped as six independently-testable slices.

**Architecture:** New payroll tables owned by the Go gateway (added to `AutoMigrate`), a pure-function calculation engine, document generators, delivery adapters, and an Angular dashboard area. JSON is camelCase; writes are manager-gated; statutory rates live as data, not code.

**Tech Stack:** Go 1.24.3 · Gin · GORM (MySQL in prod) · `glebarez/sqlite` (pure-Go, tests only) · maroto v2 (PDF) · `excelize/v2` (Excel, already a dep) · gomail (email).

## Global Constraints

- Go module is `handayani-core`; the Go service directory is **`core/`** (NOT `attendance-backend/` — the spec's paths are stale; use `core/`).
- All HTTP JSON is **camelCase**, **bare-body** (no `{"data": …}` envelope), matching `handlers/crm.go`.
- The shared DB handle is the global `database.DB *gorm.DB`. Handlers read/write through it.
- Auth: `auth.AuthMiddleware()` sets `c.Get("userID")` (uint) and `c.Get("role")` (string). `auth.ManagerMiddleware()` enforces `role == "manager"`. Payroll **writes and run management are manager-only**; employees may read **only their own** payslips (Phase 3+).
- Payroll models use plain `varchar` string fields with **app-level validation** (NOT MySQL `type:enum(...)`), and plain indexed `uint` FK columns with **no DB-level foreign-key constraints**. This keeps the models portable to the pure-Go sqlite driver used in tests (sqlite rejects `enum('a','b')` column syntax).
- Money is **`int64` rupiah** (no floats for currency). Dates are `*time.Time` with `gorm:"type:date"`.
- Verify against `go build ./...` and `go vet ./...` from `core/`. Run tests with `go test ./...` from `core/`.

## Plan scope & phasing

This plan **fully details Phase 1** (payroll master-data model + CRUD) as bite-sized TDD tasks.
Phases 2–6 are summarized in the **Roadmap** at the end and will each get their own
fully-detailed plan written just-in-time (two of them are gated on open research items —
⚠#1 instructor tax category, ⚠#2 Coretax template — per the spec §11).

**Refinement vs spec §14:** the spec lumped "statutory-config seed" into Phase 1. This plan
moves the statutory config (PTKP/TER/brackets/BPJS tables + seed) into **Phase 2**, because
those rates are only *exercised* by the calculation engine and are best introduced next to the
golden tests that verify them. Phase 1 is therefore a clean, self-contained master-data slice.

---

## Phase 1 — Payroll master-data model + CRUD

Four new tables (`EmployeeProfile`, `EmployeeCompensation`, `PayComponent`, `EmployeeComponent`)
and their manager-gated CRUD endpoints. Deliverable: a manager can fully manage employee
payroll-administrative data, pay basis, the component master, and per-employee recurring
components. No tax math yet.

### File structure (Phase 1)

- Create: `core/models/payroll.go` — the four GORM structs.
- Create: `core/models/payroll_migrate_test.go` — migration smoke test (sqlite).
- Create: `core/handlers/payroll_validate.go` — tiny shared validators.
- Create: `core/handlers/payroll_employees.go` — EmployeeProfile + EmployeeCompensation handlers.
- Create: `core/handlers/payroll_components.go` — PayComponent + EmployeeComponent handlers.
- Create: `core/handlers/payroll_routes.go` — `RegisterPayrollRoutes(r *gin.Engine)`.
- Create: `core/handlers/payroll_setup_test.go` — shared test helpers (sqlite DB + test router + tokens).
- Create: `core/handlers/payroll_employees_test.go`, `core/handlers/payroll_components_test.go`.
- Modify: `core/main.go` — add models to `AutoMigrate`, call `handlers.RegisterPayrollRoutes(r)`.
- Modify: `core/go.mod` / `go.sum` — add `github.com/glebarez/sqlite` (test-only).

---

### Task 1: Payroll master-data models + migration test

**Files:**
- Create: `core/models/payroll.go`
- Create: `core/models/payroll_migrate_test.go`
- Modify: `core/go.mod` (add `github.com/glebarez/sqlite`)
- Modify: `core/main.go:27-37` (AutoMigrate list)

**Interfaces:**
- Produces: `models.EmployeeProfile`, `models.EmployeeCompensation`, `models.PayComponent`,
  `models.EmployeeComponent` — struct types consumed by every later task.

- [ ] **Step 1: Add the test-only sqlite driver**

Run from `core/`:
```bash
go get github.com/glebarez/sqlite@latest
```
Expected: `go.mod` gains `github.com/glebarez/sqlite`.

- [ ] **Step 2: Write the failing migration test**

Create `core/models/payroll_migrate_test.go`:
```go
package models

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestPayrollModelsMigrate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&EmployeeProfile{},
		&EmployeeCompensation{},
		&PayComponent{},
		&EmployeeComponent{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, tbl := range []string{"employee_profiles", "employee_compensations", "pay_components", "employee_components"} {
		if !db.Migrator().HasTable(tbl) {
			t.Errorf("expected table %q to exist", tbl)
		}
	}
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./models/ -run TestPayrollModelsMigrate -v`
Expected: FAIL — `undefined: EmployeeProfile` (and the other three).

- [ ] **Step 4: Create the models**

Create `core/models/payroll.go`:
```go
package models

import "time"

// EmployeeProfile — payroll-administrative layer, 1:1 with User (userId is a plain
// indexed column; no DB-level FK constraint, for sqlite-test portability).
type EmployeeProfile struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	UserID          uint       `gorm:"uniqueIndex;not null" json:"userId"`
	NIK             string     `gorm:"type:varchar(32)" json:"nik"`
	NPWP            string     `gorm:"type:varchar(32)" json:"npwp"` // empty => +20% PPh 21 (Phase 2)
	PtkpStatus      string     `gorm:"type:varchar(8)" json:"ptkpStatus"`     // TK/0 … K/3
	EmploymentType  string     `gorm:"type:varchar(16)" json:"employmentType"` // permanent|contract|freelance
	Pph21Category   string     `gorm:"type:varchar(24)" json:"pph21Category"`  // pegawai_tetap|bukan_pegawai|pegawai_tidak_tetap
	BankName        string     `gorm:"type:varchar(64)" json:"bankName"`
	BankAccountNo   string     `gorm:"type:varchar(40)" json:"bankAccountNo"`
	BankAccountName string     `gorm:"type:varchar(128)" json:"bankAccountName"`
	BpjsKesehatanNo string     `gorm:"type:varchar(32)" json:"bpjsKesehatanNo"`
	BpjsTkNo        string     `gorm:"type:varchar(32)" json:"bpjsTkNo"`
	Email           string     `gorm:"type:varchar(128)" json:"email"`
	WhatsApp        string     `gorm:"type:varchar(24)" json:"whatsapp"`
	JoinDate        *time.Time `gorm:"type:date" json:"joinDate"`
	IsActive        bool       `gorm:"default:true" json:"isActive"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

// EmployeeCompensation — pay basis for an employee (effective-dated).
type EmployeeCompensation struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	UserID        uint       `gorm:"index;not null" json:"userId"`
	PayBasis      string     `gorm:"type:varchar(16)" json:"payBasis"` // monthly_fixed|per_session|per_hour
	BaseSalary    int64      `json:"baseSalary"`                       // rupiah, for monthly_fixed
	Rate          int64      `json:"rate"`                             // rupiah per session/hour
	EffectiveFrom *time.Time `gorm:"type:date" json:"effectiveFrom"`
	EffectiveTo   *time.Time `gorm:"type:date" json:"effectiveTo"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// PayComponent — master list of earning/deduction types.
type PayComponent struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	Code          string `gorm:"type:varchar(32);uniqueIndex" json:"code"`
	Name          string `gorm:"type:varchar(128)" json:"name"`
	ComponentType string `gorm:"type:varchar(12)" json:"componentType"` // earning|deduction
	Taxable       bool   `json:"taxable"`
	IsBpjsBase    bool   `json:"isBpjsBase"`
	DefaultCalc   string `gorm:"type:varchar(12)" json:"defaultCalc"` // fixed|manual
}

// EmployeeComponent — a recurring component assigned to an employee.
type EmployeeComponent struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	UserID        uint       `gorm:"index;not null" json:"userId"`
	ComponentID   uint       `gorm:"index;not null" json:"componentId"`
	Amount        int64      `json:"amount"` // rupiah
	EffectiveFrom *time.Time `gorm:"type:date" json:"effectiveFrom"`
	EffectiveTo   *time.Time `gorm:"type:date" json:"effectiveTo"`
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test ./models/ -run TestPayrollModelsMigrate -v`
Expected: PASS.

- [ ] **Step 6: Register the models in AutoMigrate**

In `core/main.go`, extend the `database.DB.AutoMigrate(...)` call (currently ending with
`&models.LearningPlan{},`) to also include the four new models:
```go
		&models.Student{},
		&models.StudentSession{},
		&models.LearningPlan{},
		&models.EmployeeProfile{},
		&models.EmployeeCompensation{},
		&models.PayComponent{},
		&models.EmployeeComponent{},
	)
```

- [ ] **Step 7: Verify the build**

Run: `go build ./...` then `go vet ./...`
Expected: both succeed, no output.

- [ ] **Step 8: Commit**

```bash
git add core/models/payroll.go core/models/payroll_migrate_test.go core/main.go core/go.mod core/go.sum
git commit -m "feat(payroll): add master-data models and migration"
```

---

### Task 2: EmployeeProfile + EmployeeCompensation CRUD

**Files:**
- Create: `core/handlers/payroll_validate.go`
- Create: `core/handlers/payroll_routes.go`
- Create: `core/handlers/payroll_employees.go`
- Create: `core/handlers/payroll_setup_test.go`
- Create: `core/handlers/payroll_employees_test.go`
- Modify: `core/main.go` (call `handlers.RegisterPayrollRoutes(r)`)

**Interfaces:**
- Consumes: `models.EmployeeProfile`, `models.EmployeeCompensation` (Task 1); `database.DB`;
  `auth.AuthMiddleware`, `auth.ManagerMiddleware`, `auth.GenerateToken`; `parseUintParam` (exists in `handlers/knowledge.go`).
- Produces: `handlers.RegisterPayrollRoutes(r *gin.Engine)`; handler funcs
  `ListEmployeeProfiles`, `CreateEmployeeProfile`, `UpdateEmployeeProfile`,
  `DeleteEmployeeProfile`, `ListEmployeeCompensations`, `UpsertEmployeeCompensation`;
  test helpers `newPayrollTestDB(t)`, `newPayrollTestRouter()`, `bearer(t, role, userID)`.

- [ ] **Step 1: Write shared validators**

Create `core/handlers/payroll_validate.go`:
```go
package handlers

func isOneOf(v string, allowed ...string) bool {
	for _, a := range allowed {
		if v == a {
			return true
		}
	}
	return false
}

var (
	validPph21Category = []string{"pegawai_tetap", "bukan_pegawai", "pegawai_tidak_tetap"}
	validEmploymentTyp = []string{"permanent", "contract", "freelance"}
	validPayBasis      = []string{"monthly_fixed", "per_session", "per_hour"}
	validComponentType = []string{"earning", "deduction"}
	validDefaultCalc   = []string{"fixed", "manual"}
)
```

- [ ] **Step 2: Write the shared test harness**

Create `core/handlers/payroll_setup_test.go`:
```go
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
```

- [ ] **Step 3: Write the failing handler tests**

Create `core/handlers/payroll_employees_test.go`:
```go
package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"handayani-core/models"
)

func TestCreateEmployeeProfile_managerOK(t *testing.T) {
	newPayrollTestDB(t)
	r := newPayrollTestRouter()
	body := `{"userId":5,"ptkpStatus":"K/1","pph21Category":"pegawai_tetap","employmentType":"permanent"}`
	w := do(r, http.MethodPost, "/api/payroll/employees", bearer(t, "manager", 1), body)
	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d (%s)", w.Code, w.Body.String())
	}
	var got models.EmployeeProfile
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ID == 0 || got.UserID != 5 || got.PtkpStatus != "K/1" {
		t.Fatalf("unexpected profile: %+v", got)
	}
}

func TestCreateEmployeeProfile_rejectsBadCategory(t *testing.T) {
	newPayrollTestDB(t)
	r := newPayrollTestRouter()
	body := `{"userId":5,"pph21Category":"nonsense"}`
	w := do(r, http.MethodPost, "/api/payroll/employees", bearer(t, "manager", 1), body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestEmployeeProfile_requiresManager(t *testing.T) {
	newPayrollTestDB(t)
	r := newPayrollTestRouter()
	w := do(r, http.MethodGet, "/api/payroll/employees", bearer(t, "employee", 9), "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403 for employee, got %d", w.Code)
	}
}
```

- [ ] **Step 4: Run the tests to verify they fail**

Run: `go test ./handlers/ -run TestCreateEmployeeProfile -v`
Expected: FAIL — `RegisterPayrollRoutes` / handlers undefined.

- [ ] **Step 5: Write the routes file**

Create `core/handlers/payroll_routes.go`:
```go
package handlers

import (
	"handayani-core/auth"

	"github.com/gin-gonic/gin"
)

// RegisterPayrollRoutes wires the manager-gated payroll endpoints onto the root
// engine (mirrors the CRM group in main.go). Called from main().
func RegisterPayrollRoutes(r *gin.Engine) {
	g := r.Group("/api/payroll")
	g.Use(auth.AuthMiddleware(), auth.ManagerMiddleware())
	{
		g.GET("/employees", ListEmployeeProfiles)
		g.POST("/employees", CreateEmployeeProfile)
		g.PUT("/employees/:id", UpdateEmployeeProfile)
		g.DELETE("/employees/:id", DeleteEmployeeProfile)

		g.GET("/employees/:id/compensation", ListEmployeeCompensations)
		g.POST("/employees/:id/compensation", UpsertEmployeeCompensation)
	}
}
```

- [ ] **Step 6: Write the handlers**

Create `core/handlers/payroll_employees.go`:
```go
package handlers

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
	p.ID = id
	if err := database.DB.Model(&models.EmployeeProfile{}).Where("id = ?", id).Updates(&p).Error; err != nil {
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
```

- [ ] **Step 7: Wire the routes into main.go**

In `core/main.go`, after the CRM group registration (around the `sessions` group), add:
```go
	// Payroll — manager-gated master data + runs (PRD: internal payroll module).
	handlers.RegisterPayrollRoutes(r)
```

- [ ] **Step 8: Run the tests to verify they pass**

Run: `go test ./handlers/ -run 'TestCreateEmployeeProfile|TestEmployeeProfile' -v`
Expected: PASS (3 tests).

- [ ] **Step 9: Verify build + vet**

Run: `go build ./...` then `go vet ./...`
Expected: both succeed.

- [ ] **Step 10: Commit**

```bash
git add core/handlers/payroll_validate.go core/handlers/payroll_routes.go core/handlers/payroll_employees.go core/handlers/payroll_setup_test.go core/handlers/payroll_employees_test.go core/main.go
git commit -m "feat(payroll): employee profile + compensation CRUD"
```

---

### Task 3: PayComponent + EmployeeComponent CRUD

**Files:**
- Create: `core/handlers/payroll_components.go`
- Create: `core/handlers/payroll_components_test.go`
- Modify: `core/handlers/payroll_routes.go` (add component routes)

**Interfaces:**
- Consumes: `models.PayComponent`, `models.EmployeeComponent`; helpers from Task 2.
- Produces: handler funcs `ListPayComponents`, `CreatePayComponent`, `UpdatePayComponent`,
  `DeletePayComponent`, `ListEmployeeComponents`, `AssignEmployeeComponent`,
  `DeleteEmployeeComponent`.

- [ ] **Step 1: Write the failing tests**

Create `core/handlers/payroll_components_test.go`:
```go
package handlers

import (
	"net/http"
	"testing"
)

func TestCreatePayComponent_managerOK(t *testing.T) {
	newPayrollTestDB(t)
	r := newPayrollTestRouter()
	body := `{"code":"TUNJ_JABATAN","name":"Tunjangan Jabatan","componentType":"earning","taxable":true,"isBpjsBase":false,"defaultCalc":"fixed"}`
	w := do(r, http.MethodPost, "/api/payroll/components", bearer(t, "manager", 1), body)
	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestCreatePayComponent_rejectsBadType(t *testing.T) {
	newPayrollTestDB(t)
	r := newPayrollTestRouter()
	body := `{"code":"X","name":"X","componentType":"bogus","defaultCalc":"fixed"}`
	w := do(r, http.MethodPost, "/api/payroll/components", bearer(t, "manager", 1), body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestAssignEmployeeComponent_managerOK(t *testing.T) {
	newPayrollTestDB(t)
	r := newPayrollTestRouter()
	body := `{"componentId":3,"amount":500000}`
	w := do(r, http.MethodPost, "/api/payroll/employees/7/components", bearer(t, "manager", 1), body)
	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d (%s)", w.Code, w.Body.String())
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./handlers/ -run 'PayComponent|EmployeeComponent' -v`
Expected: FAIL — handlers/routes undefined.

- [ ] **Step 3: Write the handlers**

Create `core/handlers/payroll_components.go`:
```go
package handlers

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
```

- [ ] **Step 4: Add the routes**

In `core/handlers/payroll_routes.go`, inside the `{ … }` block, append:
```go
		g.GET("/components", ListPayComponents)
		g.POST("/components", CreatePayComponent)
		g.PUT("/components/:id", UpdatePayComponent)
		g.DELETE("/components/:id", DeletePayComponent)

		g.GET("/employees/:id/components", ListEmployeeComponents)
		g.POST("/employees/:id/components", AssignEmployeeComponent)
		g.DELETE("/employee-components/:id", DeleteEmployeeComponent)
```

> Routing note: `:id` is reused as the param name on the `/employees/:id/...` paths, matching
> the compensation routes from Task 2 — Gin requires the same wildcard name on a shared prefix.

- [ ] **Step 5: Run to verify passing**

Run: `go test ./handlers/ -run 'PayComponent|EmployeeComponent' -v`
Expected: PASS (3 tests).

- [ ] **Step 6: Full package test + build**

Run: `go test ./... && go build ./... && go vet ./...` (from `core/`)
Expected: all pass.

- [ ] **Step 7: Commit**

```bash
git add core/handlers/payroll_components.go core/handlers/payroll_components_test.go core/handlers/payroll_routes.go
git commit -m "feat(payroll): pay-component master + employee component assignment"
```

---

## Roadmap — Phases 2–6 (each gets its own detailed plan, written just-in-time)

These are scoped here for sequencing; each will be expanded into a full TDD task list when
reached. Two are gated on open research items.

**Phase 2 — Statutory config + calculation engine** (next; partly gated on ⚠#1)
- New `core/models/statutory.go`: `PtkpAmount`, `TerRate`, `ProgressiveBracket`, `BpjsConfig`,
  `PayrollConstant` (effective-dated, rates-as-data). Seed in `core/seed/`.
- New pure-function package `core/payroll/`: `CalcVariablePay`, `CalcEarnings`, `CalcBPJS`,
  `CalcPPh21` (dispatch on `pph21Category`), `BuildPayslip`. No DB/HTTP inside.
- Golden `go test` cases: TER Jan–Nov, December annual recalc, no-NPWP surcharge, BPJS
  with/without caps. **Resolve ⚠#1** (instructor category) and **verify all seeded rates**
  against current regulation before trusting output.

**Phase 3 — Run lifecycle + payslip persistence**
- New models `PayrollRun`, `Payslip`, `PayslipLine` (+ AutoMigrate). State machine
  `draft→calculated→finalized→paid`. Endpoints: create run, `calculate` (simulation),
  `finalize` (blocked on any `calcStatus=error`), `mark-paid`; `GET /payroll/me/payslips`
  with the JWT ownership check.

**Phase 4 — Documents** (gated on ⚠#2)
- Add maroto v2. Payslip PDF + bukti-potong PDF. Coretax import file via the already-present
  `excelize/v2`. **Resolve ⚠#2** (confirm the current e-Bupot 21 import template) first.

**Phase 5 — Delivery**
- `PayslipDeliverer` interface + `Dashboard` (no-op), `Email` (gomail/SMTP), `WhatsApp`
  (net/http to a chosen gateway) adapters; `DeliveryLog` + redeliver endpoint. Pick the
  WhatsApp gateway + its env contract.

**Phase 6 — Frontend**
- Angular `/dashboard/penggajian`: manager comp setup, component master, run review, finalize,
  Coretax/bukti-potong download; employee "Slip Gaji Saya". New `ApiService` methods +
  camelCase models + `id`/`en` i18n, preserving the mock-data fallback pattern.
