package main

import (
	"log"

	"field-attendance-system/auth"
	"field-attendance-system/database"
	"field-attendance-system/handlers"
	"field-attendance-system/models"
	"field-attendance-system/seed"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file in development
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	database.Connect()

	// Auto-migrate models
	database.DB.AutoMigrate(
		&models.User{},
		&models.Attendance{},
		&models.LeaveRequest{},
		&models.OfficeLocation{},
		&models.ManagerOffice{},
		&models.SystemSettings{},
		&models.Student{},
		&models.StudentSession{},
		&models.LearningPlan{},
	)

	// Seed database with initial data
	seed.RunAll()

	r := gin.Default()

	// CORS config to allow frontend to communicate
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:4200", "http://43.163.107.154"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Public routes
	r.POST("/api/login", handlers.Login)
	r.POST("/api/register", handlers.Register)

	// Protected routes
	protected := r.Group("/api")
	protected.Use(auth.AuthMiddleware())
	{
		protected.POST("/clock-in", handlers.ClockIn)
		protected.POST("/clock-out", handlers.ClockOut)
		protected.POST("/leave", handlers.CreateLeaveRequest)
		protected.GET("/office-location", handlers.GetOfficeLocation)
		protected.GET("/my-attendance/today", handlers.GetTodayAttendance)
		protected.GET("/my-attendance/history", handlers.GetMyAttendanceHistory)
		protected.GET("/my-leave/today", handlers.GetTodayLeave)
		protected.GET("/my-leave/history", handlers.GetMyLeaveHistory)
		protected.GET("/my-offices", handlers.GetEmployeeOffices)

		// Manager routes
		admin := protected.Group("/admin")
		admin.Use(auth.ManagerMiddleware())
		{
			admin.GET("/records", handlers.GetAllRecords)
			admin.GET("/leaves", handlers.GetAllLeaveRequests)
			admin.PATCH("/leave/:id", handlers.UpdateLeaveStatus)
			admin.POST("/users", handlers.CreateUser)
			admin.GET("/employees", handlers.GetAllEmployees)
			admin.POST("/employees", handlers.CreateEmployee)
			admin.PUT("/employees/:id", handlers.UpdateEmployee)
			admin.DELETE("/employees/:id", handlers.DeleteEmployee)
			admin.POST("/office-location", handlers.SetOfficeLocation)
			admin.GET("/pending-clockins", handlers.GetPendingClockIns)
			admin.PATCH("/clockin/:id", handlers.UpdateClockInStatus)
			admin.GET("/daily-attendance", handlers.GetDailyAttendanceDashboard)

			// Office Management Routes
			admin.GET("/offices", handlers.GetAllOffices)
			admin.POST("/offices", handlers.CreateOffice)
			admin.PUT("/offices/:id", handlers.UpdateOffice)
			admin.DELETE("/offices/:id", handlers.DeleteOffice)
			admin.GET("/my-offices", handlers.GetManagerOffices)
			admin.POST("/offices/assign", handlers.AssignOfficeToManager)
			admin.POST("/offices/unassign", handlers.UnassignOfficeFromManager)

			// System Settings Routes
			admin.GET("/settings", handlers.GetSystemSettings)
			admin.GET("/settings/session-duration", handlers.GetSessionDuration)
			admin.PUT("/settings/session-duration", handlers.UpdateSessionDuration)
			admin.GET("/settings/minimum-work-hours", handlers.GetMinimumWorkHours)
			admin.PUT("/settings/minimum-work-hours", handlers.UpdateMinimumWorkHours)
			admin.GET("/settings/quota-presets", handlers.GetQuotaPresetsSetting)
			admin.PUT("/settings/quota-presets", handlers.UpdateQuotaPresets)

			// Student Management (admin-owned)
			admin.POST("/students", handlers.AdminCreateStudent)
			admin.GET("/students", handlers.AdminGetStudents)
			admin.GET("/students/roster.xlsx", handlers.AdminExportStudentRoster)
			admin.PUT("/students/:id", handlers.AdminUpdateStudent)
			admin.PUT("/students/:id/adjust-quota", handlers.AdminAdjustStudentQuota)
			admin.PUT("/students/:id/archive", handlers.AdminArchiveStudent)
			admin.PUT("/students/:id/reassign", handlers.AdminReassignStudent)
			admin.GET("/students/:id/sessions", handlers.AdminGetStudentSessions)

			// Learning Plan Management (admin-owned)
			admin.POST("/learning-plans", handlers.AdminCreateLearningPlan)
			admin.GET("/learning-plans", handlers.AdminGetLearningPlans)
			admin.POST("/learning-plans/bulk", handlers.AdminBulkCreateLearningPlan)
			admin.PUT("/learning-plans/:id", handlers.AdminUpdateLearningPlan)
			admin.DELETE("/learning-plans/:id", handlers.AdminDeleteLearningPlan)

			// Instructor Insight
			admin.GET("/instructors", handlers.AdminListInstructors)
			admin.GET("/instructor-load", handlers.AdminGetInstructorLoad)
		}

		// Instructor routes
		instructorGroup := protected.Group("/instructor")
		instructorGroup.Use(auth.InstructorMiddleware())
		{
			instructorGroup.GET("/students", handlers.GetStudents)
			instructorGroup.GET("/students/:id/sessions", handlers.GetStudentSessions)

			instructorGroup.GET("/schedule", handlers.GetLearningPlans)
			instructorGroup.PUT("/schedule/:id", handlers.UpdateLearningPlan)

			instructorGroup.POST("/session/start", handlers.StartStudentSession)
			instructorGroup.POST("/session/end", handlers.EndStudentSession)
			instructorGroup.GET("/session/active", handlers.GetActiveStudentSession)

			instructorGroup.GET("/quota-presets", handlers.GetQuotaPresets)
		}
	}

	log.Println("Server starting on port 8080...")
	r.Run(":8080")
}
