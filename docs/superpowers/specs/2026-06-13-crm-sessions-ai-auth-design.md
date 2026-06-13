# Design: CRM + Sessions backend, Gemini AI session analysis, JWT auth

**Date:** 2026-06-13
**Status:** Approved for planning
**Scope chosen by user:** Everything except n8n automation.

## Goal

Close the three implementation gaps identified in the codebase:

1. **CRM + Sessions have no backend.** The dashboard `crm` and `sesi` screens and the
   `overview` stats run entirely off bundled mock data because `/api/crm/students`
   and `/api/sessions` do not exist on the server.
2. **AI session analysis is faked.** `SesiComponent.submitNotes()` fabricates an
   analysis with a `setTimeout`. The PRD calls for a real LLM (Gemini) to analyze
   instructor notes.
3. **Auth is demo-only.** Hard-coded credentials live in `AuthService`; there is no
   server-side user store, no password hashing, and no token.

Out of scope (explicit user decision): **n8n automation.** The analyze endpoint is
triggered by the dashboard "submit notes" action, not an n8n workflow. CRM
auto-update on analysis is left as a documented seam, not built.

## Guiding constraints (existing patterns to preserve)

- **camelCase JSON on both sides.** Pydantic models use camelCase field names so
  payloads round-trip to the Angular interfaces with no transformation. New models
  follow this.
- **Graceful degradation is load-bearing.** Every `ApiService` read falls back to
  `MOCK_*` data via `catchError`; writes echo their payload. The dashboard must stay
  functional with the backend down. New calls keep this. The LLM and login get the
  same treatment (stub analysis / mock-credential fallback) so offline demos work.
- **Raw PyMySQL, one router per concern.** `get_db` yields a connection that commits
  on success / rolls back on exception. Routers write raw SQL with `DictCursor`.
  New routers match `courses.py` / `instructors.py` exactly in shape.
- **Public reads stay unauthenticated.** The landing page and RAG bot are anonymous.

---

## Part 1 — CRM + Sessions backend

### Database (`backend/schema.sql` + `backend/seed.sql`)

New table `students_crm`:

| column         | type                                   | notes                          |
|----------------|----------------------------------------|--------------------------------|
| id             | INT AUTO_INCREMENT PK                  |                                |
| name           | VARCHAR(128) NOT NULL                  |                                |
| phone          | VARCHAR(32) NOT NULL                   |                                |
| course_id      | INT NOT NULL                           | denormalized, no FK (matches mock) |
| course_name    | VARCHAR(128) NOT NULL                  |                                |
| status         | VARCHAR(16) NOT NULL DEFAULT 'lead'    | lead / active / completed      |
| progress_score | INT NOT NULL DEFAULT 0                 | 0–100                          |
| notes          | TEXT NOT NULL                          |                                |
| created_at     | TIMESTAMP DEFAULT CURRENT_TIMESTAMP    |                                |

New table `sessions`:

| column                    | type                                | notes                       |
|---------------------------|-------------------------------------|-----------------------------|
| id                        | INT AUTO_INCREMENT PK               |                             |
| student_id                | INT NOT NULL                        |                             |
| student_name              | VARCHAR(128) NOT NULL               | denormalized                |
| instructor_id             | INT NOT NULL                        |                             |
| instructor_name           | VARCHAR(128) NOT NULL               | denormalized                |
| course_id                 | INT NOT NULL                        |                             |
| course_name               | VARCHAR(128) NOT NULL               | denormalized                |
| start_time                | DATETIME NOT NULL                   | serialized ISO 8601 on wire |
| end_time                  | DATETIME NOT NULL                   |                             |
| status                    | VARCHAR(16) NOT NULL                 | scheduled/completed/cancelled |
| session_number            | INT NOT NULL                        |                             |
| total_sessions            | INT NOT NULL                        |                             |
| raw_notes                 | TEXT NULL                           |                             |
| ai_strengths              | JSON NULL                           | list[str]                   |
| ai_weaknesses             | JSON NULL                           | list[str]                   |
| ai_recommended_next_focus | TEXT NULL                           |                             |
| ai_upsell_recommendation  | TEXT NULL                           | optional                    |

Seed both tables from the rows in `frontend/src/app/core/services/mock-data.ts`
(`MOCK_STUDENTS_CRM`, `MOCK_SESSIONS`) so the dashboard shows identical content
backend-up or backend-down. Session start/end times in the mock are relative to
"today"; seed with fixed representative datetimes.

### Models (`backend/app/models.py`)

```python
class StudentCrm(BaseModel):
    id: Optional[int] = None
    name: str
    phone: str
    courseId: int
    courseName: str
    status: str = "lead"
    progressScore: int = 0
    notes: str = ""
    createdAt: Optional[str] = None   # ISO date string

class AiAnalysis(BaseModel):
    strengths: list[str] = []
    weaknesses: list[str] = []
    recommendedNextFocus: str = ""
    upsellRecommendation: Optional[str] = None

class Session(BaseModel):
    id: Optional[int] = None
    studentId: int
    studentName: str
    instructorId: int
    instructorName: str
    courseId: int
    courseName: str
    startTime: str          # ISO 8601
    endTime: str
    status: str
    sessionNumber: int
    totalSessions: int
    rawNotes: Optional[str] = None
    aiAnalysis: Optional[AiAnalysis] = None

class AnalyzeRequest(BaseModel):
    rawNotes: str
```

The session router maps the four `ai_*` columns into a nested `AiAnalysis` on read
(returning `None` when they are all empty) and back out to columns on write. JSON
columns deserialize to Python lists via PyMySQL.

### Routers

`backend/app/routers/crm.py` — `GET/POST/PUT/DELETE /api/crm/students`, mirroring
`courses.py`. Full CRUD on the server even though the dashboard CRM screen is
read-only for now (consistency with the other epics; user decision).

`backend/app/routers/sessions.py` — `GET/POST/PUT/DELETE /api/sessions` plus
`POST /api/sessions/{id}/analyze` (Part 2). `GET` returns all sessions; the
frontend already filters by instructor for the instructor role.

Both registered in `backend/app/main.py`.

### Frontend

`ApiService.getStudentsCrm()` / `getSessions()` already point at the right URLs with
mock fallback — no change needed for reads. Add write helpers used by the analyze
flow (Part 2). CRM dashboard screen stays read-only (user decision).

---

## Part 2 — Gemini AI session analysis

### `backend/app/ai.py`

```python
def analyze_session_notes(session: Session, raw_notes: str) -> AiAnalysis
```

- Reads `GEMINI_API_KEY` and `GEMINI_MODEL` (default `gemini-1.5-flash`) from env.
- If a key is present: calls `google-generativeai` with an Indonesian-language prompt
  instructing the model to return strict JSON with keys `strengths` (list),
  `weaknesses` (list), `recommendedNextFocus` (string), `upsellRecommendation`
  (string or null). Parses the JSON into `AiAnalysis`. The prompt includes session
  context (student name, course, session N of total) so the upsell recommendation
  can reason about package progress.
- **Fallback (no key, or any exception parsing/calling):** a deterministic stub that
  mirrors the current frontend heuristic — generic strengths/weaknesses plus an
  upsell recommendation when `sessionNumber >= totalSessions - 2`. This keeps the
  endpoint useful with zero configuration and matches graceful degradation.

### Endpoint

`POST /api/sessions/{id}/analyze`, body `AnalyzeRequest { rawNotes }`:

1. Load the session row (404 if missing).
2. Persist `raw_notes`, set `status = 'completed'`.
3. Run `analyze_session_notes`.
4. Persist the analysis into the `ai_*` columns.
5. Return the updated `Session` (with nested `aiAnalysis`).

Protected by `require_auth` (Part 3) — only logged-in admin/instructor users analyze.

### Frontend wiring

- `ApiService.analyzeSession(id, rawNotes): Observable<Session>` →
  `POST /api/sessions/{id}/analyze`, with `catchError` falling back to a locally
  computed mock analysis (the existing heuristic) so offline demo behavior is
  unchanged.
- `SesiComponent.submitNotes()` replaces its `setTimeout` mock with a call to
  `analyzeSession`, updating `selectedSession` from the response. Keeps `isAnalyzing`
  spinner semantics.

### Config / deps

- `backend/requirements.txt`: add `google-generativeai`.
- `backend/.env.example`: add `GEMINI_API_KEY=` and `GEMINI_MODEL=gemini-1.5-flash`.

---

## Part 3 — JWT authentication

### Database

New table `users` + seed:

| column        | type                                | notes                       |
|---------------|-------------------------------------|-----------------------------|
| id            | INT AUTO_INCREMENT PK               |                             |
| username      | VARCHAR(64) NOT NULL UNIQUE         |                             |
| password_hash | VARCHAR(255) NOT NULL               | bcrypt                      |
| name          | VARCHAR(128) NOT NULL               |                             |
| role          | VARCHAR(16) NOT NULL                | admin / instructor          |

Seed `admin`/`admin123` (admin, "Administrator") and `instruktur`/`instruktur123`
(instructor, "Pak Bambang"), bcrypt-hashed. The seed must insert real bcrypt hashes
(generated once and pasted), not plaintext — `seed.sql` stays runnable standalone.

### `backend/app/auth.py`

- `verify_password` / `hash_password` via `passlib[bcrypt]`.
- `create_token(user)` / `decode_token(token)` via `PyJWT`, signing with `JWT_SECRET`
  from env and an expiry (`JWT_EXPIRE_MINUTES`, default e.g. 720).
- FastAPI dependencies `require_auth` (any valid token) and `require_admin` (role ==
  admin), reading the `Authorization: Bearer` header, raising 401/403.

### Router `backend/app/routers/auth.py`

`POST /api/auth/login` body `{ username, password }` → `{ token, user }` on success,
401 on bad credentials. `user` is the same shape as the Angular `AuthUser`.

### Protection matrix

| Endpoint                                   | Protection      |
|--------------------------------------------|-----------------|
| `GET /api/courses`, `GET /api/mechanisms`  | public          |
| `/api/instructors/schedule/public`         | public          |
| `/api/rag/*`, `/api/health`                | public          |
| `POST /api/auth/login`                      | public          |
| courses/instructors/mechanisms POST/PUT/DELETE | `require_admin` |
| `GET /api/instructors/schedule` (admin)    | `require_auth`  |
| `PUT /api/instructors/{id}/schedule`       | `require_admin` |
| `/api/crm/students` (all)                  | `require_admin` |
| `/api/sessions` (all) + `/analyze`         | `require_auth`  |

### Config / deps

- `backend/requirements.txt`: add `PyJWT`, `passlib[bcrypt]`.
- `backend/.env.example`: add `JWT_SECRET=change-me` and `JWT_EXPIRE_MINUTES=720`.

### Frontend

- `AuthService.login(username, password)` becomes async, returning
  `Observable<boolean>` (or emitting the user). It calls `POST /api/auth/login`,
  stores `{ token, user }` in localStorage, sets `currentUser`.
  **Fallback only when the backend is unreachable** (network error / status 0): fall
  through to the existing in-memory `MOCK_USERS` check. A real `401` from a reachable
  backend is authoritative and does **not** fall back (user decision).
- `getToken()` helper; `logout()` clears token too.
- **New `authInterceptor`** (functional `HttpInterceptorFn`, registered in
  `app.config.ts` via `withInterceptors`) attaches `Authorization: Bearer <token>`
  to outgoing requests when a token is present.
- `LoginComponent.onLogin()` subscribes to the observable instead of using its
  `setTimeout`; keeps `isLoading` / `errorMessage` semantics. Distinguish "wrong
  credentials" (reachable 401) from a fallback success.
- `environment.ts` unchanged (same base URL).

---

## Testing / verification

- Backend has no test suite (per CLAUDE.md). Verify manually:
  - `mysql < schema.sql && mysql < seed.sql` succeeds; new tables populated.
  - `uvicorn app.main:app --port 8080`; `GET /api/crm/students` and `GET /api/sessions`
    return seeded rows; Swagger `/docs` lists new endpoints.
  - `POST /api/auth/login` with seeded creds returns a token; protected endpoint
    returns 401 without it and 200 with it.
  - `POST /api/sessions/{id}/analyze` returns an analysis (stub with no key).
- Frontend: `ng build` compiles. With backend up, login hits the API and the
  dashboard loads real CRM/Sessions data; submitting session notes returns a real
  analysis. With backend down, mock login + mock data + mock analysis still work.

## Risks / notes

- `google-generativeai` model id may need adjusting to whatever is current/available;
  the stub fallback means an invalid model degrades gracefully rather than breaking.
- Adding auth to admin reads changes behavior when the backend is up but the user is
  unauthenticated — mitigated because those screens are already behind the Angular
  `authGuard`, and the interceptor attaches the token.
- bcrypt hashes in `seed.sql` are static; documented as such.
