# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Knowledge Base web app for YPA Handayani (a driving/skills course provider). A public Angular landing page plus an authenticated admin dashboard, backed by a FastAPI + MySQL REST API. The work is organized around PRD Epics 2–5 (see `docs/prd.md`, `docs/wbs.md`); endpoints and components reference these epics in comments.

The repo is split into two independent apps:

- `frontend/` — Angular 18 standalone-component SPA
- `backend/` — Python 3.10+ / FastAPI REST API over MySQL

## Commands

### Frontend (`cd frontend`)
- `npm install` — install deps
- `npm start` / `ng serve` — dev server at http://localhost:4200
- `npm run build` / `ng build` — production build to `frontend/dist/`
- `npm test` / `ng test` — Karma + Jasmine unit tests (Chrome launcher)
- Single spec: `ng test --include='**/api.service.spec.ts'` (specs are `*.spec.ts` next to source)

### Backend (`cd backend`)
- Setup: `mysql -u root -p < schema.sql && mysql -u root -p < seed.sql`, then `python -m venv .venv`, activate, `pip install -r requirements.txt`
- Config: `copy .env.example .env` (DB_HOST/PORT/USER/PASSWORD/NAME; defaults = local root / no password / `handayani`)
- Run: `uvicorn app.main:app --reload --port 8080` — **port 8080 is required**, the Angular `environment.ts` points there
- Docs: http://localhost:8080/docs (Swagger UI). Health: `GET /api/health`
- No test suite exists for the backend.

## Architecture

### Frontend ↔ backend contract
- `ApiService` (`frontend/src/app/core/services/api.service.ts`) is the single HTTP boundary. Base URL comes from `environment.ts` (`apiBaseUrl: http://localhost:8080`).
- **Graceful degradation is intentional and load-bearing:** every read falls back via `catchError` to bundled mock data in `core/services/mock-data.ts`, and every write echoes its payload back (assigning a client-side `Date.now()` id on create). The dashboard stays fully functional with the backend down — keep this pattern when adding endpoints.
- **JSON is camelCase on both sides.** Backend Pydantic models in `backend/app/models.py` deliberately use camelCase field names (e.g. `timeSlot`) so the wire format matches the Angular TypeScript models in `frontend/src/app/core/models/`. When adding fields, change both.

### Frontend structure
- Bootstrapped via `app.config.ts` (no NgModules — standalone components, `provideRouter`, `provideHttpClient`, ngx-translate).
- Routes (`app.routes.ts`): `/` landing page (eager), everything else lazy-loaded. `/dashboard/*` is guarded by `authGuard` and uses `DashboardLayoutComponent` as a shell with child routes (overview, kursus, instruktur, mekanisme, crm, sesi). Dashboard route paths are in Indonesian.
- **Auth is mock/demo only** (`core/services/auth.service.ts`): hard-coded credentials (`admin`/`admin123`, `instruktur`/`instruktur123`), session persisted to `localStorage`, no real token. Roles: `admin` | `instructor`.
- i18n via `@ngx-translate` v17. Default language `id` (Indonesian); also `en`. Translation files: `frontend/public/i18n/{id,en}.json`. Much UI text and domain vocabulary is Indonesian.

### Backend structure
- `app/main.py` — FastAPI app, wide-open CORS (`*`), registers routers.
- `app/database.py` — **raw PyMySQL**, not an ORM (the README's "SQLAlchemy" note is inaccurate; only `pymysql` is in `requirements.txt`). `get_db` is a FastAPI dependency yielding a connection that commits on success / rolls back on exception. Routers write raw SQL with `DictCursor`.
- `app/routers/` — one router per epic: `courses.py` (Epic 2), `instructors.py` (Epic 3), `mechanisms.py` (Epic 4), `rag.py` (Epic 5). All under `/api/...`.
- **Schedule matrix** (`app/schedule.py`): the `schedules` table is stored **sparse** — only non-`Tersedia` slots are persisted. Reads reconstruct the full 7-day × 3-timeslot matrix defaulting empty cells to `Tersedia`; `PUT /api/instructors/{id}/schedule` deletes-then-reinserts the whole matrix inside the request transaction. `DAYS`/`TIME_SLOTS` here must stay in sync with the Angular schedule grid components.
- **RAG sync** (`GET /api/rag/knowledge-sync`): returns `text/markdown` (not JSON) — a single flattened KB document the chat bot polls on an interval. There are no webhooks (PRD §4.2).
