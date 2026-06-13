# YPA Handayani — Knowledge Base & Operations Platform

A web platform for **YPA Handayani**, a vocational training provider (LPK) best known for its
driving school (*Kursus Mengemudi Handayani*) and other courses (sewing, computer, English,
Mandarin). It combines:

- a **public landing page** where prospective students browse courses & pricing, view instructor
  availability, read the SIM (driver's licence) procedure, and chat with an FAQ bot, and
- an **authenticated dashboard** where managers and instructors run the course catalog, CRM,
  training sessions (with AI note analysis), and field attendance.

The system is built around the PRD in [`docs/prd.md`](docs/prd.md) (Epics 2–5) and the work
breakdown in [`docs/wbs.md`](docs/wbs.md). Many endpoints and components reference these epics in
their comments.

---

## Table of contents

- [Tech stack](#tech-stack)
- [Architecture at a glance](#architecture-at-a-glance)
- [Repository layout](#repository-layout)
- [Quick start (Docker — recommended)](#quick-start-docker--recommended)
- [Local development (without Docker)](#local-development-without-docker)
- [Configuration & environment variables](#configuration--environment-variables)
- [Authentication, roles & seeded accounts](#authentication-roles--seeded-accounts)
- [HTTP API reference](#http-api-reference)
- [AI session analysis (Gemini)](#ai-session-analysis-gemini)
- [RAG knowledge sync](#rag-knowledge-sync)
- [Frontend notes](#frontend-notes)
- [Data model](#data-model)
- [Testing](#testing)
- [Project status & known gaps](#project-status--known-gaps)
- [Further documentation](#further-documentation)

---

## Tech stack

| Layer | Technology |
|-------|-----------|
| Frontend | Angular 18 (standalone components, signals), `@ngx-translate` v17 (id/en), served by nginx in prod |
| Knowledge-Base API | Python 3.10+/3.12 · FastAPI · raw **PyMySQL** (no ORM) · `httpx` (gateway) |
| Attendance/Auth API ("NYAMPE") | Go 1.24 · Gin · **GORM** (auto-migrate) · `golang-jwt` |
| Database | MySQL 8 — a single `handayani` database shared by both backends |
| Auth | HS256 JWT minted by the Go service, validated by FastAPI with a shared `JWT_SECRET` |
| AI | Google **Gemini 2.5** via the `google-genai` SDK (with a deterministic offline stub) |
| Orchestration | Docker Compose (`compose.yml`) — 4 services on one network |

---

## Architecture at a glance

```
                              ┌───────────────────────────────────────────────┐
   Browser                    │                  FastAPI  (api)                 │
 ┌──────────┐                 │                http://localhost:8080            │
 │ Angular  │  HTTPS / JSON   │                                                 │
 │  SPA     │ ───────────────►│  /api/courses      (Epic 2, CRUD)               │
 │ :4200    │   (Bearer JWT)  │  /api/mechanisms   (Epic 4, CRUD)               │
 └──────────┘                 │  /api/crm/students (manager-only CRUD)          │
   ▲                          │  /api/sessions     (CRUD + /analyze → Gemini)   │
   │ token in localStorage    │  /api/rag/knowledge-sync (text/markdown)        │
   │ (auth interceptor)       │                                                 │
   │                          │  /api/auth|attendance|admin|instructor  ──┐     │
   │                          └───────────────────────────────────────────┼─────┘
   │                                                                       │ reverse-proxy
   │                                                                       ▼ (httpx)
   │                          ┌───────────────────────────────────────────────┐
   │                          │            NYAMPE Go backend (attendance)       │
   │   login response         │              http://localhost:8090             │
   └──────────────────────────│  /api/login /api/register   → issues the JWT    │
       { token, role, ... }   │  /api/clock-in /clock-out /leave /my-*          │
                              │  /api/admin/*   (manager)                       │
                              │  /api/instructor/*  (instructor)                │
                              └───────────────────────────────────────────────┘
                                            │                     │
                                            ▼                     ▼
                              ┌───────────────────────────────────────────────┐
                              │                   MySQL 8 (db) :3306           │
                              │  FastAPI tables: courses, mechanisms,           │
                              │     students_crm, sessions   (schema.sql/seed)  │
                              │  Go tables (GORM auto-migrate): users,          │
                              │     attendances, leave_requests, offices,       │
                              │     students, student_sessions, learning_plans  │
                              └───────────────────────────────────────────────┘
```

Key design decisions:

- **The browser only ever talks to FastAPI** (`:8080`). Auth and attendance live in the Go service,
  but the SPA reaches them through FastAPI's gateway router (`backend/app/routers/gateway.py`),
  which forwards `/api/auth|attendance|admin|instructor/*` to the Go service and rewrites the prefix
  (`/api/auth/login` → `/api/login`, `/api/attendance/clock-in` → `/api/clock-in`).
- **One JWT, two services.** The Go service signs an HS256 token; FastAPI validates it with the
  *same* `JWT_SECRET`. Neither stores sessions server-side.
- **One database, two owners.** Both backends use the `handayani` schema. FastAPI's tables are loaded
  from `backend/schema.sql` + `backend/seed.sql`; the Go service auto-migrates and seeds its own.
- **JSON is camelCase on both sides.** FastAPI's Pydantic models in `backend/app/models.py`
  deliberately use camelCase (e.g. `timeSlot`, `progressScore`) so payloads round-trip to the Angular
  TypeScript models with no transformation. **When you add a field, change both sides.**
- **Graceful degradation is load-bearing.** The Angular `ApiService` wraps every read in `catchError`
  and falls back to bundled mock data (`core/services/mock-data.ts`); writes echo their payload back
  (assigning a client-side `Date.now()` id on create). The dashboard stays usable with the backend
  down — keep this pattern when adding endpoints.

---

## Repository layout

```
.
├── compose.yml              # one-shot Docker stack (db + attendance + api + web)
├── DOCKER.md                # Docker usage notes
├── CLAUDE.md                # contributor/agent guidance for this repo
├── docs/
│   ├── prd.md               # product requirements (Epics 2–5)
│   ├── wbs.md               # work breakdown structure
│   └── superpowers/         # design specs & implementation plans
│
├── frontend/                # Angular 18 SPA
│   ├── src/app/
│   │   ├── landing-page/     # public site: hero, pricing, schedule, SIM guide, chat-bot, ...
│   │   ├── dashboard/        # authed: overview, kursus, instruktur, mekanisme, crm, sesi, absensi...
│   │   ├── auth/login/       # login screen
│   │   ├── core/             # services (api, auth, attendance, theme), guards, interceptors, models
│   │   └── shared/           # navbar, shared components
│   ├── public/i18n/{id,en}.json
│   ├── Dockerfile · nginx.conf
│
├── backend/                 # FastAPI Knowledge-Base API + gateway
│   ├── app/
│   │   ├── main.py           # app, CORS, router registration, /api/health
│   │   ├── database.py       # PyMySQL connection (get_db dependency)
│   │   ├── models.py         # Pydantic schemas (camelCase)
│   │   ├── auth.py           # validates the Go-issued JWT
│   │   ├── ai.py             # Gemini session analysis (+ stub)
│   │   ├── schedule.py       # sparse weekly-matrix helpers
│   │   └── routers/          # courses, mechanisms, rag, crm, sessions, gateway
│   ├── schema.sql · seed.sql
│   ├── requirements.txt · Dockerfile
│   └── tests/                # pytest + respx (gateway proxy tests)
│
└── attendance-backend/      # "NYAMPE" Go service (auth + field attendance)
    ├── main.go               # routes, CORS, JWT middleware groups
    ├── auth/                 # JWT issue + verify, role middleware
    ├── handlers/             # auth, attendance, leave, office, admin, instructor, settings
    ├── models/               # GORM models (User, Attendance, LeaveRequest, Office, ...)
    ├── seed/                 # demo users, offices, students, attendance history
    └── Dockerfile
```

---

## Quick start (Docker — recommended)

Requires Docker with Compose v2. From the repo root:

```bash
docker compose up --build
```

This brings up four services on one network:

| Service | URL / port | Description |
|---------|-----------|-------------|
| `web` | http://localhost:4200 | Angular SPA (production build via nginx) |
| `api` | http://localhost:8080 | FastAPI — KB/CRM/Sessions/RAG + gateway to Go; Swagger at `/docs` |
| `attendance` | http://localhost:8090 | NYAMPE Go backend (auth + attendance) |
| `db` | localhost:3306 | MySQL 8, single `handayani` database |

Then open **http://localhost:4200** and log in with a seeded account
(see [seeded accounts](#authentication-roles--seeded-accounts), e.g. `admin` / `admin`).

The FastAPI tables are initialised from `backend/schema.sql` + `backend/seed.sql` **only on the first
boot** (i.e. on an empty MySQL volume); the Go service auto-migrates and seeds its own tables on every
start.

Common commands:

```bash
docker compose up --build -d     # run detached
docker compose logs -f api       # tail one service
docker compose down              # stop (keeps the db volume)
docker compose down -v           # stop and wipe the database (re-seeds next up)
```

Optionally export `GEMINI_API_KEY`, `GEMINI_MODEL`, and/or `JWT_SECRET` before `up` to override the
defaults. See [`DOCKER.md`](DOCKER.md) for more detail.

---

## Local development (without Docker)

Run MySQL yourself, then start each app in its own terminal.

### 1. Database (MySQL 8)

```bash
mysql -u root -p < backend/schema.sql   # creates the `handayani` DB + FastAPI tables
mysql -u root -p < backend/seed.sql     # seeds courses/mechanisms/CRM/sessions
```
The Go service creates and seeds its own tables automatically on first run.

### 2. FastAPI backend — `cd backend`

```bash
python -m venv .venv
.venv\Scripts\activate          # Windows PowerShell
# source .venv/bin/activate       # macOS / Linux
pip install -r requirements.txt
copy .env.example .env            # edit if your MySQL isn't root/no-password
uvicorn app.main:app --reload --port 8080
```
**Port 8080 is required** — the Angular `environment.ts` points there. Swagger UI:
http://localhost:8080/docs · health: `GET /api/health`.

### 3. Go attendance backend — `cd attendance-backend`

```bash
cp .env.example .env              # set DB_* and (ideally) JWT_SECRET; PORT=8090
go run .                          # or: go build -o server . && ./server
```
Set `PORT=8090` so it matches the gateway's default `GO_BACKEND_URL`. The Go and Python
services **must share the same `JWT_SECRET`** (both default to `super-secret-key-default` in dev).

### 4. Frontend — `cd frontend`

```bash
npm install
npm start                         # ng serve → http://localhost:4200
npm run build                     # production build to frontend/dist/
npm test                          # Karma + Jasmine unit tests (Chrome)
```

---

## Configuration & environment variables

### FastAPI (`backend/.env`)

| Variable | Default | Purpose |
|----------|---------|---------|
| `DB_HOST` / `DB_PORT` | `127.0.0.1` / `3306` | MySQL host/port |
| `DB_USER` / `DB_PASSWORD` / `DB_NAME` | `root` / *(empty)* / `handayani` | MySQL credentials & database |
| `JWT_SECRET` | `super-secret-key-default` | **Must equal the Go service's secret** to validate tokens |
| `GEMINI_API_KEY` | *(empty)* | Enables real Gemini analysis; blank → deterministic stub |
| `GEMINI_MODEL` | `gemini-2.5-flash` | Gemini model id |
| `GO_BACKEND_URL` | `http://localhost:8090` | Where the gateway proxies auth/attendance calls |

### Go service (`attendance-backend/.env`)

| Variable | Default | Purpose |
|----------|---------|---------|
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` | local MySQL | DB connection (point at the same `handayani` DB) |
| `JWT_SECRET` | `super-secret-key-default` | Signs the JWT (share with FastAPI) |
| `PORT` | `8090` | Listen port |
| `GIN_MODE` | `debug` | `release` in production |

### Frontend

`frontend/src/environments/environment.ts` sets `apiBaseUrl: http://localhost:8080`. This is baked
into the production build; to host the API elsewhere, change it and rebuild the `web` image.

---

## Authentication, roles & seeded accounts

Authentication is **real** (JWT), not a mock. Flow:

1. The SPA `POST`s credentials to `/api/auth/login`; FastAPI's gateway forwards to the Go service's
   `/api/login`.
2. The Go service verifies the bcrypt password hash and returns
   `{ token, role, full_name, username }`. With `remember_me: true` the token lasts 7 days, otherwise
   it uses the `session_duration_hours` system setting (default 24h).
3. The SPA stores the token in `localStorage` and attaches `Authorization: Bearer <token>` to every
   request via an HTTP interceptor. A `401` clears the token and redirects to `/login`.
4. FastAPI validates the token (`backend/app/auth.py`) on protected routes.

**Roles** (`employee` | `manager` | `instructor`). `manager` is the admin-equivalent/elevated role —
there is no separate `admin` role. The legacy `isAdmin()` helper now just maps to `isManager()`.

Demo accounts created by the Go seeder on first boot (username / password):

| Username | Password | Role | Notes |
|----------|----------|------|-------|
| `admin` | `admin` | manager | super admin |
| `admin2` | `admin2` | manager | regular manager |
| `admin_kendari` | `admin_kendari` | manager | Kendari office manager |
| `instructor1` | `instructor1` | instructor | has seeded students, plans & sessions |
| `karyawan1`…`karyawan7` | *(same as username)* | employee | assigned to various offices |
| `hidayat` | `hidayat` | employee | Kampus B |

> These are development seed credentials. Change `JWT_SECRET` and the seeded passwords before any
> real deployment.

---

## HTTP API reference

Base URL: `http://localhost:8080`. JSON is camelCase. Interactive docs at `/docs`.

### Served directly by FastAPI

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| `GET` | `/api/health` | — | Liveness check |
| `GET` | `/api/courses` | public | List courses (Epic 2) |
| `POST` / `PUT` / `DELETE` | `/api/courses[/{id}]` | public* | Create / update / delete course |
| `GET` | `/api/mechanisms` | public | List SIM-procedure steps & fees (Epic 4) |
| `POST` / `PUT` / `DELETE` | `/api/mechanisms[/{id}]` | public* | Create / update / delete step |
| `GET` | `/api/crm/students` | manager | List CRM students |
| `POST` / `PUT` / `DELETE` | `/api/crm/students[/{id}]` | manager | CRM CRUD |
| `GET` | `/api/sessions` | any auth | List training sessions |
| `POST` / `PUT` / `DELETE` | `/api/sessions[/{id}]` | manager | Session CRUD |
| `POST` | `/api/sessions/{id}/analyze` | any auth | Run AI analysis on instructor notes |
| `GET` | `/api/rag/knowledge-sync` | public | Whole KB flattened to `text/markdown` for the bot |

\* Course and mechanism writes are currently unauthenticated at the API layer; the dashboard gates
them behind the manager role in the UI.

### Proxied to the Go service (gateway)

FastAPI forwards these to the NYAMPE backend, passing the `Authorization`/`Content-Type`/`Accept`
headers and the query string through. If the Go service is unreachable it returns `502` with
`{"error": "Layanan absensi tidak tersedia"}`.

| Frontend prefix | Forwarded to | Examples |
|-----------------|--------------|----------|
| `/api/auth/*` | `/api/*` | `login`, `register` |
| `/api/attendance/*` | `/api/*` | `clock-in`, `clock-out`, `leave`, `my-attendance/history`, … |
| `/api/admin/*` | `/api/admin/*` | records, leaves, employees, offices, settings, students, learning-plans, instructor insight |
| `/api/instructor/*` | `/api/instructor/*` | students, schedule, session start/end/active, quota presets |

The Go service additionally exposes `POST /api/login` and `POST /api/register` publicly, and protects
the rest with JWT + role middleware (`manager` for `/api/admin/*`, `instructor` for
`/api/instructor/*`). It can also export a student roster as `.xlsx`
(`GET /api/admin/students/roster.xlsx`).

---

## AI session analysis (Gemini)

`POST /api/sessions/{id}/analyze` takes `{ "rawNotes": "..." }` (free-text instructor notes in
Indonesian), runs them through `backend/app/ai.py`, persists the result, flips the session to
`completed`, and returns the session with a nested `aiAnalysis`:

```json
{
  "strengths": ["..."],
  "weaknesses": ["..."],
  "recommendedNextFocus": "...",
  "upsellRecommendation": "... or null"
}
```

- With `GEMINI_API_KEY` set, it calls **Gemini 2.5** (`GEMINI_MODEL`, default `gemini-2.5-flash`) with
  a JSON-only Indonesian prompt and parses the response.
- With no key — or on any SDK/network/parse error — it falls back to a **deterministic stub** so the
  feature works with zero configuration. The Angular side has its own mock fallback too, so the demo
  survives even with the API down.

`upsellRecommendation` is only populated when a student is near the end of their package but still has
significant weaknesses.

---

## RAG knowledge sync

`GET /api/rag/knowledge-sync` returns **`text/markdown`** (not JSON): the entire knowledge base —
course catalog & pricing, instructor availability, SIM procedure & fees, plus the official WhatsApp
contacts — flattened into one Indonesian document optimised for LLM retrieval. The chatbot polls this
on an interval; there are **no webhooks** (per PRD §4.2).

---

## Frontend notes

- **Bootstrapping** is NgModule-free (`app.config.ts`): `provideRouter`, `provideHttpClient` with the
  auth interceptor, and ngx-translate.
- **Routing** (`app.routes.ts`): `/` landing page (eager), everything else lazy-loaded. `/dashboard`
  is guarded by `authGuard` and uses `DashboardLayoutComponent` as a shell. Dashboard route paths are
  in Indonesian. Wired dashboard sections and the roles that see them:

  | Route | Section | Visible to |
  |-------|---------|-----------|
  | `/dashboard` | Overview | manager, instructor |
  | `/dashboard/kursus` | Kursus & Harga (courses) | manager |
  | `/dashboard/instruktur` | Instruktur & schedule | manager |
  | `/dashboard/mekanisme` | Mekanisme SIM | manager |
  | `/dashboard/crm` | CRM Siswa | manager |
  | `/dashboard/sesi` | Sesi Pelatihan (sessions + AI) | manager, instructor |

- **Landing page** components: hero, course pricing, instructor schedule (public, with booked-slot
  names masked to *"Terisi"*), SIM mechanism guide, video testimonial, Instagram feed, FAQ chat-bot,
  CTA footer, navbar.
- **i18n**: `@ngx-translate` v17, default language `id` (Indonesian), also `en`
  (`frontend/public/i18n/{id,en}.json`). Much of the UI and domain vocabulary is Indonesian.

---

## Data model

**FastAPI-owned tables** (`backend/schema.sql`):

- `courses` — category, program type, specifics, duration, price, registration fee, remarks.
- `mechanisms` — SIM requirement steps: requirement name, issuing body, cost, notes, sort order.
- `students_crm` — CRM pipeline: name, phone, course, status (`lead`/`active`/`completed`),
  progress score, notes, created date.
- `sessions` — training sessions: student/instructor/course, start & end time, status, session number
  / total, raw notes, and four `ai_*` columns surfaced as a nested `aiAnalysis` object on the wire.

**Go-owned tables** (GORM auto-migrate): `users`, `attendances`, `leave_requests`,
`office_locations`, `manager_offices`, `system_settings`, `students`, `student_sessions`,
`learning_plans`. Attendance is geofenced — clock-ins record lat/long and are validated against an
office location's allowed radius.

> The instructor weekly schedule is stored **sparse**: only non-`Tersedia` slots are persisted, and
> reads reconstruct the full 7-day × 3-timeslot matrix (`backend/app/schedule.py`). Booked slots hold
> the student's name, so the public read masks them to *"Terisi"*.

---

## Testing

- **Frontend:** `cd frontend && npm test` — Karma + Jasmine, specs live next to source as `*.spec.ts`.
  Run a single spec with `ng test --include='**/api.service.spec.ts'`.
- **FastAPI:** `cd backend && pytest` — `respx` mocks the Go service to test the gateway's
  prefix-rewriting, header/query passthrough, and the `502`-on-down behaviour (`tests/test_gateway.py`).
- **Go:** no automated test suite at present.

---

## Project status & known gaps

This repo is under active development; a few things are mid-migration and worth knowing:

- **Instructor schedule endpoints are not currently served by FastAPI.** The Angular `ApiService`
  still calls `/api/instructors/schedule[/public]`, but no `instructors` router is registered in
  `main.py` and `schema.sql` doesn't create `instructors`/`schedules` tables. Today the instructor
  views fall back to bundled mock data via graceful degradation. `rag.py` and `schedule.py` still
  reference those tables, so `/api/rag/knowledge-sync` will error against a fresh database until they
  are reintroduced or the RAG query is updated.
- **Attendance/leave UI is partially wired.** Components exist under `frontend/src/app/dashboard/`
  (`absensi`, `cuti`, `riwayat-absensi`, `riwayat-cuti`, `kehadiran-tim`) and an `attendance.service`
  consumes the Go endpoints, but they are not yet added to `app.routes.ts`/the sidebar.
- **`backend/README.md` is partly stale** — it predates the gateway/CRM/sessions work and still
  documents the removed instructor endpoints and a "SQLAlchemy" note (the backend uses raw PyMySQL).
  Treat this top-level README as the source of truth.
- **Dev secrets are committed for convenience** (default `JWT_SECRET`, seeded passwords, open CORS
  `*` on FastAPI). Lock these down before any real deployment.

---

## Further documentation

- [`docs/prd.md`](docs/prd.md) — product requirements (vision, personas, Epics 2–5).
- [`docs/wbs.md`](docs/wbs.md) — work breakdown structure.
- [`docs/superpowers/`](docs/superpowers/) — design specs & implementation plans for the NYAMPE
  attendance integration and the CRM/Sessions/AI/Auth work.
- [`DOCKER.md`](DOCKER.md) — running the full stack with Docker.
- [`CLAUDE.md`](CLAUDE.md) — contributor/agent guidance and architectural invariants.
</content>
