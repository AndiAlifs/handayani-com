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
