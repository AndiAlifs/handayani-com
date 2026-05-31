package seed

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"field-attendance-system/database"
	"field-attendance-system/models"
	"field-attendance-system/utils"

	"golang.org/x/crypto/bcrypt"
)

// RunAll executes all seeding functions in order
func RunAll() {
	log.Println("Starting database seeding...")
	SeedSuperAdmins()
	SeedInstructors()
	SeedOffices()
	SeedDefaultOfficeAssignment()
	SeedEmployees()
	SeedInstructorStudentsAndPlans()
	SeedAttendanceRecords()
	SeedSystemSettings()
	log.Println("Database seeding completed!")
}

// SeedSuperAdmins creates admin users (1 super admin, 1 regular manager)
func SeedSuperAdmins() {
	admins := []struct {
		username     string
		fullName     string
		password     string
		isSuperAdmin bool
	}{
		{"admin", "Administrator 1", "admin", true},
		{"admin2", "Administrator 2", "admin2", false},                    // Regular manager, not super admin
		{"admin_kendari", "Admin Kantor Kendari", "admin_kendari", false}, // Kendari office manager
	}

	for _, admin := range admins {
		// Check if admin user already exists
		var existingUser models.User
		result := database.DB.Where("username = ?", admin.username).First(&existingUser)

		if result.Error == nil {
			log.Printf("Admin user '%s' already exists, skipping", admin.username)
			continue
		}

		// Create admin user
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(admin.password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Failed to hash password for %s: %v", admin.username, err)
			continue
		}

		adminUser := models.User{
			Username:     admin.username,
			FullName:     admin.fullName,
			PasswordHash: string(hashedPassword),
			Role:         "manager",
			IsSuperAdmin: admin.isSuperAdmin,
		}

		if err := database.DB.Create(&adminUser).Error; err != nil {
			log.Printf("Failed to create admin user %s: %v", admin.username, err)
			continue
		}

		roleType := "Regular Manager"
		if admin.isSuperAdmin {
			roleType = "Super Admin"
		}

		log.Printf("✓ Admin user '%s' created successfully", admin.username)
		log.Printf("  Username: %s", admin.username)
		log.Printf("  Password: %s", admin.password)
		log.Printf("  Role: manager (%s)", roleType)
	}
}

// SeedOffices creates the initial office locations
func SeedOffices() {
	offices := []struct {
		name        string
		address     string
		latitude    float64
		longitude   float64
		radius      float64
		clockInTime string
	}{
		{"Kantor Kendari", "", -3.98929160, 122.50396530, 300.00, "05:30"},
		{"Kantor Jakarta", "", -6.17433700, 106.79221800, 350.00, "09:20"},
		{"Kantor Palopo", "", -3.00378910, 120.18998010, 300.00, "02:12"},
		{"Kantor Makassar", "", -5.15187740, 119.44655790, 300.00, "10:00"},
		{"Kampus B", "", -3.989306312396674, 122.50644229642265, 300.00, "08:00"},
		{"Lapangan MTQ", "", -3.9742242989210776, 122.51430769144108, 300.00, "08:00"},
		{"Kampus A", "", -3.961466180535831, 122.53117878627592, 300.00, "08:00"},
	}

	for _, office := range offices {
		// Check if office already exists
		var existingOffice models.OfficeLocation
		result := database.DB.Where("name = ?", office.name).First(&existingOffice)

		if result.Error == nil {
			log.Printf("Office '%s' already exists, skipping", office.name)
			continue
		}

		newOffice := models.OfficeLocation{
			Name:                office.name,
			Address:             office.address,
			Latitude:            office.latitude,
			Longitude:           office.longitude,
			AllowedRadiusMeters: office.radius,
			ClockInTime:         office.clockInTime,
			IsActive:            true,
		}

		if err := database.DB.Create(&newOffice).Error; err != nil {
			log.Printf("Failed to create office %s: %v", office.name, err)
			continue
		}

		log.Printf("✓ Office '%s' created successfully", office.name)
	}
}

// SeedDefaultOfficeAssignment assigns offices to managers
func SeedDefaultOfficeAssignment() {
	// Get all admins
	var admin1, admin2, adminKendari models.User
	if err := database.DB.Where("username = ?", "admin").First(&admin1).Error; err != nil {
		log.Println("Admin user not found, skipping office assignment")
		return
	}
	if err := database.DB.Where("username = ?", "admin2").First(&admin2).Error; err != nil {
		log.Println("Admin2 user not found, skipping office assignment")
		return
	}
	if err := database.DB.Where("username = ?", "admin_kendari").First(&adminKendari).Error; err != nil {
		log.Println("Admin Kendari user not found, skipping office assignment")
		return
	}

	// Get all offices
	var offices []models.OfficeLocation
	if err := database.DB.Find(&offices).Error; err != nil {
		log.Println("No offices found, skipping assignment")
		return
	}

	if len(offices) < 7 {
		log.Println("Not enough offices, skipping assignment")
		return
	}

	// Helper function to assign office to manager
	assignOffice := func(managerID uint, office models.OfficeLocation, managerName string) {
		var existingAssignment models.ManagerOffice
		if err := database.DB.Where("manager_id = ? AND office_id = ?", managerID, office.ID).
			First(&existingAssignment).Error; err != nil {
			assignment := models.ManagerOffice{
				ManagerID: managerID,
				OfficeID:  office.ID,
			}
			if err := database.DB.Create(&assignment).Error; err == nil {
				log.Printf("✓ Office '%s' assigned to %s", office.Name, managerName)
			}
		}
	}

	// Assign first 2 offices to admin1 (Kendari, Jakarta)
	for i := 0; i < 2; i++ {
		assignOffice(admin1.ID, offices[i], "admin")
	}

	// Assign offices 2-3 to admin2 (Palopo, Makassar)
	for i := 2; i < 4; i++ {
		assignOffice(admin2.ID, offices[i], "admin2")
	}

	// Assign offices 4-6 to admin_kendari (Kampus B, Lapangan MTQ, Kampus A)
	for i := 4; i < 7; i++ {
		assignOffice(adminKendari.ID, offices[i], "admin_kendari")
	}
}

// SeedEmployees creates initial employees assigned to different offices
func SeedEmployees() {
	// Get all offices
	var offices []models.OfficeLocation
	if err := database.DB.Find(&offices).Error; err != nil || len(offices) < 7 {
		log.Println("Not enough offices found, skipping employee creation")
		return
	}

	employeeNames := []string{
		"Andi Prasetyo",
		"Budi Santoso",
		"Citra Dewi",
		"Dian Kartika",
		"Eko Wijaya",
		"Fitri Handayani",
		"Gilang Ramadhan",
	}

	// Assign employees to different offices:
	// karyawan1, karyawan2 -> Kendari (office 0) - managed by admin
	// karyawan3 -> Jakarta (office 1) - managed by admin
	// karyawan4, karyawan5 -> Palopo (office 2) - managed by admin2
	// karyawan6, karyawan7 -> Kendari (office 0) - managed by admin
	officeAssignments := []int{0, 0, 1, 2, 2, 0, 0}

	for i := 1; i <= 7; i++ {
		username := fmt.Sprintf("karyawan%d", i)

		// Check if employee already exists
		var existingUser models.User
		result := database.DB.Where("username = ?", username).First(&existingUser)

		if result.Error == nil {
			log.Printf("Employee '%s' already exists, skipping", username)
			continue
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(username), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Failed to hash password for %s: %v", username, err)
			continue
		}

		officeIndex := officeAssignments[i-1]
		officeID := offices[officeIndex].ID

		employee := models.User{
			Username:     username,
			FullName:     employeeNames[i-1],
			PasswordHash: string(hashedPassword),
			Role:         "employee",
			OfficeID:     &officeID,
			IsSuperAdmin: false,
		}

		if err := database.DB.Create(&employee).Error; err != nil {
			log.Printf("Failed to create employee %s: %v", username, err)
			continue
		}

		log.Printf("✓ Employee '%s' created successfully", username)
		log.Printf("  Username: %s", username)
		log.Printf("  Password: %s", username)
		log.Printf("  Full Name: %s", employeeNames[i-1])
		log.Printf("  Office: %s", offices[officeIndex].Name)
	}

	// Create hidayat employee for admin_kendari (assigned to Kampus B - office 4)
	var existingHidayat models.User
	if err := database.DB.Where("username = ?", "hidayat").First(&existingHidayat).Error; err != nil {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("hidayat"), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Failed to hash password for hidayat: %v", err)
		} else {
			kampusBID := offices[4].ID // Kampus B
			hidayat := models.User{
				Username:     "hidayat",
				FullName:     "Andi Hidayat Hasim",
				PasswordHash: string(hashedPassword),
				Role:         "employee",
				OfficeID:     &kampusBID,
				IsSuperAdmin: false,
			}

			if err := database.DB.Create(&hidayat).Error; err != nil {
				log.Printf("Failed to create employee hidayat: %v", err)
			} else {
				log.Printf("✓ Employee 'hidayat' created successfully")
				log.Printf("  Username: hidayat")
				log.Printf("  Password: hidayat")
				log.Printf("  Full Name: Andi Hidayat Hasim")
				log.Printf("  Office: %s", offices[4].Name)
			}
		}
	} else {
		log.Printf("Employee 'hidayat' already exists, skipping")
	}
}

// SeedAttendanceRecords creates attendance records for the last 7 days
func SeedAttendanceRecords() {
	// Get all employees
	var employees []models.User
	if err := database.DB.Where("role = ?", "employee").Find(&employees).Error; err != nil {
		log.Println("No employees found, skipping attendance records")
		return
	}

	if len(employees) == 0 {
		log.Println("No employees found, skipping attendance records")
		return
	}

	// Get all offices
	var offices []models.OfficeLocation
	if err := database.DB.Find(&offices).Error; err != nil || len(offices) == 0 {
		log.Println("No offices found, skipping attendance records")
		return
	}

	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	recordsCreated := 0
	rejectedCount := 0
	pendingCount := 0

	// Create records for last 7 days
	for day := 0; day < 7; day++ {
		recordDate := time.Now().AddDate(0, 0, -day)

		// Randomly select 6-7 employees to have attendance each day
		numRecords := 6 + rand.Intn(2)
		shuffledEmployees := make([]models.User, len(employees))
		copy(shuffledEmployees, employees)
		rand.Shuffle(len(shuffledEmployees), func(i, j int) {
			shuffledEmployees[i], shuffledEmployees[j] = shuffledEmployees[j], shuffledEmployees[i]
		})

		// Fetch all existing attendance user IDs for this day upfront
		startOfDay := time.Date(recordDate.Year(), recordDate.Month(), recordDate.Day(), 0, 0, 0, 0, recordDate.Location())
		nextDay := startOfDay.AddDate(0, 0, 1)
		var existingRecords []models.Attendance
		database.DB.Select("user_id").
			Where("clock_in_time >= ? AND clock_in_time < ?", startOfDay, nextDay).
			Find(&existingRecords)
		existingUserIDs := make(map[uint]bool, len(existingRecords))
		for _, r := range existingRecords {
			existingUserIDs[r.UserID] = true
		}

		for i := 0; i < numRecords && i < len(shuffledEmployees); i++ {
			employee := shuffledEmployees[i]

			// Check if attendance record already exists for this user on this day
			if existingUserIDs[employee.ID] {
				log.Printf("Attendance record for employee %d on %s already exists, skipping", employee.ID, recordDate.Format("2006-01-02"))
				continue
			}

			office := offices[rand.Intn(len(offices))]

			// Determine status with constraints
			var status string
			if rejectedCount < 3 && rand.Float32() < 0.1 {
				status = "rejected"
				rejectedCount++
			} else if pendingCount < 3 && rand.Float32() < 0.1 {
				status = "pending"
				pendingCount++
			} else {
				status = "approved"
			}

			// Random location near office (with some variance)
			latVariance := (rand.Float64() - 0.5) * 0.01
			longVariance := (rand.Float64() - 0.5) * 0.01
			clockInLat := office.Latitude + latVariance
			clockInLong := office.Longitude + longVariance

			// Random clock-in time (between 05:00 and 11:00)
			hour := 5 + rand.Intn(7)
			minute := rand.Intn(60)
			clockInTime := time.Date(recordDate.Year(), recordDate.Month(), recordDate.Day(),
				hour, minute, 0, 0, recordDate.Location())

			// Determine if late based on office clock-in time
			isLate := false
			minutesLate := 0
			// Simple comparison - consider late if after 08:00
			if hour > 8 || (hour == 8 && minute > 0) {
				isLate = true
				minutesLate = (hour-8)*60 + minute
			}

			// Calculate actual distance from office
			distance := utils.CalculateDistance(clockInLat, clockInLong, office.Latitude, office.Longitude)

			var approvedOfficeID *uint
			if status == "approved" {
				approvedOfficeID = &office.ID
			}

			// Generate clock out time (8-9 hours after clock in, 80% chance of having clock out)
			var clockOutTime *time.Time
			var clockOutLat *float64
			var clockOutLong *float64
			var workHours *float64

			if rand.Float32() < 0.8 { // 80% of records have clock out
				// Random work duration: 8-9 hours
				workDuration := 8 + rand.Float64()
				clockOut := clockInTime.Add(time.Duration(workDuration * float64(time.Hour)))
				clockOutTime = &clockOut

				// Clock out location near office (with some variance)
				clockOutLatVariance := (rand.Float64() - 0.5) * 0.01
				clockOutLongVariance := (rand.Float64() - 0.5) * 0.01
				clockOutLatValue := office.Latitude + clockOutLatVariance
				clockOutLongValue := office.Longitude + clockOutLongVariance
				clockOutLat = &clockOutLatValue
				clockOutLong = &clockOutLongValue

				// Calculate work hours
				workHoursValue := workDuration
				workHours = &workHoursValue
			}

			attendance := models.Attendance{
				UserID:           employee.ID,
				ClockInTime:      clockInTime,
				ClockOutTime:     clockOutTime,
				Latitude:         clockInLat,
				Longitude:        clockInLong,
				LatitudeOut:      clockOutLat,
				LongitudeOut:     clockOutLong,
				Status:           status,
				Distance:         distance,
				WorkHours:        workHours,
				ApprovedOfficeID: approvedOfficeID,
				IsLate:           isLate,
				MinutesLate:      minutesLate,
			}

			if err := database.DB.Create(&attendance).Error; err != nil {
				log.Printf("Failed to create attendance record: %v", err)
				continue
			}

			recordsCreated++
		}
	}

	log.Printf("✓ Created %d attendance records", recordsCreated)
	log.Printf("  - Approved: %d", recordsCreated-rejectedCount-pendingCount)
	log.Printf("  - Rejected: %d", rejectedCount)
	log.Printf("  - Pending: %d", pendingCount)
}

// SeedSystemSettings creates default system settings
func SeedSystemSettings() {
	defaultSettings := []struct {
		key         string
		value       string
		description string
	}{
		{models.SettingSessionDurationHours, "24", "Durasi sesi login default (jam)"},
		{models.SettingQuotaPresetOptions, "8,10", "Opsi preset kuota murid (jam, dipisahkan koma)"},
	}

	for _, s := range defaultSettings {
		var existing models.SystemSettings
		if err := database.DB.Where("setting_key = ?", s.key).First(&existing).Error; err == nil {
			log.Printf("System setting '%s' already exists, skipping", s.key)
			continue
		}

		setting := models.SystemSettings{
			SettingKey:   s.key,
			SettingValue: s.value,
			Description:  s.description,
		}

		if err := database.DB.Create(&setting).Error; err != nil {
			log.Printf("Failed to create system setting '%s': %v", s.key, err)
			continue
		}

		log.Printf("✓ Created default system setting: %s = %s", s.key, s.value)
	}
}

// SeedInstructors creates default instructor users.
func SeedInstructors() {
	instructors := []struct {
		username string
		fullName string
		password string
	}{
		{"instructor1", "Instruktur Utama", "instructor1"},
	}

	for _, instructor := range instructors {
		var existingUser models.User
		if err := database.DB.Where("username = ?", instructor.username).First(&existingUser).Error; err == nil {
			log.Printf("Instructor user '%s' already exists, skipping", instructor.username)
			continue
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(instructor.password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Failed to hash password for %s: %v", instructor.username, err)
			continue
		}

		user := models.User{
			Username:     instructor.username,
			FullName:     instructor.fullName,
			PasswordHash: string(hashedPassword),
			Role:         "instructor",
			IsSuperAdmin: false,
		}

		if err := database.DB.Create(&user).Error; err != nil {
			log.Printf("Failed to create instructor user %s: %v", instructor.username, err)
			continue
		}

		log.Printf("✓ Instructor user '%s' created successfully", instructor.username)
		log.Printf("  Username: %s", instructor.username)
		log.Printf("  Password: %s", instructor.password)
		log.Printf("  Role: instructor")
	}
}

// SeedInstructorStudentsAndPlans creates sample students, learning plans, and session data for instructor users.
func SeedInstructorStudentsAndPlans() {
	var instructor models.User
	if err := database.DB.Where("username = ? AND role = ?", "instructor1", "instructor").First(&instructor).Error; err != nil {
		log.Println("Instructor user 'instructor1' not found, skipping instructor student/plan seed")
		return
	}

	studentSeeds := []struct {
		name         string
		quota        float64
		whatsapp     string
		gender       string
		meetingPoint string
	}{
		{"Aulia Rahman", 24, "081234567890", "male", "Kampus B"},
		{"Nadia Putri", 16, "081234567891", "female", "Lapangan MTQ"},
		{"Rizky Pratama", 20, "081234567892", "male", ""},
	}

	studentByName := make(map[string]models.Student)
	for _, s := range studentSeeds {
		var student models.Student
		err := database.DB.Where("instructor_id = ? AND name = ?", instructor.ID, s.name).First(&student).Error
		if err == nil {
			studentByName[s.name] = student
			log.Printf("Student '%s' already exists for instructor '%s', skipping", s.name, instructor.Username)
			continue
		}

		student = models.Student{
			Name:                s.name,
			InstructorID:        instructor.ID,
			TotalQuotaHours:     s.quota,
			RemainingQuotaHours: s.quota,
			WhatsApp:            s.whatsapp,
			Gender:              s.gender,
			MeetingPoint:        s.meetingPoint,
			IsActive:            true,
		}

		if err := database.DB.Create(&student).Error; err != nil {
			log.Printf("Failed to create student '%s': %v", s.name, err)
			continue
		}

		studentByName[s.name] = student
		log.Printf("✓ Student '%s' created for instructor '%s'", s.name, instructor.Username)
	}

	var existingStudents []models.Student
	if err := database.DB.Where("instructor_id = ?", instructor.ID).Find(&existingStudents).Error; err == nil {
		for _, st := range existingStudents {
			studentByName[st.Name] = st
		}
	}

	if len(studentByName) == 0 {
		log.Printf("No students found for instructor '%s', skipping learning plan seed", instructor.Username)
		return
	}

	today := time.Now()
	plans := []struct {
		studentName string
		date        time.Time
		startTime   string
		endTime     string
		status      string
	}{
		{"Aulia Rahman", today.AddDate(0, 0, 1), "09:00", "11:00", "planned"},
		{"Nadia Putri", today.AddDate(0, 0, 2), "13:00", "15:00", "planned"},
		{"Rizky Pratama", today.AddDate(0, 0, -1), "08:30", "10:00", "completed"},
	}

	for _, p := range plans {
		student, ok := studentByName[p.studentName]
		if !ok {
			continue
		}

		scheduledDate := time.Date(p.date.Year(), p.date.Month(), p.date.Day(), 0, 0, 0, 0, p.date.Location())

		var existingPlan models.LearningPlan
		err := database.DB.Where(
			"instructor_id = ? AND student_id = ? AND scheduled_date = ? AND start_time = ? AND end_time = ?",
			instructor.ID,
			student.ID,
			scheduledDate,
			p.startTime,
			p.endTime,
		).First(&existingPlan).Error
		if err == nil {
			log.Printf("Learning plan for '%s' on %s already exists, skipping", p.studentName, scheduledDate.Format("2006-01-02"))
			continue
		}

		plan := models.LearningPlan{
			InstructorID:  instructor.ID,
			StudentID:     student.ID,
			ScheduledDate: scheduledDate,
			StartTime:     p.startTime,
			EndTime:       p.endTime,
			Status:        p.status,
		}

		if err := database.DB.Create(&plan).Error; err != nil {
			log.Printf("Failed to create learning plan for '%s': %v", p.studentName, err)
			continue
		}

		log.Printf("✓ Learning plan created for '%s' (%s %s-%s)", p.studentName, scheduledDate.Format("2006-01-02"), p.startTime, p.endTime)
	}

	seedInstructorSampleSessions(instructor, studentByName)
}

func seedInstructorSampleSessions(instructor models.User, studentByName map[string]models.Student) {
	sampleStudent, ok := studentByName["Aulia Rahman"]
	if !ok {
		return
	}

	var existingCompleted int64
	database.DB.Model(&models.StudentSession{}).
		Where("instructor_id = ? AND student_id = ? AND check_out_time IS NOT NULL", instructor.ID, sampleStudent.ID).
		Count(&existingCompleted)

	if existingCompleted == 0 {
		checkIn := time.Now().Add(-4 * time.Hour)
		checkOut := checkIn.Add(90 * time.Minute)
		deducted := checkOut.Sub(checkIn).Hours()

		session := models.StudentSession{
			StudentID:     sampleStudent.ID,
			InstructorID:  instructor.ID,
			CheckInTime:   checkIn,
			CheckOutTime:  &checkOut,
			DeductedHours: deducted,
			Latitude:      -3.98929160,
			Longitude:     122.50396530,
		}

		if err := database.DB.Create(&session).Error; err != nil {
			log.Printf("Failed to create completed sample session: %v", err)
		} else {
			// Keep quota realistic if session was newly seeded.
			if sampleStudent.RemainingQuotaHours >= deducted {
				sampleStudent.RemainingQuotaHours -= deducted
				_ = database.DB.Save(&sampleStudent).Error
			}
			log.Printf("✓ Completed sample student session created for '%s'", sampleStudent.Name)
		}
	}

	var existingActive int64
	database.DB.Model(&models.StudentSession{}).
		Where("instructor_id = ? AND check_out_time IS NULL", instructor.ID).
		Count(&existingActive)

	if existingActive == 0 {
		activeStudent, found := studentByName["Nadia Putri"]
		if !found {
			return
		}

		activeSession := models.StudentSession{
			StudentID:    activeStudent.ID,
			InstructorID: instructor.ID,
			CheckInTime:  time.Now().Add(-25 * time.Minute),
			Latitude:     -3.98930631,
			Longitude:    122.50644229,
		}

		if err := database.DB.Create(&activeSession).Error; err != nil {
			log.Printf("Failed to create active sample session: %v", err)
		} else {
			log.Printf("✓ Active sample student session created for '%s'", activeStudent.Name)
		}
	}
}
