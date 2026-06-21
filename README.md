# YPA Handayani — Knowledge Base & Operations Platform

A web platform for **YPA Handayani**, a vocational training provider (LPK) best known for its
driving school (*Kursus Mengemudi Handayani*) and other courses (sewing, computer, English,
Mandarin). It combines:

- a **public landing page** where prospective students browse courses & pricing, view instructor
  availability, read the SIM (driver's licence) procedure, and chat with an FAQ bot, and
- an **authenticated dashboard** where managers and instructors run the course catalog, CRM,
  training sessions (with AI note analysis), and field attendance.

The system is built around the PRD in [`docs/archived/prd.md`](docs/archived/prd.md) (Epics 2–5) and the work
breakdown in [`docs/archived/wbs.md`](docs/archived/wbs.md). Many endpoints and components reference these epics in
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
| API gateway + Attendance/Auth ("NYAMPE") | Go 1.24 · Gin · **GORM** (auto-migrate) · `golang-jwt` — the public front door on `:8080` |
| AI service | Python 3.10+/3.12 · FastAPI · raw **PyMySQL** (no ORM) · `httpx` — internal on `:8081`, behind the Go gateway |
| Database | MySQL 8 — a single `handayani` database shared by both backends |
| Auth | HS256 JWT minted by the Go service; the AI service validates the forwarded token on `/analyze` with a shared `JWT_SECRET` |
| AI | Google **Gemini 2.5** via the `google-genai` SDK (with a deterministic offline stub) |
| Orchestration | Docker Compose (`compose.yml`) — 4 services on one network |

---

## Architecture at a glance

```
                              ┌───────────────────────────────────────────────┐
   Browser                    │              NYAMPE Go gateway (core)           │
 ┌──────────┐                 │              http://localhost:8080              │
 │ Angular  │  HTTPS / JSON   │                                                 │
 │  SPA     │ ───────────────►│  /api/courses      (Epic 2, CRUD)               │
 │ :4200    │   (Bearer JWT)  │  /api/mechanisms   (Epic 4, CRUD)               │
 └──────────┘                 │  /api/crm/students (manager-only CRUD)          │
   ▲                          │  /api/sessions     (CRUD)                       │
   │ token in localStorage    │  /api/login /register /clock-in /admin/* ...    │
   │ (auth interceptor)       │  /api/auth|attendance/*  → native /api/* alias  │
   │                          │                                                 │
   │                          │  /api/sessions/{id}/analyze   ──┐                │
   │   login response         │  /api/rag/knowledge-sync[.json] ┤ reverse-proxy  │
   └──────────────────────────└─────────────────────────────────┼───────────────┘
       { token, role, ... }                                      │
                                                                 ▼
                              ┌───────────────────────────────────────────────┐
                              │             FastAPI AI service (ai)             │
                              │          http://localhost:8081 (internal)       │
                              │  /api/sessions/{id}/analyze   → Gemini 2.5      │
                              │  /api/rag/knowledge-sync      → text/markdown   │
                              └───────────────────────────────────────────────┘
                                            │                     │
                                            ▼                     ▼
                              ┌───────────────────────────────────────────────┐
                              │                   MySQL 8 (db) :3306           │
                              │  Content tables: courses, mechanisms,           │
                              │     students_crm, sessions   (schema.sql/seed)  │
                              │  Go tables (GORM auto-migrate): users,          │
                              │     attendances, leave_requests, offices,       │
                              │     students, student_sessions, learning_plans  │
                              └───────────────────────────────────────────────┘
```

(RAG also calls back into the Go gateway server-side, with a manager service account, to fetch the
instructor list for the knowledge base.)

Key design decisions:

- **The browser only ever talks to the Go gateway** (`:8080`). It serves auth, attendance, and all
  content CRUD natively, and reverse-proxies the two AI endpoints to the internal FastAPI service
  (`:8081`). The SPA's `/api/auth/*` and `/api/attendance/*` calls are rewritten to the Go service's
  native `/api/*` routes (`/api/auth/login` → `/api/login`, `/api/attendance/clock-in` → `/api/clock-in`);
  `/api/admin/*` and `/api/instructor/*` are served at those exact paths.
- **One JWT, two services.** The Go service signs an HS256 token and validates it on its protected
  routes; the AI service validates the *same* token (shared `JWT_SECRET`) on `/analyze`, and the Go
  service validates the service token RAG mints to fetch instructors. Neither stores sessions server-side.
- **One shared database.** Both backends use the `handayani` schema. The content tables are loaded
  from `backend/schema.sql` + `backend/seed.sql`; the Go service auto-migrates and seeds its own. The
  AI service reads courses/mechanisms and reads/writes the `sessions.ai_*` columns directly.
- **JSON is camelCase on both sides.** The Go gateway's content structs (`models/knowledge.go`) and
  FastAPI's Pydantic models (`backend/app/models.py`) deliberately use camelCase (e.g. `programType`,
  `progressScore`) so payloads round-trip to the Angular TypeScript models with no transformation.
  **When you add a field, change both sides.**
- **Graceful degradation is load-bearing.** The Angular `ApiService` wraps every read in `catchError`
  and falls back to bundled mock data (`core/services/mock-data.ts`); writes echo their payload back
  (assigning a client-side `Date.now()` id on create). The dashboard stays usable with the backend
  down — keep this pattern when adding endpoints.

---

## Repository layout

```
.
├── compose.yml              # one-shot Docker stack (db + core + ai + web)
├── DOCKER.md                # Docker usage notes
├── CLAUDE.md                # contributor/agent guidance for this repo
├── docs/
│   ├── planned/             # future work — specs & roadmaps
│   │   ├── ai-improvements-plan.md
│   │   └── payroll-module-design.md
│   ├── archived/            # completed / historical docs
│   │   ├── prd.md           # product requirements (Epics 2–5)
│   │   └── wbs.md           # work breakdown structure
│   └── reference/           # active operational docs
│       └── seed-credentials.md
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
├── backend/                 # FastAPI internal AI service (analyze + RAG)
│   ├── app/
│   │   ├── main.py           # app + router registration + /api/health (no CORS — internal)
│   │   ├── database.py       # PyMySQL connection (get_db dependency)
│   │   ├── models.py         # Pydantic schemas for /analyze (camelCase)
│   │   ├── auth.py           # validates the Go-issued JWT (forwarded on /analyze)
│   │   ├── ai.py             # Gemini session analysis (+ stub)
│   │   └── routers/          # rag, sessions (analyze-only)
│   ├── schema.sql · seed.sql # content tables, loaded by MySQL initdb (shared DB)
│   ├── requirements.txt · Dockerfile
│   └── tests/
│
└── core/                    # "NYAMPE" Go service — the API gateway (auth + attendance + content CRUD)
    ├── main.go               # routes, CORS, JWT middleware groups, AI reverse-proxy, prefix aliases
    ├── auth/                 # JWT issue + verify, role middleware
    ├── handlers/             # auth, attendance, admin, instructor, settings,
    │                         #   courses, mechanisms, crm, sessions, aiproxy
    ├── models/               # GORM models (User, Attendance, ...; knowledge.go: Course/Mechanism/...)
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
| `core` | http://localhost:8080 | NYAMPE Go backend — public API gateway (auth + attendance + content CRUD) |
| `ai` | (internal `:8081`) | FastAPI AI service — `/analyze` + RAG; not published by default |
| `db` | localhost:3306 | MySQL 8, single `handayani` database |

Then open **http://localhost:4200** and log in with a seeded account
(see [seeded accounts](#authentication-roles--seeded-accounts), e.g. `admin` / `admin`).

The content tables are initialised from `backend/schema.sql` + `backend/seed.sql` **only on the first
boot** (i.e. on an empty MySQL volume); the Go service auto-migrates and seeds its own tables on every
start.

Common commands:

```bash
docker compose up --build -d     # run detached
docker compose logs -f core      # tail one service (gateway); or `ai`, `db`, `web`
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
mysql -u root -p < backend/schema.sql   # creates the `handayani` DB + content tables
mysql -u root -p < backend/seed.sql     # seeds courses/mechanisms/CRM/sessions
```
The Go service creates and seeds its own tables automatically on first run. The content tables must
exist before the Go gateway serves traffic, so load these first.

### 2. Go gateway — `cd core`

```bash
cp .env.example .env              # set DB_*, JWT_SECRET, AI_SERVICE_URL; PORT=8080
go run .                          # or: go build -o server . && ./server
```
**Port 8080 is required** — the Angular `environment.ts` points there. `AI_SERVICE_URL` (default
`http://localhost:8081`) is where the gateway proxies the analyze + RAG endpoints.

### 3. FastAPI AI service — `cd backend`

```bash
python -m venv .venv
.venv\Scripts\activate          # Windows PowerShell
# source .venv/bin/activate       # macOS / Linux
pip install -r requirements.txt
copy .env.example .env            # set DB_*, JWT_SECRET, GO_BACKEND_URL=http://localhost:8080
uvicorn app.main:app --reload --port 8081
```
Runs on **8081**, behind the Go gateway. The Go and Python services **must share the same
`JWT_SECRET`** (both default to `super-secret-key-default` in dev). Swagger UI:
http://localhost:8081/docs · health: `GET /api/health`.

### 4. Frontend — `cd frontend`

```bash
npm install
npm start                         # ng serve → http://localhost:4200
npm run build                     # production build to frontend/dist/
npm test                          # Karma + Jasmine unit tests (Chrome)
```

---

## Configuration & environment variables

### FastAPI AI service (`backend/.env`)

| Variable | Default | Purpose |
|----------|---------|---------|
| `DB_HOST` / `DB_PORT` | `127.0.0.1` / `3306` | MySQL host/port |
| `DB_USER` / `DB_PASSWORD` / `DB_NAME` | `root` / *(empty)* / `handayani` | MySQL credentials & database |
| `JWT_SECRET` | `super-secret-key-default` | **Must equal the Go service's secret** to validate the forwarded token |
| `GEMINI_API_KEY` | *(empty)* | Enables real Gemini analysis; blank → deterministic stub |
| `GEMINI_MODEL` | `gemini-2.5-flash` | Gemini model id |
| `GO_BACKEND_URL` | `http://localhost:8080` | Go gateway address; RAG fetches the instructor list from it |
| `RAG_SERVICE_USERNAME` / `RAG_SERVICE_PASSWORD` | *(empty)* | Manager service account for the RAG instructor fetch; blank → omit that section |

Run uvicorn on `--port 8081` (the Go gateway is the front door on 8080).

### Go gateway (`core/.env`)

| Variable | Default | Purpose |
|----------|---------|---------|
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` | local MySQL | DB connection (point at the same `handayani` DB) |
| `JWT_SECRET` | `super-secret-key-default` | Signs the JWT (share with the AI service) |
| `PORT` | `8080` | Listen port (the Angular `environment.ts` points here) |
| `AI_SERVICE_URL` | `http://localhost:8081` | Where the gateway reverse-proxies the analyze + RAG endpoints |
| `GIN_MODE` | `debug` | `release` in production |

### Frontend

`frontend/src/environments/environment.ts` sets `apiBaseUrl: http://localhost:8080`. This is baked
into the production build; to host the API elsewhere, change it and rebuild the `web` image.

---

## Authentication, roles & seeded accounts

Authentication is **real** (JWT), not a mock. Flow:

1. The SPA `POST`s credentials to `/api/auth/login`; the Go gateway serves it via its `/api/auth/*`
   alias (rewritten to the native `/api/login`).
2. The Go service verifies the bcrypt password hash and returns
   `{ token, role, full_name, username }`. With `remember_me: true` the token lasts 7 days, otherwise
   it uses the `session_duration_hours` system setting (default 24h).
3. The SPA stores the token in `localStorage` and attaches `Authorization: Bearer <token>` to every
   request via an HTTP interceptor. A `401` clears the token and redirects to `/login`.
4. The Go gateway validates the token (`auth/jwt.go`) on its protected routes; the AI service
   re-validates the forwarded token (`backend/app/auth.py`) on `/api/sessions/{id}/analyze`.

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

### Served natively by the Go gateway

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
| `POST` / `GET` | `/api/login` `/api/register`, `/api/auth/*` alias | public | Auth (issues the JWT) |
| various | `/api/attendance/*` (alias), `/api/admin/*`, `/api/instructor/*` | role-gated | Attendance, manager & instructor tooling; `.xlsx` roster export |

\* Course and mechanism writes are currently unauthenticated at the API layer; the dashboard gates
them behind the manager role in the UI.

### Reverse-proxied to the FastAPI AI service

The gateway forwards these unchanged to the internal AI service (`AI_SERVICE_URL`), preserving the
`Authorization` header and request body. If that service is unreachable it returns `502` with
`{"error": "AI service unavailable"}`.

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| `POST` | `/api/sessions/{id}/analyze` | any auth | Run Gemini analysis on instructor notes; persists `ai_*` |
| `GET` | `/api/rag/knowledge-sync[.json]` | public | Whole KB flattened to `text/markdown` (or JSON chunks) for the bot |

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

**Content tables** (`backend/schema.sql`, shared — served by the Go gateway, also read by the AI
service for RAG and `/analyze`):

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

> The instructor weekly schedule (PRD Epic 3) is not currently served by either backend — the
> Angular instructor views fall back to bundled mock data (see *Project status & known gaps*).

---

## Testing

- **Frontend:** `cd frontend && npm test` — Karma + Jasmine, specs live next to source as `*.spec.ts`.
  Run a single spec with `ng test --include='**/api.service.spec.ts'`.
- **FastAPI AI service:** no automated test suite at present (the former gateway proxy tests were
  removed with the gateway).
- **Go:** no automated test suite at present.

---

## Project status & known gaps

This repo is under active development; a few things are mid-migration and worth knowing:

- **Instructor schedule endpoints (PRD Epic 3) are not served by either backend.** The Angular
  `ApiService` still calls `/api/instructors/schedule[/public]`, but neither the Go gateway nor the
  AI service serves them, so the instructor views fall back to bundled mock data via graceful
  degradation. (RAG fetches its instructor data from the Go gateway's `/api/admin/instructors`, which
  is unrelated to this missing schedule-matrix feature.)
- **Course & mechanism writes are unauthenticated** at the API layer (preserved from the prior
  FastAPI behaviour); the dashboard gates them behind the manager role in the UI. Consider gating the
  write routes behind `ManagerMiddleware` (GET stays public) before any real deployment.
- **Attendance/leave UI is partially wired.** Components exist under `frontend/src/app/dashboard/`
  (`absensi`, `cuti`, `riwayat-absensi`, `riwayat-cuti`, `kehadiran-tim`) and an `attendance.service`
  consumes the Go endpoints, but they are not yet added to `app.routes.ts`/the sidebar.
- **Dev secrets are committed for convenience** (default `JWT_SECRET`, seeded passwords). Lock these
  down before any real deployment.

---

## Further documentation

- [`docs/archived/prd.md`](docs/archived/prd.md) — product requirements (vision, personas, Epics 2–5).
- [`docs/archived/wbs.md`](docs/archived/wbs.md) — work breakdown structure.
- [`docs/planned/ai-improvements-plan.md`](docs/planned/ai-improvements-plan.md) — AI/RAG improvements roadmap.
- [`docs/planned/payroll-module-design.md`](docs/planned/payroll-module-design.md) — payroll module design spec.
- [`docs/reference/seed-credentials.md`](docs/reference/seed-credentials.md) — seeded accounts & data reference.
- [`DOCKER.md`](DOCKER.md) — running the full stack with Docker.
- [`CLAUDE.md`](CLAUDE.md) — contributor/agent guidance and architectural invariants.
</content>
