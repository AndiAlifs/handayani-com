### Epic 1: Public Landing Page & Student Onboarding
**Focus:** Building the user-facing UI and consuming the backend APIs.

* **Task 1.1:** Initialize the Angular component for the Landing Page (`ng generate component landing-page`) and set up the routing.
* **Task 1.2:** Build the Hero Section UI (HTML/CSS) introducing YPA Handayani.
* **Task 1.3:** Build and embed the AI RAG Bot chat interface UI (chat window, input field, send button).
* **Task 1.4:** Create an Angular HTTP Service to fetch public data from the Python/FastAPI backend (`GET /api/courses` and `GET /api/instructors/schedule`).
* **Task 1.5:** Build the UI components (tables or cards) to display the fetched Course Pricing and Instructor Schedules dynamically.
* **Task 1.6:** Build a static or dynamic UI section outlining the SIM A Mechanism requirements and costs.
* **Task 1.7:** Add the Call-to-Action (CTA) buttons linking directly to the official WhatsApp numbers (using `https://wa.me/` links for 082191927620 and 082193234971).

### Epic 2: Course & Pricing Management
**Focus:** Full-stack CRUD (Create, Read, Update, Delete) for the course catalog.

* **Task 2.1 (Database):** Create the `courses` table in MySQL with columns: `id`, `category`, `program_type`, `specifics`, `duration`, `price`, `registration_fee`, and `remarks`.
* **Task 2.2 (Backend):** Define the Pydantic Course model and set up the database connection.
* **Task 2.3 (Backend):** Implement the FastAPI REST API route handlers for CRUD operations (`GET`, `POST`, `PUT`, `DELETE` on `/api/courses`).
* **Task 2.4 (Frontend):** Create the Angular Course Management component, featuring a data grid/table to list all existing courses.
* **Task 2.5 (Frontend):** Build the Angular reactive form to input course details (Category, Price, etc.) for creating and editing records.
* **Task 2.6 (Integration):** Connect the Angular form and data grid to the FastAPI backend API to fully enable data management.

### Epic 3: Instructor & Schedule Management
**Focus:** Relational data and a custom grid UI for scheduling.

* **Task 3.1 (Database):** Create the `instructors` table and the `schedules` table in MySQL. Ensure the `schedules` table has a foreign key linking to the `instructor_id`.
* **Task 3.2 (Backend):** Define the Pydantic Instructor and Schedule models.
* **Task 3.3 (Backend):** Implement the FastAPI REST API handlers for Instructor CRUD (`/api/instructors`).
* **Task 3.4 (Backend):** Implement the FastAPI REST API handlers for updating the schedule matrix (`GET` and `POST`/`PUT` on `/api/instructors/{id}/schedule`).
* **Task 3.5 (Frontend):** Create the Angular Instructor Management component (a form and list for Name, Gender, Age, Vehicle, Transmission).
* **Task 3.6 (Frontend):** Build the Angular Weekly Schedule Grid UI. It needs columns for days (Senin - Minggu) and rows for predefined time slots.
* **Task 3.7 (Frontend):** Add interactive logic to the Schedule Grid: Admin clicks a specific cell, types a student's name or "Libur", and saves that specific block to the backend.

### Epic 4: Administrative Mechanisms
**Focus:** Simple full-stack CRUD for the SIM A process steps.

* **Task 4.1 (Database):** Create the `mechanisms` table in MySQL with columns: `id`, `requirement_name`, `issuing_body`, `cost`, and `notes`.
* **Task 4.2 (Backend):** Define the Pydantic Mechanism model.
* **Task 4.3 (Backend):** Implement the FastAPI REST API handlers for CRUD operations (`GET`, `POST`, `PUT`, `DELETE` on `/api/mechanisms`).
* **Task 4.4 (Frontend):** Create the Angular Mechanism Management component containing a grid to list the current steps.
* **Task 4.5 (Frontend):** Build the Angular form to add or edit mechanism steps.
* **Task 4.6 (Integration):** Connect the Angular mechanism UI to the FastAPI API.
