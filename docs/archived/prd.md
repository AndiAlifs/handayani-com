# Handayani.com — All-in-One Driving School Management & AI Platform

**Date:** 2025-01-01 · **Status:** Archived — founding PRD, platform implemented · **Owner:** Andi Alifsyah

> **Master Product Requirements Document (PRD)**

---

## 📌 1. Executive Summary & Product Vision
**Handayani.com** is a comprehensive, end-to-end web platform designed to digitize and automate the operations of "Kursus Mengemudi Handayani" (Handayani Driving School). 

The platform serves a dual purpose:
1. **Customer-Facing Portal:** A modern landing page and customer portal where prospective students can browse courses, check instructor availability, learn about the SIM (driver's license) mechanism, and interact with an AI Chatbot for instant inquiries.
2. **Internal Management Dashboard:** A robust system for admins and instructors to handle CRM, scheduling, and session management.

**The "AI Advantage"**: This platform is heavily integrated with AI. An autonomous agent (powered by Gemini & LangChain) acts as a virtual assistant that handles customer support via the chatbot, and deeply integrates into Session Management to analyze instructor notes, suggest follow-up courses, and auto-update the CRM without manual data entry.

---

## 🏗️ 2. Technical Architecture & Tech Stack

| Layer | Technology | Description |
| :--- | :--- | :--- |
| **Frontend** | Angular 18 | Single Page Application (SPA) serving both the landing page and the secure admin dashboard. |
| **Backend** | Python (FastAPI) | High-performance backend handling API requests, business logic, and database transactions. |
| **Database** | MySQL | Relational database for structured storage of CRM data, schedules, and courses. |
| **AI Framework** | LangChain (Python) | Orchestrates tool-calling and agentic workflows. |
| **LLM Model** | Gemini API | The brain behind the chatbot and session analysis. |
| **Automation** | n8n API | Workflow automation tool to handle webhooks, notifications, and background integrations. |

---

## 👥 3. Core User Personas

1. **The Student (Customer)**
   - **Goal:** Find a suitable driving course, understand pricing, check schedules, and register.
   - **Needs:** Fast, responsive UI; instant answers to questions (via Chatbot); clear pricing and SIM mechanisms.
2. **The Instructor**
   - **Goal:** View daily schedules, manage student sessions, and log post-session notes.
   - **Needs:** Mobile-friendly dashboard, low administrative overhead (aided by AI auto-updating CRM).
3. **The Administrator**
   - **Goal:** Oversee the business, manage the fleet, track revenue, and monitor CRM health.
   - **Needs:** High-level analytics, master calendar, and full control over course offerings.

---

## 📋 4. Feature Specifications

### A. Customer-Facing Portal (Landing Page)
- **Hero Section:** High-conversion call-to-action to book a course.
- **Course Pricing:** Dynamic list of available packages (e.g., Manual vs. Automatic, 5 vs. 10 sessions).
- **Instructor Schedule View:** Public view of instructors and high-level availability.
- **SIM Mechanism Guide:** Educational section on how to get a driver's license.
- **AI Chatbot (Widget):** 
  - Trained on Handayani's FAQs, pricing, and policies.
  - Can capture lead information (Name, Phone) and send it directly to the CRM via backend APIs.

### B. Admin & Instructor Dashboard
- **Authentication & Role Management:** Secure login (Admin vs. Instructor).
- **Master Calendar (Scheduling):** Drag-and-drop calendar for booking driving sessions, assigning cars, and assigning instructors.
- **CRM (Customer Relationship Management):** 
  - Track leads (from chatbot/forms) through the pipeline (Lead -> Registered -> Completed).
  - View student history, payment status, and progress.
- **Session Management:**
  - Instructors can view their upcoming students.
  - Form to input "Session Notes" (e.g., "Student struggles with parallel parking, but clutch control is improving.").

### C. Agentic AI Integrations (The "Smart" Layer)
- **AI Session Analyst:** 
  - *Trigger:* When an instructor submits plain-text Session Notes.
  - *Action:* The LangChain agent processes the text using Gemini API.
  - *Output:* It automatically extracts structured data (Strengths, Weaknesses), suggests the focus for the *next* session, and **auto-updates the student's CRM profile**.
  - *Upsell logic:* If the AI detects the student is still struggling at the end of their package, it generates a draft recommendation for a "Top-up / Follow-up Course" to send to the student.

---

## 🗄️ 5. High-Level Database Schema (MySQL)

- **`users`**: id, name, role (admin/instructor/student), email, phone, created_at
- **`courses`**: id, name, type (manual/auto), price, total_sessions
- **`students_crm`**: id, user_id, status (lead/active/completed), notes, progress_score
- **`sessions`**: id, student_id, instructor_id, start_time, end_time, status (scheduled/completed/cancelled)
- **`session_logs`**: id, session_id, raw_instructor_notes, ai_structured_analysis, recommended_next_steps

---

## 🔗 6. APIs and Workflow Orchestration

1. **Python FastAPI Backend:** Serves REST endpoints (e.g., `POST /api/sessions/notes`, `GET /api/courses`).
2. **LangChain Tools:** 
   - `UpdateCRMTool`: Allows the Gemini agent to execute SQL updates to the CRM based on natural language reasoning.
   - `CheckScheduleTool`: Allows the chatbot to check MySQL for available slots.
3. **n8n Automation:**
   - Triggers WhatsApp or Email notifications to students when the AI suggests a follow-up course.
   - Sends daily schedule summaries to instructors via email/messaging.

---

## 🚀 7. Proposed Development Phases

### Phase 1: Foundation & Landing Page
- Initialize Python/FastAPI backend & MySQL schema.
- Finalize Angular Landing Page components (Pricing, Hero, SIM Mechanism).
- Connect Frontend to Backend for dynamic course/pricing data.

### Phase 2: Dashboard & Core CRUD
- Implement Admin/Instructor Authentication.
- Build the CRM interface and Session Management views in Angular.
- Build standard CRUD APIs in Python.

### Phase 3: AI & Automation
- Integrate LangChain and Gemini API into the Python backend.
- Build the public-facing Chatbot capable of RAG (Retrieval-Augmented Generation) on Handayani FAQs.
- Implement the **AI Session Analyst** to auto-process instructor notes.
- Connect n8n workflows for notifications.
