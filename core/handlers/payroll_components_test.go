package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"handayani-core/database"
	"handayani-core/models"
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

// Regression: GORM's struct-based Updates skips zero-valued fields, so flipping
// a boolean to false used to be silently ignored. UpdatePayComponent now selects
// the columns explicitly; false must persist.
func TestUpdatePayComponent_persistsFalseBooleans(t *testing.T) {
	newPayrollTestDB(t)
	r := newPayrollTestRouter()
	create := `{"code":"TUNJ","name":"Tunjangan","componentType":"earning","taxable":true,"isBpjsBase":true,"defaultCalc":"fixed"}`
	w := do(r, http.MethodPost, "/api/payroll/components", bearer(t, "manager", 1), create)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: want 201, got %d (%s)", w.Code, w.Body.String())
	}
	var created models.PayComponent
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create: %v", err)
	}

	update := `{"code":"TUNJ","name":"Tunjangan","componentType":"earning","taxable":false,"isBpjsBase":false,"defaultCalc":"fixed"}`
	w = do(r, http.MethodPut, fmt.Sprintf("/api/payroll/components/%d", created.ID), bearer(t, "manager", 1), update)
	if w.Code != http.StatusOK {
		t.Fatalf("update: want 200, got %d (%s)", w.Code, w.Body.String())
	}

	var got models.PayComponent
	if err := database.DB.First(&got, created.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.Taxable || got.IsBpjsBase {
		t.Fatalf("false booleans not persisted: taxable=%v isBpjsBase=%v", got.Taxable, got.IsBpjsBase)
	}
}

func TestUpdatePayComponent_rejectsBadDefaultCalc(t *testing.T) {
	newPayrollTestDB(t)
	r := newPayrollTestRouter()
	create := `{"code":"X","name":"X","componentType":"earning","defaultCalc":"fixed"}`
	w := do(r, http.MethodPost, "/api/payroll/components", bearer(t, "manager", 1), create)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: want 201, got %d (%s)", w.Code, w.Body.String())
	}
	var created models.PayComponent
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create: %v", err)
	}
	w = do(r, http.MethodPut, fmt.Sprintf("/api/payroll/components/%d", created.ID), bearer(t, "manager", 1),
		`{"defaultCalc":"bogus"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for bad defaultCalc, got %d (%s)", w.Code, w.Body.String())
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
