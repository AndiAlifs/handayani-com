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
