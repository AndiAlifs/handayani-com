# Product Requirements Document (PRD)

**Product Name:** YPA Handayani Knowledge Base Panel & Landing Page
**Platform:** Web Application (Angular Frontend) & REST API (Golang Backend)
**Database:** MySQL
**Development Agency:** Simi Studio

---

## 1. Executive Summary

### 1.1. Product Vision
The YPA Handayani Knowledge Base Panel serves as the central nervous system for the institution's operational data. By digitizing course catalogs, instructor schedules, and administrative procedures, this system establishes a single source of truth. It empowers YPA Handayani to seamlessly distribute accurate, real-time information to a public-facing landing page and provides highly structured context to an AI RAG (Retrieval-Augmented Generation) Bot for automated customer service.

### 1.2. Problem Statement
Currently, YPA Handayani's information is locked in static assets like HTML files and image brochures. This makes updating schedules, changing prices, or modifying SIM procedures a manual, error-prone process. Furthermore, static image assets cannot be ingested by modern AI agents, preventing the automation of customer inquiries.

## 2. Target Personas

* **System Administrator (YPA Staff):** Needs a fast, unified, and intuitive interface to update course prices, add new instructors, and manage weekly schedules based on offline bookings.
* **Website Visitor (Prospective Student):** Expects a welcoming landing page to browse the most up-to-date pricing, view instructor availability, and chat with the AI Bot for onboarding.
* **The AI RAG Bot (System Consumer):** Requires an endpoint that delivers the entire database in a clean, flattened, and highly descriptive text format via standard polling so it can accurately answer human queries.

## 3. Technical Stack

* **Frontend:** Angular (Powers both the Admin Dashboard and the Public Landing Page).
* **Backend API:** Golang (RESTful architecture optimized for high concurrency and standard interval polling).
* **Database:** MySQL.

## 4. Scope & Constraints

### 4.1. In Scope for MVP
* **Unified Access Dashboard:** No complex role-based access control; all authenticated users have full Admin privileges.
* **CRUD Operations:** Manage Courses, Instructors, and SIM Mechanisms.
* **Manual Schedule Builder:** Visual grid to manually block out time slots post-confirmation.
* **Public Landing Page:** For user onboarding and displaying dynamic data.
* **REST APIs:** Standard endpoints for web data consumption and a specialized flat-text endpoint for AI bot polling.

### 4.2. Out of Scope for MVP
* Online payment gateways.
* Automated student booking/reservation system.
* Webhooks or push notifications to a Vector Database (Bot handles updates via polling).

## 5. Core Entities & Data Structure

### 5.1. Courses & Pricing
* **Categories:** Driving (Manual, Matic, Hybrid), Sewing, Computer, English, and Mandarin.
* **Data Fields:** Category, Schedule Type (Reguler/Weekend), Specifics/Level/Vehicle (e.g., Avanza/Xenia, HSK 1), Duration/Meetings, Price (Rp), Registration Fee (Rp), and Remarks (e.g., "Excludes SIM & Certificate").

### 5.2. Instructors & Schedules
* **Instructor Profiles:** Name, Gender, Age, Vehicle Model, and Transmission Type.
* **Availability Matrix:** Weekly scheduling mapping time slots (e.g., 09.00 - 12.00, 13.00 - 15.00) to days of the week (Senin - Minggu). Tracks assigned students or "Libur" (Holiday) statuses.

### 5.3. Administrative Mechanisms
* **Data Fields:** Requirement Name (e.g., Driving Certificate, Health Certificate, Psychotest), Issuing Body, Estimated Cost (Rp), and Notes.

## 6. Functional Requirements (Epics)

### Epic 1: Public Landing Page & Student Onboarding
**User Story:** As a prospective student, I want a clear landing page to browse courses and talk to the AI assistant so I can register.
* **Acceptance Criteria 1:** Page features a Hero Section introducing YPA Handayani and an embedded AI RAG Bot chat UI.
* **Acceptance Criteria 2:** Page dynamically fetches and displays active Course Pricing and Instructor Schedules from the API.
* **Acceptance Criteria 3:** Page outlines the SIM A Mechanism requirements and costs.
* **Acceptance Criteria 4:** Call-to-Action (CTA) buttons direct users to register via the official WhatsApp/Phone contacts: 082191927620 or 082193234971.

### Epic 2: Course & Pricing Management
**User Story:** As an Admin, I want to manage the catalog of courses so the website and bot always display accurate information.
* **Acceptance Criteria 1:** Admin can create, edit, or delete a course.
* **Acceptance Criteria 2:** Course entries must capture Category, Program, Specifics, Duration, Price, Registration Fee, and Remarks.

### Epic 3: Instructor & Schedule Management
**User Story:** As an Admin, I want to manage instructor profiles and their weekly schedules to show transparency.
* **Acceptance Criteria 1:** Admin can manage profiles (Name, Gender, Age, Vehicle, Transmission).
* **Acceptance Criteria 2:** Admin has access to a weekly grid (Senin - Minggu) with predefined time slots.
* **Acceptance Criteria 3:** Admin can manually type a student's name into a time slot to block it out after an offline booking is confirmed.

### Epic 4: Administrative Mechanisms
**User Story:** As an Admin, I want to outline the SIM A mechanism steps so users know requirements and costs.
* **Acceptance Criteria 1:** Admin can add/edit steps (e.g., "Health Certificate").
* **Acceptance Criteria 2:** Steps include Issuing Body, Cost, and Notes.

### Epic 5: API & System Integrations
**User Story:** As the AI RAG Bot, I need to pull the latest institutional data so I can generate accurate responses.
* **Acceptance Criteria 1:** The Golang backend must expose `GET /api/rag/knowledge-sync`.
* **Acceptance Criteria 2:** The payload must be a flattened text/markdown string optimized for LLM semantic search.
