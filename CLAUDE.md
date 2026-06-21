# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Knowledge Base web app for YPA Handayani (a driving/skills course provider). A public Angular landing page plus an authenticated dashboard, backed by two services over a shared MySQL database. The work is organized around PRD Epics 2–5 (see `docs/prd.md`, `docs/wbs.md`); endpoints and components reference these epics in comments.

The repo is split into three apps:

- `frontend/` — Angular 18 standalone-component SPA
- `attendance-backend/` — Go (Gin + GORM) "NYAMPE" service. The **public API gateway** on port 8080: serves auth, attendance, and CRUD for courses/mechanisms/CRM/sessions, and reverse-proxies the AI endpoints to the FastAPI service.
- `backend/` — Python 3.10+ / FastAPI **internal AI service** on port 8081: Gemini session analysis + RAG knowledge-sync. Reached only through the Go gateway's reverse proxy.

## Commands

### Frontend (`cd frontend`)
- `npm install` — install deps
- `npm start` / `ng serve` — dev server at http://localhost:4200
- `npm run build` / `ng build` — production build to `frontend/dist/`
- `npm test` / `ng test` — Karma + Jasmine unit tests (Chrome launcher)
- Single spec: `ng test --include='**/api.service.spec.ts'` (specs are `*.spec.ts` next to source)

### Go gateway (`cd attendance-backend`) — the front door
- Config: `cp .env.example .env` (DB_*, `JWT_SECRET`, `PORT=8080`, `AI_SERVICE_URL=http://localhost:8081`)
- Run: `go run .` — **port 8080 is required**, the Angular `environment.ts` points there. Auto-migrates + seeds its own tables (users, attendances, students, ...).
- Build/vet: `go build ./...` · `go vet ./...`. No test suite.

### FastAPI AI service (`cd backend`) — internal, behind the gateway
- DB setup (content tables, shared DB): `mysql -u root -p < schema.sql && mysql -u root -p < seed.sql` (load before the Go gateway serves traffic)
- Then `python -m venv .venv`, activate, `pip install -r requirements.txt`, `copy .env.example .env`
- Run: `uvicorn app.main:app --reload --port 8081`. Docs: http://localhost:8081/docs. Health: `GET /api/health`
- Both services must share the same `JWT_SECRET`. No test suite.

## Architecture

### Frontend ↔ backend contract
- `ApiService` (`frontend/src/app/core/services/api.service.ts`) is the single HTTP boundary. Base URL comes from `environment.ts` (`apiBaseUrl: http://localhost:8080`) — the Go gateway. Auth/attendance use `/api/auth/*` and `/api/attendance/*` prefixes that the gateway rewrites to its native `/api/*` routes.
- **Graceful degradation is intentional and load-bearing:** every read falls back via `catchError` to bundled mock data in `core/services/mock-data.ts`, and every write echoes its payload back (assigning a client-side `Date.now()` id on create). The dashboard stays fully functional with the backend down — keep this pattern when adding endpoints.
- **JSON is camelCase everywhere.** The Go gateway's content structs (`attendance-backend/models/knowledge.go`) and the FastAPI Pydantic models (`backend/app/models.py`) use camelCase field names (e.g. `programType`) so the wire format matches the Angular TypeScript models in `frontend/src/app/core/models/`. When adding a field, change the Go struct (json + gorm tags), its handler, and the Angular model.

### Frontend structure
- Bootstrapped via `app.config.ts` (no NgModules — standalone components, `provideRouter`, `provideHttpClient`, ngx-translate).
- Routes (`app.routes.ts`): `/` landing page (eager), everything else lazy-loaded. `/dashboard/*` is guarded by `authGuard` and uses `DashboardLayoutComponent` as a shell with child routes (overview, kursus, instruktur, mekanisme, crm, sesi). Dashboard route paths are in Indonesian.
- **Auth is real JWT** (`core/services/auth.service.ts` + `core/interceptors/auth.interceptor.ts`): the SPA POSTs to `/api/auth/login`, the Go gateway issues an HS256 token, stored in `localStorage` and attached as `Authorization: Bearer` on every request; a 401 clears it and redirects to `/login`. Roles: `employee` | `manager` | `instructor` (`manager` is the admin-equivalent).
- i18n via `@ngx-translate` v17. Default language `id` (Indonesian); also `en`. Translation files: `frontend/public/i18n/{id,en}.json`. Much UI text and domain vocabulary is Indonesian.

### Go gateway structure (`attendance-backend/`)
- `main.go` — Gin engine, CORS, route groups. Public auth + `/api/health` + courses/mechanisms; `AuthMiddleware`-gated attendance, CRM (manager-only), sessions (GET any-auth, writes manager-only). Reverse-proxies `/api/sessions/{id}/analyze` and `/api/rag/knowledge-sync[.json]` to `AI_SERVICE_URL`, and aliases `/api/auth/*` + `/api/attendance/*` to native `/api/*` via `r.HandleContext`.
- `models/knowledge.go` — GORM structs for the migrated content tables (Course, Mechanism, StudentCrm, Session) with camelCase `json` + explicit `gorm:"column:..."` tags. **Deliberately NOT added to AutoMigrate** — `backend/schema.sql`/`seed.sql` own those tables' DDL (shared DB).
- `handlers/{courses,mechanisms,crm,sessions}.go` — bare-body CRUD (no `gin.H{"data":...}` envelope, unlike the attendance handlers). **The four `sessions.ai_*` columns are owned by the Python `/analyze` endpoint; Go never writes them** — create leaves them NULL, update uses a column-scoped map. On read they surface as a nested `aiAnalysis` object.
- `handlers/aiproxy.go` — `httputil.ReverseProxy` to the AI service; returns 502 when it's down.

### FastAPI AI service structure (`backend/`)
- `app/main.py` — FastAPI app (no CORS — internal, reached only via the gateway), registers the `rag` + `sessions` routers + `/api/health`.
- `app/database.py` — **raw PyMySQL**, not an ORM. `get_db` yields a connection that commits on success / rolls back on exception. Reads courses/mechanisms and reads/writes `sessions.ai_*` on the shared DB.
- `app/routers/sessions.py` — only `POST /api/sessions/{id}/analyze` (Gemini analysis via `app/ai.py`, with a deterministic offline stub).
- **RAG sync** (`app/routers/rag.py`, `GET /api/rag/knowledge-sync[.json]`): returns `text/markdown` (or JSON chunks) — a single flattened KB document the chat bot polls on an interval (no webhooks, PRD §4.2). Fetches the instructor list from the Go gateway (`GO_BACKEND_URL`) with a manager service account.
