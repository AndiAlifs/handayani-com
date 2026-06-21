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
