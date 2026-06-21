package handlers

// Small shared validators for the payroll handlers. Payroll models store
// enum-like fields as varchar, so membership is validated here in the app layer.

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
