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

		g.GET("/components", ListPayComponents)
		g.POST("/components", CreatePayComponent)
		g.PUT("/components/:id", UpdatePayComponent)
		g.DELETE("/components/:id", DeletePayComponent)

		g.GET("/employees/:id/components", ListEmployeeComponents)
		g.POST("/employees/:id/components", AssignEmployeeComponent)
		g.DELETE("/employee-components/:id", DeleteEmployeeComponent)
	}
}
