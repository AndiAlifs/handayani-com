# NYAMPE Attendance → handayani.com Super App — Design

**Date:** 2026-05-31
**Status:** Approved for planning
**Branch:** `worktree-nyampe-integration`

## 1. Goal

Fold the **NYAMPE** field-attendance system (project `C:\projects\absence_alip`) into
**handayani.com**, turning handayani into a super app: a public driving-school
marketing site plus an authenticated staff dashboard that now includes
GPS attendance, leave, manager oversight, office management, and the
NYAMPE student/instructor/learning-plan features.

NYAMPE's *logic* is reused as-is (Go/Gin backend), but it is fronted by handayani's
FastAPI as a single gateway and points at **one** shared MySQL database (`handayani`).

## 2. Source systems (as-is)

| | handayani.com | NYAMPE (absence_alip) |
|---|---|---|
| Frontend | Angular 18, standalone components, ngx-translate | Angular 16, NgModule |
| Backend | Python / FastAPI (PyMySQL) | Go 1.24 / Gin + GORM |
| DB | MySQL `handayani` (courses, instructors, schedules, mechanisms) | MySQL `attendance_db` (users, attendance, leave, offices, students, …) |
| Auth | **Mocked** (`AuthService`, hardcoded `admin`/`instruktur`) | Real JWT + RBAC (employee/manager/instructor, `is_super_admin`) |
| Port | FastAPI on 8080 (frontend expects 8080) | Go on 8080 |

handayani already has the super-app skeleton: a `/dashboard` shell with a
role-filtered sidebar (`dashboard-layout`), an `authGuard`, a `/login` page, and
`AuthService`/`ApiService`. Login is currently fake. This is the integration target.

## 3. Decisions (locked with the user)

1. **Full NYAMPE in scope**, including its student / learning-plan / instructor-teaching modules.
2. **Replace, don't exclude.** Where NYAMPE overlaps handayani, NYAMPE wins:
   - **Keep** (handayani-unique): public landing page, `kursus` (course pricing), `mekanisme` (SIM mechanisms).
   - **Replace** with NYAMPE equivalents: `instruktur`, `crm` (student CRM), `sesi` (training sessions).
   - **Add** (NYAMPE-new): attendance (clock-in/out), leave, manager team dashboard, approvals, office management, system settings.
3. **Backend = Go behind a FastAPI gateway.** Keep the Go service; FastAPI proxies to it. No porting of Go business logic.
4. **Frontend = port into handayani.** NYAMPE's Angular 16 components are re-created as Angular 18 standalone components inside handayani's existing `/dashboard` shell.
5. **Entry = "Login / Staff" link** on the landing-page header → `/login` → role-based redirect into `/dashboard`.
6. **One database: `handayani`.** No `attendance_db`. Go connects to `handayani` and AutoMigrates NYAMPE's tables into it.
7. **One spec, all at once** (no phased delivery).
8. **docker-compose** runs the whole stack (MySQL + Go + FastAPI gateway + Angular/nginx).

## 4. Target architecture

```
                         Browser — Angular 18 (single app, nginx :4200)
                                        │  every call → /api/*  (one origin)
                                        ▼
                        ┌───────────────────────────────────┐
                        │  FastAPI gateway   (:8080)          │
                        │  - /api/courses, /api/mechanisms    │ ── direct ──┐
                        │  - /api/health                      │             │
                        │  - /api/auth/**         ─┐          │             │
                        │  - /api/attendance/**    │ proxy    │             │
                        │  - /api/admin/**         │ (httpx)  │             │
                        │  - /api/instructor/**    │          │             │
                        │  - /api/students/**     ─┘          │             │
                        └───────────────┬─────────────────────┘             │
                                        │                                   │
                                        ▼                                   ▼
                          Go/Gin NYAMPE logic (:8090) ──► MySQL `handayani` ◄──
                                   AutoMigrate + seed       (ONE db, ONE container)
```

- The browser talks **only** to the FastAPI gateway. The Go service is internal.
- The Go service moves **8080 → 8090** so FastAPI keeps 8080 (the port the frontend already expects). Browser→Go CORS becomes unnecessary; the only CORS surface is FastAPI (already `allow_origins=["*"]`).

### 4.1 Gateway proxy

A new FastAPI router (`backend/app/routers/gateway.py`) forwards a fixed set of path
prefixes to the Go service using `httpx.AsyncClient`.

- Prefix → target rewrite:
  - `POST /api/auth/login`     → Go `POST /api/login`
  - `POST /api/auth/register`  → Go `POST /api/register`
  - `/api/attendance/{rest}`   → Go `/api/{rest}`   (e.g. `/api/attendance/clock-in` → `/api/clock-in`)
  - `/api/admin/{rest}`        → Go `/api/admin/{rest}`   (includes student/learning-plan/office/settings admin endpoints)
  - `/api/instructor/{rest}`   → Go `/api/instructor/{rest}`
- Forward verbatim: HTTP method, `Authorization` header, query string, and request body. Return the Go response status, body, and content-type unchanged.
- Catch-all by prefix using `{path:path}` so new Go endpoints need no gateway change.
- Target base URL from env `GO_BACKEND_URL` (default `http://localhost:8090`).
- New dependency: `httpx` added to `backend/requirements.txt`.

**Why a prefix namespace** (`/api/attendance/...`) rather than passing Go's raw
`/api/clock-in`: it keeps handayani's own `/api/courses|mechanisms` cleanly separated
from the proxied surface and avoids any path collision (both apps define `/api/login`,
`/api/instructors`, etc.).

### 4.2 Database (single `handayani` db)

- **Go** `DB_NAME=handayani` (env), connects to the shared MySQL, runs `AutoMigrate`
  for all NYAMPE models (`User`, `Attendance`, `LeaveRequest`, `OfficeLocation`,
  `ManagerOffice`, `SystemSettings`, `Student`, `StudentSession`, `LearningPlan`,
  and its own `Instructor`-shaped tables) plus `seed.RunAll()` on startup.
- **FastAPI** keeps reading/writing `courses` and `mechanisms` in the same db.
- **`backend/schema.sql`**: keep `courses` + `mechanisms`; **remove `instructors` and
  `schedules`** — NYAMPE's Go models now own the `instructors`/student/session tables.
- **`backend/seed.sql`**: drop instructor/schedule seed rows; keep course/mechanism seeds.
- **Migration ordering**: MySQL container init creates the `handayani` db and loads
  `schema.sql`/`seed.sql` (courses + mechanisms only). Go AutoMigrate then adds all
  NYAMPE tables on first startup. Go waits for MySQL via compose `depends_on` + healthcheck.
- **Collision note**: only `instructors` exists in both worlds. After removing it from
  `schema.sql`, GORM owns it. No other table names overlap.

## 5. Auth & role unification

- Replace mocked `AuthService.login()` with a real HTTP call to `POST /api/auth/login`
  (through the gateway). Store the Go-issued JWT in `localStorage` (`token`) and the
  decoded user. `isAuthenticated()` checks token presence/expiry.
- **Role model** unified to: `'employee' | 'instructor' | 'manager'` plus a
  `isSuperAdmin` boolean (super-admin = `manager` + `is_super_admin`). handayani's old
  `'admin'` role maps to `manager` (super-admin). `AuthUser` gains `isSuperAdmin` and
  `officeId`.
- An **HTTP interceptor** (`provideHttpClient(withInterceptors([authInterceptor]))`)
  attaches `Authorization: Bearer <token>` to every `/api/**` call and routes `401`
  responses to `/login`. This replaces NYAMPE's per-call `getHeaders()`.
- `authGuard` stays; add a `roleGuard` (data-driven, mirrors NYAMPE's `data: { roles: [...] }`)
  for manager/instructor-only routes.

### Seeded credentials (from Go `seed.RunAll`, unchanged)
- Super admins: `admin`/`admin`, `admin2`/`admin2`
- Employees: `karyawan1`–`karyawan5` (password = username)

## 6. Frontend — components to port

NYAMPE Angular 16 (NgModule/inline-template) → Angular 18 **standalone**, restyled to
match handayani's dashboard. All live as children of the existing `/dashboard` shell.

| NYAMPE component | New handayani route | Roles | Notes |
|---|---|---|---|
| `clock-in` | `/dashboard/absensi` | employee, instructor | Browser geolocation ports as-is |
| `my-attendance-history` | `/dashboard/riwayat-absensi` | employee, instructor | |
| `leave-request` | `/dashboard/cuti` | employee, instructor | |
| `leave-history` | `/dashboard/riwayat-cuti` | employee, instructor | |
| `manager-dashboard` | `/dashboard/kehadiran-tim` | manager | daily team dashboard; large inline template |
| pending-clockins (part of dashboard) | `/dashboard/persetujuan` | manager | approve/reject off-site |
| `attendance-reports` | `/dashboard/laporan` | manager | all records |
| `leave-management` | `/dashboard/manajemen-cuti` | manager | approve/reject leave |
| employee CRUD (admin) | `/dashboard/karyawan` | manager | replaces nothing; new |
| `office-management` | `/dashboard/kantor` | manager | multi-office, radius, target time |
| settings | `/dashboard/pengaturan` | super-admin | session duration, min work hours, quota presets |
| `admin-students` | `/dashboard/crm` | manager, instructor | **replaces** handayani `crm` |
| `admin-learning-plans` | `/dashboard/sesi` | manager, instructor | **replaces** handayani `sesi` |
| `admin-instructors` | `/dashboard/instruktur` | manager | **replaces** handayani `instruktur` |
| instructor module (dashboard/learning-plan/student-mgmt) | `/dashboard/instruktur-saya/*` | instructor | NYAMPE instructor self-service |

**Kept** handayani screens: `/dashboard/kursus`, `/dashboard/mekanisme`, `/dashboard` (overview).
**Removed** handayani files: old `instruktur`, `crm`, `sesi` components and their `core/models`/mock data; FastAPI `instructors` router.

- A single `AttendanceService` (Angular) wraps the gateway endpoints (mirrors NYAMPE's
  `api.service.ts` but pointed at `/api/attendance|admin|instructor` and using the interceptor).
- Sidebar `navItems` in `dashboard-layout` extended with the new entries and role filters.
- Landing page: the **instructor-schedule** section now reads NYAMPE instructor/schedule
  data via the gateway (public-readable endpoint) instead of handayani's old tables.
- Add **"Login / Staff"** link to the landing header (`shared/components/navbar` or hero).

## 7. Routing map (frontend → gateway → Go)

| Frontend call | Gateway path | Go path |
|---|---|---|
| login | `/api/auth/login` | `/api/login` |
| clock-in / clock-out | `/api/attendance/clock-in` `/clock-out` | `/api/clock-in` `/clock-out` |
| my attendance/leave | `/api/attendance/my-attendance/*`, `/my-leave/*` | `/api/my-attendance/*`, `/my-leave/*` |
| office location (read) | `/api/attendance/office-location` | `/api/office-location` |
| manager records/leaves/approvals/offices/settings | `/api/admin/*` | `/api/admin/*` |
| instructor self-service | `/api/instructor/*` | `/api/instructor/*` |
| courses, mechanisms, health | `/api/courses`, `/api/mechanisms`, `/api/health` | *(FastAPI direct)* |

## 8. docker-compose

`docker-compose.yml` at repo root, four services:

```
services:
  mysql:        # mysql:8 ; env MYSQL_DATABASE=handayani ; volume db_data
                # ./backend/schema.sql + seed.sql mounted into /docker-entrypoint-initdb.d
                # healthcheck: mysqladmin ping
  go-backend:   # build ./ (NYAMPE Go copied in) ; port 8090 internal
                # env DB_HOST=mysql DB_NAME=handayani DB_USER/PASSWORD, JWT_SECRET
                # depends_on mysql (service_healthy)
  gateway:      # build ./backend (FastAPI) ; port 8080:8080
                # env GO_BACKEND_URL=http://go-backend:8090 + handayani DB creds
                # depends_on mysql (healthy), go-backend
  frontend:     # build ./frontend (ng build) → nginx ; port 4200:80
                # nginx proxies /api → gateway:8080
volumes: { db_data: {} }
```

- The NYAMPE Go source is vendored into the handayani repo under `attendance-backend/`
  (new top-level dir) with its own Dockerfile; compose builds it.
- Each service gets a Dockerfile: `backend/Dockerfile` (python:3.12-slim + uvicorn),
  `attendance-backend/Dockerfile` (golang build → distroless/alpine), `frontend/Dockerfile`
  (node build → nginx:alpine with an `nginx.conf` that proxies `/api`).
- `.env.example` documents `MYSQL_ROOT_PASSWORD`, `JWT_SECRET`, DB creds.
- `docker compose up --build` brings the whole super app online; frontend at
  `http://localhost:4200`, all `/api` traffic funnels through the gateway.

## 9. Error handling

- **Gateway**: on Go connection failure return `502 {"error": "Layanan absensi tidak tersedia"}`;
  pass through Go's 4xx/5xx bodies unchanged. Timeout via httpx (e.g. 15s).
- **Geolocation**: keep NYAMPE's denied/timeout handling; show Indonesian guidance
  ("Izinkan akses lokasi…"). HTTPS required in prod for geolocation.
- **Auth**: interceptor redirects 401 → `/login`; expired JWT (24h, no refresh) forces re-login.
- **Frontend offline**: handayani's existing courses/mechanisms mock-fallback is kept for
  those two read paths only; attendance calls do **not** silently fall back (would mask
  real attendance state) — they surface errors.

## 10. Testing strategy

- **Gateway (pytest + httpx mock / respx)**: prefix rewriting (`/api/attendance/clock-in`
  → `/api/clock-in`), Authorization + body forwarding, status pass-through, 502 on Go down.
- **FastAPI direct routers**: existing courses/mechanisms tests stay green after removing
  the instructors router.
- **Go**: existing NYAMPE handler/distance tests run unchanged against `handayani` db
  (CI uses a throwaway MySQL).
- **Frontend (Karma/Jasmine)**: `AuthService` real-login + interceptor token attach;
  `authGuard`/`roleGuard`; a smoke test per ported component (renders, calls service).
- **End-to-end (manual, documented in spec acceptance)**: `docker compose up`, log in as
  `karyawan1` → clock-in shows status; log in as `admin` → team dashboard + approve a
  pending clock-in + manage an office.

## 11. Acceptance criteria

1. `docker compose up --build` starts MySQL, Go (8090), gateway (8080), frontend (4200) with one `handayani` db.
2. Landing page reachable at `/`; "Login / Staff" link → `/login`.
3. Login with seeded `karyawan1`/`karyawan1` (employee) and `admin`/`admin` (super-admin) works through the gateway and stores a real JWT.
4. Employee can clock-in (GPS), clock-out, view attendance + leave history, submit leave.
5. Manager can view team daily dashboard, approve/reject pending clock-ins, approve/reject leave, CRUD employees, manage offices, change settings.
6. Instructor/student/learning-plan screens (`instruktur`, `crm`, `sesi`) are served by NYAMPE logic; handayani's old versions are gone.
7. `kursus`, `mekanisme`, and the public landing page still work, served by FastAPI directly.
8. All automated tests pass.

## 12. Out of scope

- Refactoring NYAMPE's Go internals or unifying it into Python.
- A token-refresh mechanism (NYAMPE has none; 24h expiry retained).
- Production hardening (TLS, secrets manager) beyond `.env` + compose.
- Merging the two `landing-page` concepts — handayani's landing stays; NYAMPE's is dropped.
```

