# CRM + Sessions Backend, Gemini 2.5 Analysis, JWT Auth — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the missing CRM and Sessions REST API, real Gemini 2.5 session-note analysis, and real JWT authentication to the YPA Handayani app, preserving the existing camelCase wire contract and graceful-degradation behavior.

**Architecture:** Follow the existing epic-per-router FastAPI + raw-PyMySQL pattern. New tables (`users`, `students_crm`, `sessions`) with seed data mirroring `mock-data.ts`. A standalone `ai.py` module wraps Gemini 2.5 via the `google-genai` SDK with a deterministic stub fallback. JWT auth lives in `auth.py` (token + dependencies) and an `auth` router; protection is applied per the spec's matrix. Frontend `AuthService` becomes HTTP-backed with an interceptor, falling back to the in-memory mock only when the backend is unreachable.

**Tech Stack:** FastAPI 0.115, PyMySQL, PyJWT, bcrypt, google-genai (Gemini 2.5), Angular 18 standalone components, RxJS.

**Reference spec:** `docs/superpowers/specs/2026-06-13-crm-sessions-ai-auth-design.md`

**No backend test suite exists** (per CLAUDE.md). Verification steps use Swagger/curl and `ng build`/`ng test`, matching the project's actual practice. Each task ends with a concrete verification command and a commit.

---

## File Structure

**Backend — create:**
- `backend/app/auth.py` — password hashing, JWT issue/verify, FastAPI auth dependencies
- `backend/app/ai.py` — Gemini 2.5 analyzer + deterministic stub fallback
- `backend/app/routers/auth.py` — `POST /api/auth/login`
- `backend/app/routers/crm.py` — CRM students CRUD
- `backend/app/routers/sessions.py` — sessions CRUD + `/analyze`

**Backend — modify:**
- `backend/schema.sql` — add `users`, `students_crm`, `sessions` tables
- `backend/seed.sql` — seed the three new tables
- `backend/app/models.py` — add `StudentCrm`, `AiAnalysis`, `Session`, `AnalyzeRequest`, `LoginRequest`, `AuthUser`, `LoginResponse`
- `backend/app/main.py` — register the three new routers
- `backend/app/routers/courses.py` — protect writes with `require_admin`
- `backend/app/routers/instructors.py` — protect writes + admin schedule read
- `backend/app/routers/mechanisms.py` — protect writes with `require_admin`
- `backend/requirements.txt` — add `PyJWT`, `bcrypt`, `google-genai`
- `backend/.env.example` — add JWT + Gemini vars

**Frontend — create:**
- `frontend/src/app/core/interceptors/auth.interceptor.ts` — attach Bearer token

**Frontend — modify:**
- `frontend/src/app/core/services/auth.service.ts` — HTTP login + token + fallback
- `frontend/src/app/core/services/api.service.ts` — `analyzeSession()`
- `frontend/src/app/auth/login/login.component.ts` — subscribe to async login
- `frontend/src/app/dashboard/sesi/sesi.component.ts` — call real analyze endpoint
- `frontend/src/app/app.config.ts` — register the interceptor

---

## Phase A — Database & Models

### Task 1: Add new tables to schema

**Files:**
- Modify: `backend/schema.sql` (append after the `mechanisms` table)

- [ ] **Step 1: Append the three new tables**

Add to the end of `backend/schema.sql`:

```sql
-- ── Authentication (JWT) ────────────────────────────────────
CREATE TABLE IF NOT EXISTS users (
  id            INT AUTO_INCREMENT PRIMARY KEY,
  username      VARCHAR(64)  NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  name          VARCHAR(128) NOT NULL,
  role          VARCHAR(16)  NOT NULL DEFAULT 'instructor'
) ENGINE=InnoDB;

-- ── CRM (admin tooling) ─────────────────────────────────────
CREATE TABLE IF NOT EXISTS students_crm (
  id             INT AUTO_INCREMENT PRIMARY KEY,
  name           VARCHAR(128) NOT NULL,
  phone          VARCHAR(32)  NOT NULL,
  course_id      INT          NOT NULL,
  course_name    VARCHAR(128) NOT NULL,
  status         VARCHAR(16)  NOT NULL DEFAULT 'lead',
  progress_score INT          NOT NULL DEFAULT 0,
  notes          TEXT         NOT NULL,
  created_at     TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- ── Training Sessions + AI analysis (admin tooling) ─────────
CREATE TABLE IF NOT EXISTS sessions (
  id                        INT AUTO_INCREMENT PRIMARY KEY,
  student_id                INT          NOT NULL,
  student_name              VARCHAR(128) NOT NULL,
  instructor_id             INT          NOT NULL,
  instructor_name           VARCHAR(128) NOT NULL,
  course_id                 INT          NOT NULL,
  course_name               VARCHAR(128) NOT NULL,
  start_time                DATETIME     NOT NULL,
  end_time                  DATETIME     NOT NULL,
  status                    VARCHAR(16)  NOT NULL DEFAULT 'scheduled',
  session_number            INT          NOT NULL DEFAULT 1,
  total_sessions            INT          NOT NULL DEFAULT 10,
  raw_notes                 TEXT         NULL,
  ai_strengths              JSON         NULL,
  ai_weaknesses             JSON         NULL,
  ai_recommended_next_focus TEXT         NULL,
  ai_upsell_recommendation  TEXT         NULL
) ENGINE=InnoDB;
```

- [ ] **Step 2: Verify the schema applies cleanly**

Run: `mysql -u root -p < backend/schema.sql`
Expected: no errors. Then `mysql -u root -p handayani -e "SHOW TABLES;"` lists `users`, `students_crm`, `sessions` alongside the existing tables.

- [ ] **Step 3: Commit**

```bash
git add backend/schema.sql
git commit -m "feat(db): add users, students_crm, sessions tables"
```

---

### Task 2: Seed the new tables

**Files:**
- Modify: `backend/seed.sql` (append)

- [ ] **Step 1: Append CRM and session seed rows**

Add to the end of `backend/seed.sql`:

```sql
INSERT INTO students_crm (id, name, phone, course_id, course_name, status, progress_score, notes, created_at) VALUES
(1, 'Andi Setiawan', '08123456789', 1, 'Manual - Avanza/Xenia', 'active', 60, 'Sudah menguasai pengereman. Perlu latihan parkir paralel.', '2026-05-01'),
(2, 'Rina Marlina', '08234567890', 2, 'Matic - Avanza/Xenia', 'active', 80, 'Progres sangat baik. Siap ujian minggu depan.', '2026-05-05'),
(3, 'Dewi Pertiwi', '08345678901', 1, 'Manual - Avanza/Xenia', 'lead', 0, 'Tertarik kursus manual. Menunggu konfirmasi jadwal.', '2026-05-20'),
(4, 'Budi Kurniawan', '08456789012', 4, 'Matic Weekend - Avanza/Xenia', 'completed', 100, 'Lulus. SIM A diterbitkan 15 Mei 2026.', '2026-04-01'),
(5, 'Tia Lestari', '08567890123', 1, 'Manual - Avanza/Xenia', 'active', 40, 'Masih kesulitan kopling. Perlu fokus pada perpindahan gigi.', '2026-05-10'),
(6, 'Yuni Wahyuni', '08678901234', 2, 'Matic - Avanza/Xenia', 'lead', 0, 'Chatbot lead — ingin mendaftar kursus matic.', '2026-05-28');

INSERT INTO sessions
(id, student_id, student_name, instructor_id, instructor_name, course_id, course_name,
 start_time, end_time, status, session_number, total_sessions, raw_notes,
 ai_strengths, ai_weaknesses, ai_recommended_next_focus, ai_upsell_recommendation) VALUES
(1, 1, 'Andi Setiawan', 1, 'Pak Bambang', 1, 'Manual - Avanza/Xenia',
 '2026-06-13 09:00:00', '2026-06-13 12:00:00', 'scheduled', 7, 10, NULL,
 NULL, NULL, NULL, NULL),
(2, 2, 'Rina Marlina', 2, 'Bu Sari', 2, 'Matic - Avanza/Xenia',
 '2026-06-13 13:00:00', '2026-06-13 15:00:00', 'completed', 9, 10,
 'Rina sudah sangat baik dalam mengemudi di jalan raya. Parkir mundur masih perlu sedikit penyesuaian, tapi kopling sudah sempurna.',
 '["Pengendalian kemudi di jalan raya", "Kontrol kecepatan yang baik", "Disiplin rambu-rambu lalu lintas"]',
 '["Parkir mundur masih kurang presisi"]',
 'Latihan parkir mundur dan parkir paralel intensif untuk sesi terakhir.', NULL),
(3, 5, 'Tia Lestari', 1, 'Pak Bambang', 1, 'Manual - Avanza/Xenia',
 '2026-06-14 13:00:00', '2026-06-14 15:00:00', 'scheduled', 4, 10,
 'Tia masih kesulitan dengan kopling saat di tanjakan. Perpindahan gigi 1 ke 2 masih ragu-ragu.',
 '["Pengereman sudah baik", "Steering control mulai membaik"]',
 '["Penggunaan kopling di tanjakan", "Perpindahan gigi 1→2 masih ragu"]',
 'Latihan khusus tanjakan dengan teknik setengah kopling dan hill-start assist.',
 'Siswa menunjukkan kesulitan signifikan pada sesi ke-4. Direkomendasikan penambahan 2 sesi khusus tanjakan.');
```

- [ ] **Step 2: Generate bcrypt hashes and append the users seed**

The `users` rows need real bcrypt hashes (no plaintext at rest). Generate them after deps are installed (Task 3 installs `bcrypt`). Run from `backend/` with the venv active:

```bash
python -c "import bcrypt; print(bcrypt.hashpw(b'admin123', bcrypt.gensalt()).decode()); print(bcrypt.hashpw(b'instruktur123', bcrypt.gensalt()).decode())"
```

Paste the two printed hashes into a new block appended to `backend/seed.sql`, replacing `<HASH_ADMIN>` / `<HASH_INSTRUKTUR>` with the exact output lines:

```sql
INSERT INTO users (id, username, password_hash, name, role) VALUES
(1, 'admin', '<HASH_ADMIN>', 'Administrator', 'admin'),
(2, 'instruktur', '<HASH_INSTRUKTUR>', 'Pak Bambang', 'instructor');
```

> Note: bcrypt embeds its salt, so any correctly generated hash verifies. These are static by design (documented in the spec).

- [ ] **Step 3: Verify seed applies**

Run: `mysql -u root -p < backend/seed.sql`
Then: `mysql -u root -p handayani -e "SELECT username, role FROM users; SELECT COUNT(*) FROM students_crm; SELECT COUNT(*) FROM sessions;"`
Expected: 2 users, 6 CRM students, 3 sessions.

- [ ] **Step 4: Commit**

```bash
git add backend/seed.sql
git commit -m "feat(db): seed users, students_crm, sessions"
```

---

### Task 3: Add Python dependencies and env config

**Files:**
- Modify: `backend/requirements.txt`
- Modify: `backend/.env.example`

- [ ] **Step 1: Add dependencies**

Replace the contents of `backend/requirements.txt` with:

```
fastapi==0.115.6
uvicorn==0.34.0
PyMySQL==1.1.1
python-dotenv==1.0.1
PyJWT==2.10.1
bcrypt==4.2.1
google-genai==1.2.0
```

- [ ] **Step 2: Add env vars**

Replace the contents of `backend/.env.example` with:

```
# Copy to .env (or export these) before running the API.
# (Run uvicorn with --port 8080 to match the Angular environment config.)
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASSWORD=
DB_NAME=handayani

# Auth — change JWT_SECRET in any real deployment.
JWT_SECRET=change-me-in-production
JWT_EXPIRE_MINUTES=720

# Gemini 2.5 session analysis. Leave GEMINI_API_KEY blank to use the
# built-in deterministic stub (no external calls).
GEMINI_API_KEY=
GEMINI_MODEL=gemini-2.5-flash
```

- [ ] **Step 3: Install**

Run from `backend/` (venv active): `pip install -r requirements.txt`
Expected: installs PyJWT, bcrypt, google-genai and their deps with no errors.

- [ ] **Step 4: Commit**

```bash
git add backend/requirements.txt backend/.env.example
git commit -m "build(backend): add PyJWT, bcrypt, google-genai deps and env config"
```

---

### Task 4: Add Pydantic models

**Files:**
- Modify: `backend/app/models.py` (append after `Mechanism`)

- [ ] **Step 1: Append models**

Add to the end of `backend/app/models.py`:

```python
# ── CRM & Sessions ──────────────────────────────────────────
class StudentCrm(BaseModel):
    id: Optional[int] = None
    name: str
    phone: str
    courseId: int
    courseName: str
    status: str = "lead"
    progressScore: int = 0
    notes: str = ""
    createdAt: Optional[str] = None


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
    startTime: str
    endTime: str
    status: str = "scheduled"
    sessionNumber: int = 1
    totalSessions: int = 10
    rawNotes: Optional[str] = None
    aiAnalysis: Optional[AiAnalysis] = None


class AnalyzeRequest(BaseModel):
    rawNotes: str


# ── Auth ────────────────────────────────────────────────────
class LoginRequest(BaseModel):
    username: str
    password: str


class AuthUser(BaseModel):
    id: int
    username: str
    name: str
    role: str


class LoginResponse(BaseModel):
    token: str
    user: AuthUser
```

- [ ] **Step 2: Verify it imports**

Run from `backend/`: `python -c "from app import models; print(models.Session.__fields__.keys())"`
Expected: prints the Session field names including `aiAnalysis`. No import error.

- [ ] **Step 3: Commit**

```bash
git add backend/app/models.py
git commit -m "feat(models): add StudentCrm, Session, AiAnalysis, auth models"
```

---

## Phase B — Authentication

### Task 5: Auth core module

**Files:**
- Create: `backend/app/auth.py`

- [ ] **Step 1: Write the module**

Create `backend/app/auth.py`:

```python
"""Authentication: bcrypt password hashing, JWT issue/verify, and FastAPI
dependencies that guard protected endpoints.

JWT_SECRET and expiry come from the environment. Tokens are HS256-signed and
carry the user id, username, name, and role so the dependencies can authorize
without a second DB round-trip.
"""
import os
from datetime import datetime, timedelta, timezone

import bcrypt
import jwt
from fastapi import Depends, HTTPException, status
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer

from .models import AuthUser

_ALGORITHM = "HS256"
_bearer = HTTPBearer(auto_error=False)


def _secret() -> str:
    return os.getenv("JWT_SECRET", "change-me-in-production")


def _expire_minutes() -> int:
    return int(os.getenv("JWT_EXPIRE_MINUTES", "720"))


def hash_password(plain: str) -> str:
    return bcrypt.hashpw(plain.encode(), bcrypt.gensalt()).decode()


def verify_password(plain: str, hashed: str) -> bool:
    try:
        return bcrypt.checkpw(plain.encode(), hashed.encode())
    except (ValueError, TypeError):
        return False


def create_token(user: AuthUser) -> str:
    now = datetime.now(timezone.utc)
    payload = {
        "sub": str(user.id),
        "username": user.username,
        "name": user.name,
        "role": user.role,
        "iat": now,
        "exp": now + timedelta(minutes=_expire_minutes()),
    }
    return jwt.encode(payload, _secret(), algorithm=_ALGORITHM)


def _decode(token: str) -> AuthUser:
    try:
        payload = jwt.decode(token, _secret(), algorithms=[_ALGORITHM])
    except jwt.PyJWTError:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED, detail="Invalid or expired token"
        )
    return AuthUser(
        id=int(payload["sub"]),
        username=payload["username"],
        name=payload["name"],
        role=payload["role"],
    )


def require_auth(
    creds: HTTPAuthorizationCredentials = Depends(_bearer),
) -> AuthUser:
    """Any authenticated user (admin or instructor)."""
    if creds is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED, detail="Not authenticated"
        )
    return _decode(creds.credentials)


def require_admin(user: AuthUser = Depends(require_auth)) -> AuthUser:
    """Admin-only endpoints."""
    if user.role != "admin":
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN, detail="Admin role required"
        )
    return user
```

- [ ] **Step 2: Verify round-trip**

Run from `backend/`:
```bash
python -c "from app.auth import hash_password, verify_password, create_token, _decode; from app.models import AuthUser; h=hash_password('x'); print(verify_password('x',h), verify_password('y',h)); u=AuthUser(id=1,username='a',name='A',role='admin'); t=create_token(u); print(_decode(t).role)"
```
Expected: `True False` then `admin`.

- [ ] **Step 3: Commit**

```bash
git add backend/app/auth.py
git commit -m "feat(auth): bcrypt hashing, JWT tokens, auth dependencies"
```

---

### Task 6: Auth router (login)

**Files:**
- Create: `backend/app/routers/auth.py`
- Modify: `backend/app/main.py`

- [ ] **Step 1: Write the login router**

Create `backend/app/routers/auth.py`:

```python
"""Authentication endpoints. POST /api/auth/login verifies credentials against
the users table and returns a JWT plus the user profile (same shape as the
Angular AuthUser)."""
from fastapi import APIRouter, Depends, HTTPException, status
from pymysql.connections import Connection

from ..auth import create_token, verify_password
from ..database import get_db
from ..models import AuthUser, LoginRequest, LoginResponse

router = APIRouter(prefix="/api/auth", tags=["auth"])


@router.post("/login", response_model=LoginResponse)
def login(body: LoginRequest, db: Connection = Depends(get_db)):
    with db.cursor() as cur:
        cur.execute(
            "SELECT id, username, password_hash, name, role FROM users WHERE username=%s",
            (body.username,),
        )
        row = cur.fetchone()
    if row is None or not verify_password(body.password, row["password_hash"]):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Username atau password salah",
        )
    user = AuthUser(id=row["id"], username=row["username"], name=row["name"], role=row["role"])
    return LoginResponse(token=create_token(user), user=user)
```

- [ ] **Step 2: Register the router in main.py**

In `backend/app/main.py`, update the import line and add the include. Change:

```python
from .routers import courses, instructors, mechanisms, rag
```
to:
```python
from .routers import auth, courses, crm, instructors, mechanisms, rag, sessions
```

And add after `app.include_router(rag.router)`:

```python
app.include_router(auth.router)
app.include_router(crm.router)
app.include_router(sessions.router)
```

> Note: `crm` and `sessions` modules are created in Tasks 8 and 10. If running the server before those exist, this import will fail — implement Tasks 8 and 10 before starting uvicorn. (Subagent-driven execution does these in order.)

- [ ] **Step 3: Commit**

```bash
git add backend/app/routers/auth.py backend/app/main.py
git commit -m "feat(auth): add POST /api/auth/login and register new routers"
```

---

### Task 7: Protect existing write endpoints

**Files:**
- Modify: `backend/app/routers/courses.py`
- Modify: `backend/app/routers/instructors.py`
- Modify: `backend/app/routers/mechanisms.py`

- [ ] **Step 1: Protect courses writes**

In `backend/app/routers/courses.py`, add to the imports:

```python
from ..auth import require_admin
```

Add `_: object = Depends(require_admin)` as the final parameter of `create_course`, `update_course`, and `delete_course`. Example for `create_course`:

```python
@router.post("", response_model=Course, status_code=status.HTTP_201_CREATED)
def create_course(course: Course, db: Connection = Depends(get_db), _: object = Depends(require_admin)):
```

Apply the same `_: object = Depends(require_admin)` final parameter to `update_course` and `delete_course`. Leave `list_courses` (GET) public.

- [ ] **Step 2: Protect mechanisms writes**

In `backend/app/routers/mechanisms.py`, add `from ..auth import require_admin` and add `_: object = Depends(require_admin)` as the final parameter of the create, update, and delete handlers. Leave the GET handler public.

- [ ] **Step 3: Protect instructors writes + admin schedule read**

In `backend/app/routers/instructors.py`, add `from ..auth import require_admin, require_auth`. Then:
- `list_instructors_with_schedule` (the admin `GET /schedule`): add `_: object = Depends(require_auth)` final parameter.
- `create_instructor`, `update_instructor`, `delete_instructor`, `update_schedule`: add `_: object = Depends(require_admin)` final parameter.
- Leave `list_instructors_with_public_schedule` (`/schedule/public`) public.

- [ ] **Step 4: Verify protection**

Start MySQL and the server: `uvicorn app.main:app --port 8080` (from `backend/`, after Tasks 8 & 10 exist). Then:
```bash
curl -s -o /dev/null -w "%{http_code}\n" -X POST http://localhost:8080/api/courses -H "Content-Type: application/json" -d "{}"
```
Expected: `401` (or `403`). And `curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/api/courses` → `200` (public read still works).

- [ ] **Step 5: Commit**

```bash
git add backend/app/routers/courses.py backend/app/routers/instructors.py backend/app/routers/mechanisms.py
git commit -m "feat(auth): require admin/auth on write and admin-read endpoints"
```

---

## Phase C — CRM & Sessions API + AI

### Task 8: CRM router

**Files:**
- Create: `backend/app/routers/crm.py`

- [ ] **Step 1: Write the router**

Create `backend/app/routers/crm.py`:

```python
"""CRM students CRUD (admin tooling). Admin-only per the auth matrix."""
from fastapi import APIRouter, Depends, status
from pymysql.connections import Connection

from ..auth import require_admin
from ..database import get_db
from ..models import StudentCrm

router = APIRouter(prefix="/api/crm/students", tags=["crm"])

_SELECT = (
    "SELECT id, name, phone, course_id AS courseId, course_name AS courseName, "
    "status, progress_score AS progressScore, notes, "
    "DATE_FORMAT(created_at, '%%Y-%%m-%%d') AS createdAt "
    "FROM students_crm ORDER BY created_at DESC, id DESC"
)


@router.get("", response_model=list[StudentCrm])
def list_students(db: Connection = Depends(get_db), _: object = Depends(require_admin)):
    with db.cursor() as cur:
        cur.execute(_SELECT)
        return [StudentCrm(**row) for row in cur.fetchall()]


@router.post("", response_model=StudentCrm, status_code=status.HTTP_201_CREATED)
def create_student(student: StudentCrm, db: Connection = Depends(get_db), _: object = Depends(require_admin)):
    with db.cursor() as cur:
        cur.execute(
            "INSERT INTO students_crm "
            "(name, phone, course_id, course_name, status, progress_score, notes) "
            "VALUES (%(name)s, %(phone)s, %(courseId)s, %(courseName)s, "
            "%(status)s, %(progressScore)s, %(notes)s)",
            student.model_dump(exclude={"id", "createdAt"}),
        )
        student.id = cur.lastrowid
    return student


@router.put("/{student_id}", response_model=StudentCrm)
def update_student(student_id: int, student: StudentCrm, db: Connection = Depends(get_db), _: object = Depends(require_admin)):
    student.id = student_id
    with db.cursor() as cur:
        cur.execute(
            "UPDATE students_crm SET name=%(name)s, phone=%(phone)s, "
            "course_id=%(courseId)s, course_name=%(courseName)s, status=%(status)s, "
            "progress_score=%(progressScore)s, notes=%(notes)s WHERE id=%(id)s",
            student.model_dump(exclude={"createdAt"}),
        )
    return student


@router.delete("/{student_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_student(student_id: int, db: Connection = Depends(get_db), _: object = Depends(require_admin)):
    with db.cursor() as cur:
        cur.execute("DELETE FROM students_crm WHERE id=%(id)s", {"id": student_id})
```

> Note the `%%Y-%%m-%%d` double-percent: PyMySQL treats `%` as a parameter marker, so literal percents in SQL must be escaped.

- [ ] **Step 2: Verify (after server runs in Task 11)**

`curl` with a token (see Task 11 verification) `GET http://localhost:8080/api/crm/students` returns 6 students with `courseId`/`createdAt` camelCase fields.

- [ ] **Step 3: Commit**

```bash
git add backend/app/routers/crm.py
git commit -m "feat(crm): add /api/crm/students CRUD (admin-only)"
```

---

### Task 9: Gemini 2.5 analyzer module

**Files:**
- Create: `backend/app/ai.py`

- [ ] **Step 1: Write the analyzer**

Create `backend/app/ai.py`:

```python
"""Session-note analysis. Uses Gemini 2.5 via the google-genai SDK when
GEMINI_API_KEY is set; otherwise falls back to a deterministic stub so the
feature works with zero configuration (graceful degradation, as elsewhere in
this app)."""
import json
import os

from .models import AiAnalysis, Session

_PROMPT = """Anda adalah evaluator instruktur mengemudi di YPA Handayani.
Analisis catatan sesi berikut dan kembalikan HANYA JSON valid (tanpa markdown).

Konteks sesi:
- Siswa: {student}
- Kursus: {course}
- Sesi ke-{n} dari {total}

Catatan instruktur:
\"\"\"{notes}\"\"\"

Kembalikan JSON dengan struktur persis:
{{
  "strengths": ["..."],
  "weaknesses": ["..."],
  "recommendedNextFocus": "...",
  "upsellRecommendation": "... atau null"
}}
Semua teks dalam Bahasa Indonesia. "upsellRecommendation" diisi hanya jika
siswa mendekati akhir paket namun masih ada kelemahan signifikan; jika tidak,
gunakan null."""


def _stub(session: Session, raw_notes: str) -> AiAnalysis:
    near_end = session.sessionNumber >= session.totalSessions - 2
    return AiAnalysis(
        strengths=["Kontrol kemudi dasar", "Kepatuhan instruksi"],
        weaknesses=["Perlu perbaikan pada saat parkir", "Masih ragu saat perpindahan gigi"],
        recommendedNextFocus="Fokus pada teknik parkir paralel dan mundur di area sempit.",
        upsellRecommendation=(
            "Siswa hampir menyelesaikan paket namun masih ada kekurangan teknis. "
            "Tawarkan paket top-up 3 sesi tambahan."
            if near_end
            else None
        ),
    )


def analyze_session_notes(session: Session, raw_notes: str) -> AiAnalysis:
    api_key = os.getenv("GEMINI_API_KEY", "").strip()
    if not api_key:
        return _stub(session, raw_notes)
    try:
        from google import genai
        from google.genai import types

        client = genai.Client(api_key=api_key)
        prompt = _PROMPT.format(
            student=session.studentName,
            course=session.courseName,
            n=session.sessionNumber,
            total=session.totalSessions,
            notes=raw_notes,
        )
        resp = client.models.generate_content(
            model=os.getenv("GEMINI_MODEL", "gemini-2.5-flash"),
            contents=prompt,
            config=types.GenerateContentConfig(response_mime_type="application/json"),
        )
        data = json.loads(resp.text)
        upsell = data.get("upsellRecommendation")
        if isinstance(upsell, str) and upsell.strip().lower() in ("null", "none", ""):
            upsell = None
        return AiAnalysis(
            strengths=list(data.get("strengths", [])),
            weaknesses=list(data.get("weaknesses", [])),
            recommendedNextFocus=data.get("recommendedNextFocus", ""),
            upsellRecommendation=upsell,
        )
    except Exception:
        # Any SDK/network/parse failure degrades to the deterministic stub.
        return _stub(session, raw_notes)
```

- [ ] **Step 2: Verify the stub path**

Run from `backend/` (no GEMINI_API_KEY set):
```bash
python -c "from app.ai import analyze_session_notes; from app.models import Session; s=Session(studentId=1,studentName='T',instructorId=1,instructorName='B',courseId=1,courseName='Manual',startTime='2026-06-13T09:00:00',endTime='2026-06-13T12:00:00',sessionNumber=9,totalSessions=10); a=analyze_session_notes(s,'notes'); print(a.upsellRecommendation is not None)"
```
Expected: `True` (session 9/10 triggers the upsell branch).

- [ ] **Step 3: Commit**

```bash
git add backend/app/ai.py
git commit -m "feat(ai): Gemini 2.5 session analyzer with deterministic stub fallback"
```

---

### Task 10: Sessions router (CRUD + analyze)

**Files:**
- Create: `backend/app/routers/sessions.py`

- [ ] **Step 1: Write the router**

Create `backend/app/routers/sessions.py`:

```python
"""Training sessions CRUD + AI analysis (admin tooling). Requires auth.

The four ai_* columns map to a nested AiAnalysis object on the wire: they are
returned as `aiAnalysis` (or null when empty) to match the Angular Session
interface. JSON columns deserialize to Python lists via PyMySQL."""
import json

from fastapi import APIRouter, Depends, HTTPException, status
from pymysql.connections import Connection

from ..ai import analyze_session_notes
from ..auth import require_auth
from ..database import get_db
from ..models import AiAnalysis, AnalyzeRequest, Session

router = APIRouter(prefix="/api/sessions", tags=["sessions"])

_SELECT = (
    "SELECT id, student_id AS studentId, student_name AS studentName, "
    "instructor_id AS instructorId, instructor_name AS instructorName, "
    "course_id AS courseId, course_name AS courseName, "
    "DATE_FORMAT(start_time, '%%Y-%%m-%%dT%%H:%%i:%%s') AS startTime, "
    "DATE_FORMAT(end_time, '%%Y-%%m-%%dT%%H:%%i:%%s') AS endTime, "
    "status, session_number AS sessionNumber, total_sessions AS totalSessions, "
    "raw_notes AS rawNotes, ai_strengths AS aiStrengths, ai_weaknesses AS aiWeaknesses, "
    "ai_recommended_next_focus AS aiNextFocus, ai_upsell_recommendation AS aiUpsell "
    "FROM sessions"
)


def _as_list(value) -> list[str]:
    if value is None:
        return []
    if isinstance(value, str):
        return json.loads(value)
    return list(value)


def _row_to_session(row: dict) -> Session:
    analysis = None
    if row["aiNextFocus"] or row["aiStrengths"] or row["aiWeaknesses"]:
        analysis = AiAnalysis(
            strengths=_as_list(row["aiStrengths"]),
            weaknesses=_as_list(row["aiWeaknesses"]),
            recommendedNextFocus=row["aiNextFocus"] or "",
            upsellRecommendation=row["aiUpsell"],
        )
    return Session(
        id=row["id"],
        studentId=row["studentId"],
        studentName=row["studentName"],
        instructorId=row["instructorId"],
        instructorName=row["instructorName"],
        courseId=row["courseId"],
        courseName=row["courseName"],
        startTime=row["startTime"],
        endTime=row["endTime"],
        status=row["status"],
        sessionNumber=row["sessionNumber"],
        totalSessions=row["totalSessions"],
        rawNotes=row["rawNotes"],
        aiAnalysis=analysis,
    )


def _fetch_one(db: Connection, session_id: int) -> dict | None:
    with db.cursor() as cur:
        cur.execute(_SELECT + " WHERE id=%s", (session_id,))
        return cur.fetchone()


@router.get("", response_model=list[Session])
def list_sessions(db: Connection = Depends(get_db), _: object = Depends(require_auth)):
    with db.cursor() as cur:
        cur.execute(_SELECT + " ORDER BY start_time")
        return [_row_to_session(row) for row in cur.fetchall()]


@router.post("", response_model=Session, status_code=status.HTTP_201_CREATED)
def create_session(session: Session, db: Connection = Depends(get_db), _: object = Depends(require_auth)):
    with db.cursor() as cur:
        cur.execute(
            "INSERT INTO sessions "
            "(student_id, student_name, instructor_id, instructor_name, course_id, "
            "course_name, start_time, end_time, status, session_number, total_sessions, raw_notes) "
            "VALUES (%(studentId)s, %(studentName)s, %(instructorId)s, %(instructorName)s, "
            "%(courseId)s, %(courseName)s, %(startTime)s, %(endTime)s, %(status)s, "
            "%(sessionNumber)s, %(totalSessions)s, %(rawNotes)s)",
            session.model_dump(exclude={"id", "aiAnalysis"}),
        )
        session.id = cur.lastrowid
    return session


@router.put("/{session_id}", response_model=Session)
def update_session(session_id: int, session: Session, db: Connection = Depends(get_db), _: object = Depends(require_auth)):
    session.id = session_id
    with db.cursor() as cur:
        cur.execute(
            "UPDATE sessions SET student_id=%(studentId)s, student_name=%(studentName)s, "
            "instructor_id=%(instructorId)s, instructor_name=%(instructorName)s, "
            "course_id=%(courseId)s, course_name=%(courseName)s, start_time=%(startTime)s, "
            "end_time=%(endTime)s, status=%(status)s, session_number=%(sessionNumber)s, "
            "total_sessions=%(totalSessions)s, raw_notes=%(rawNotes)s WHERE id=%(id)s",
            session.model_dump(exclude={"aiAnalysis"}),
        )
    return session


@router.delete("/{session_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_session(session_id: int, db: Connection = Depends(get_db), _: object = Depends(require_auth)):
    with db.cursor() as cur:
        cur.execute("DELETE FROM sessions WHERE id=%(id)s", {"id": session_id})


@router.post("/{session_id}/analyze", response_model=Session)
def analyze_session(session_id: int, body: AnalyzeRequest, db: Connection = Depends(get_db), _: object = Depends(require_auth)):
    row = _fetch_one(db, session_id)
    if row is None:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="Session not found")
    current = _row_to_session(row)
    analysis = analyze_session_notes(current, body.rawNotes)
    with db.cursor() as cur:
        cur.execute(
            "UPDATE sessions SET raw_notes=%(notes)s, status='completed', "
            "ai_strengths=%(strengths)s, ai_weaknesses=%(weaknesses)s, "
            "ai_recommended_next_focus=%(focus)s, ai_upsell_recommendation=%(upsell)s "
            "WHERE id=%(id)s",
            {
                "notes": body.rawNotes,
                "strengths": json.dumps(analysis.strengths),
                "weaknesses": json.dumps(analysis.weaknesses),
                "focus": analysis.recommendedNextFocus,
                "upsell": analysis.upsellRecommendation,
                "id": session_id,
            },
        )
    current.rawNotes = body.rawNotes
    current.status = "completed"
    current.aiAnalysis = analysis
    return current
```

- [ ] **Step 2: Commit**

```bash
git add backend/app/routers/sessions.py
git commit -m "feat(sessions): CRUD + POST /analyze with Gemini analysis"
```

---

### Task 11: End-to-end backend verification

**Files:** none (verification only)

- [ ] **Step 1: Start the server**

From `backend/` with MySQL running and schema+seed applied:
`uvicorn app.main:app --reload --port 8080`
Expected: starts with no import errors; `/docs` lists auth, crm, sessions routes.

- [ ] **Step 2: Log in and capture a token**

```bash
curl -s -X POST http://localhost:8080/api/auth/login -H "Content-Type: application/json" -d "{\"username\":\"admin\",\"password\":\"admin123\"}"
```
Expected: JSON with `token` and `user.role == "admin"`. Bad password → `401`.

- [ ] **Step 3: Hit protected endpoints with the token**

```bash
TOKEN=... # paste token value
curl -s http://localhost:8080/api/crm/students -H "Authorization: Bearer $TOKEN" | head
curl -s http://localhost:8080/api/sessions -H "Authorization: Bearer $TOKEN" | head
```
Expected: 6 students; 3 sessions, session id 2 carrying a nested `aiAnalysis`, session id 1 with `aiAnalysis: null`. Without the header → `401`.

- [ ] **Step 4: Analyze a session (stub path, no Gemini key)**

```bash
curl -s -X POST http://localhost:8080/api/sessions/1/analyze -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "{\"rawNotes\":\"Siswa latihan parkir.\"}"
```
Expected: returns session 1 with `status: "completed"` and a populated `aiAnalysis` (upsell present since 7/10 ≥ 8? no — 7 < 8, so upsell null). Re-query confirms persistence.

- [ ] **Step 5: Commit (if any fixes were needed)**

```bash
git commit -am "fix(backend): address issues found in e2e verification" --allow-empty
```

---

## Phase D — Frontend

### Task 12: HTTP-backed AuthService with offline fallback

**Files:**
- Modify: `frontend/src/app/core/services/auth.service.ts`

- [ ] **Step 1: Rewrite the service**

Replace `frontend/src/app/core/services/auth.service.ts` with:

```typescript
import { Injectable, signal } from '@angular/core';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { Router } from '@angular/router';
import { Observable, of } from 'rxjs';
import { catchError, map } from 'rxjs/operators';
import { environment } from '../../../environments/environment';

export type UserRole = 'admin' | 'instructor';

export interface AuthUser {
  id: number;
  username: string;
  name: string;
  role: UserRole;
}

interface LoginResponse {
  token: string;
  user: AuthUser;
}

interface MockCredential {
  username: string;
  password: string;
  user: AuthUser;
}

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly STORAGE_KEY = 'handayani_auth_user';
  private readonly TOKEN_KEY = 'handayani_auth_token';
  public currentUser = signal<AuthUser | null>(null);

  private readonly MOCK_USERS: MockCredential[] = [
    { username: 'admin', password: 'admin123', user: { id: 1, username: 'admin', name: 'Administrator', role: 'admin' } },
    { username: 'instruktur', password: 'instruktur123', user: { id: 2, username: 'instruktur', name: 'Pak Bambang', role: 'instructor' } }
  ];

  constructor(private http: HttpClient, private router: Router) {
    this.restoreSession();
  }

  private restoreSession(): void {
    const stored = localStorage.getItem(this.STORAGE_KEY);
    if (stored) {
      try {
        this.currentUser.set(JSON.parse(stored));
      } catch {
        localStorage.removeItem(this.STORAGE_KEY);
      }
    }
  }

  /**
   * Logs in against POST /api/auth/login. A real 401 from a reachable backend
   * is authoritative (emits false). Only a connection failure (status 0) falls
   * back to the in-memory mock credentials so offline demos still work.
   */
  login(username: string, password: string): Observable<boolean> {
    return this.http.post<LoginResponse>(`${environment.apiBaseUrl}/api/auth/login`, { username, password }).pipe(
      map(res => {
        this.persist(res.user, res.token);
        return true;
      }),
      catchError((err: HttpErrorResponse) => {
        if (err.status === 0) {
          return of(this.mockLogin(username, password));
        }
        return of(false);
      })
    );
  }

  private mockLogin(username: string, password: string): boolean {
    const match = this.MOCK_USERS.find(u => u.username === username && u.password === password);
    if (match) {
      this.persist(match.user, 'mock-offline-token');
      return true;
    }
    return false;
  }

  private persist(user: AuthUser, token: string): void {
    this.currentUser.set(user);
    localStorage.setItem(this.STORAGE_KEY, JSON.stringify(user));
    localStorage.setItem(this.TOKEN_KEY, token);
  }

  getToken(): string | null {
    return localStorage.getItem(this.TOKEN_KEY);
  }

  logout(): void {
    this.currentUser.set(null);
    localStorage.removeItem(this.STORAGE_KEY);
    localStorage.removeItem(this.TOKEN_KEY);
    this.router.navigate(['/login']);
  }

  isAuthenticated(): boolean {
    return this.currentUser() !== null;
  }

  isAdmin(): boolean {
    return this.currentUser()?.role === 'admin';
  }
}
```

- [ ] **Step 2: Verify build**

Run from `frontend/`: `ng build`
Expected: compiles. (Login component still references `login()` synchronously — fixed in Task 14; if `ng build` fails only on that, proceed to Task 14 then re-check.)

- [ ] **Step 3: Commit**

```bash
git add frontend/src/app/core/services/auth.service.ts
git commit -m "feat(auth-fe): HTTP login + JWT storage, offline mock fallback"
```

---

### Task 13: Auth HTTP interceptor

**Files:**
- Create: `frontend/src/app/core/interceptors/auth.interceptor.ts`
- Modify: `frontend/src/app/app.config.ts`

- [ ] **Step 1: Write the interceptor**

Create `frontend/src/app/core/interceptors/auth.interceptor.ts`:

```typescript
import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { AuthService } from '../services/auth.service';

/** Attaches the JWT as a Bearer token to outgoing requests when present. */
export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const token = inject(AuthService).getToken();
  if (token && token !== 'mock-offline-token') {
    req = req.clone({ setHeaders: { Authorization: `Bearer ${token}` } });
  }
  return next(req);
};
```

- [ ] **Step 2: Register it in app.config.ts**

In `frontend/src/app/app.config.ts`, find the `provideHttpClient(...)` call and add the interceptor. Update the import to include `withInterceptors` from `@angular/common/http`, import `authInterceptor`, and change the provider to:

```typescript
provideHttpClient(withInterceptors([authInterceptor]))
```

> If `provideHttpClient` currently passes other args (e.g. `withFetch()`), keep them: `provideHttpClient(withFetch(), withInterceptors([authInterceptor]))`.

- [ ] **Step 3: Verify build**

Run from `frontend/`: `ng build`
Expected: compiles with the interceptor registered.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/app/core/interceptors/auth.interceptor.ts frontend/src/app/app.config.ts
git commit -m "feat(auth-fe): attach Bearer token via HTTP interceptor"
```

---

### Task 14: Update LoginComponent for async login

**Files:**
- Modify: `frontend/src/app/auth/login/login.component.ts`

- [ ] **Step 1: Replace onLogin to subscribe**

In `frontend/src/app/auth/login/login.component.ts`, replace the `onLogin()` method body (the version using `setTimeout`) with:

```typescript
  onLogin(): void {
    if (!this.username || !this.password) {
      this.errorMessage.set('Username dan password tidak boleh kosong.');
      return;
    }

    this.isLoading.set(true);
    this.errorMessage.set('');

    this.authService.login(this.username, this.password).subscribe(success => {
      this.isLoading.set(false);
      if (success) {
        this.router.navigate(['/dashboard']);
      } else {
        this.errorMessage.set('Username atau password salah. Silakan coba lagi.');
      }
    });
  }
```

- [ ] **Step 2: Verify build**

Run from `frontend/`: `ng build`
Expected: compiles cleanly now that `login()` returns an Observable.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/app/auth/login/login.component.ts
git commit -m "feat(auth-fe): subscribe to async login observable"
```

---

### Task 15: Wire session analysis to the real endpoint

**Files:**
- Modify: `frontend/src/app/core/services/api.service.ts`
- Modify: `frontend/src/app/dashboard/sesi/sesi.component.ts`

- [ ] **Step 1: Add analyzeSession to ApiService**

In `frontend/src/app/core/services/api.service.ts`, inside the "CRM & SESSIONS" section (after `getSessions()`), add:

```typescript
  /**
   * Sends raw instructor notes for AI analysis. On HTTP error (backend down)
   * falls back to a locally-computed mock analysis so the demo keeps working.
   */
  analyzeSession(session: Session, rawNotes: string): Observable<Session> {
    return this.http.post<Session>(`${this.baseUrl}/api/sessions/${session.id}/analyze`, { rawNotes }).pipe(
      catchError(() => of(this.mockAnalyze(session, rawNotes)))
    );
  }

  private mockAnalyze(session: Session, rawNotes: string): Session {
    const nearEnd = session.sessionNumber >= session.totalSessions - 2;
    return {
      ...session,
      rawNotes,
      status: 'completed',
      aiAnalysis: {
        strengths: ['Kontrol kemudi dasar', 'Kepatuhan instruksi'],
        weaknesses: ['Perlu perbaikan pada saat parkir', 'Masih ragu saat perpindahan gigi'],
        recommendedNextFocus: 'Fokus pada teknik parkir paralel dan mundur di area sempit.',
        upsellRecommendation: nearEnd
          ? 'Siswa hampir menyelesaikan paket namun masih ada kekurangan teknis. Tawarkan paket top-up 3 sesi tambahan.'
          : undefined
      }
    };
  }
```

- [ ] **Step 2: Update SesiComponent.submitNotes**

In `frontend/src/app/dashboard/sesi/sesi.component.ts`, replace the `submitNotes()` method (the `setTimeout` mock version) with:

```typescript
  submitNotes() {
    if (!this.selectedSession || !this.newNotes.trim()) return;

    this.isAnalyzing = true;
    this.api.analyzeSession(this.selectedSession, this.newNotes).subscribe(updated => {
      const idx = this.sessions.findIndex(s => s.id === updated.id);
      if (idx !== -1) this.sessions[idx] = updated;
      this.selectedSession = updated;
      this.newNotes = updated.rawNotes || '';
      this.isAnalyzing = false;
    });
  }
```

- [ ] **Step 3: Verify build**

Run from `frontend/`: `ng build`
Expected: compiles cleanly.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/app/core/services/api.service.ts frontend/src/app/dashboard/sesi/sesi.component.ts
git commit -m "feat(sessions-fe): call real /analyze endpoint with mock fallback"
```

---

### Task 16: Full-stack smoke verification

**Files:** none (verification only)

- [ ] **Step 1: Run both servers**

Backend: `uvicorn app.main:app --port 8080` (from `backend/`, MySQL up, seed applied).
Frontend: `npm start` (from `frontend/`).

- [ ] **Step 2: Verify real login + data**

In the browser at `http://localhost:4200/login`, log in as `admin`/`admin123`.
Expected: redirected to dashboard; Overview stats and CRM screen show the seeded students; Sesi screen shows 3 sessions. Check the Network tab: `/api/auth/login` returned a token, subsequent `/api/crm/students` and `/api/sessions` carried the `Authorization: Bearer` header.

- [ ] **Step 3: Verify AI analysis flow**

Open a `scheduled` session, enter notes, submit.
Expected: spinner shows, then the session flips to `completed` with a populated analysis (from the backend stub, or real Gemini if a key is configured in `backend/.env`).

- [ ] **Step 4: Verify offline fallback**

Stop the backend. Reload `/login`, log in as `admin`/`admin123`.
Expected: login still succeeds (mock fallback, status 0), dashboard shows mock data, analyzing notes still returns a mock analysis. Logging in with a wrong password while the backend is **up** must fail (no fallback).

- [ ] **Step 5: Final commit**

```bash
git commit -am "test: full-stack smoke verification of auth, CRM, sessions, AI" --allow-empty
```

---

## Self-Review Notes

- **Spec coverage:** Part 1 (CRM/Sessions backend) → Tasks 1,2,4,8,10. Part 2 (Gemini 2.5 analysis) → Tasks 9,10,15. Part 3 (JWT auth) → Tasks 3,5,6,7,12,13,14. Protection matrix → Task 7 + per-router deps. Frontend wiring → Tasks 12–15. Verification → Tasks 11,16.
- **Login fallback decision** (only-when-unreachable) → Task 12 `catchError` checks `err.status === 0`.
- **CRM decision** (backend CRUD, read-only UI) → Task 8 builds full CRUD; no CRM UI task included.
- **Type consistency:** `AiAnalysis`/`Session` field names match between `models.py` (Task 4), the SQL aliases (Tasks 8/10), and the Angular interfaces (unchanged). `getToken()`/`login()` signatures match between `auth.service.ts` (Task 12), the interceptor (Task 13), and `LoginComponent` (Task 14).
- **PyMySQL `%%` escaping** noted in CRM and sessions SELECTs (DATE_FORMAT literals).
- **Router import ordering:** `main.py` (Task 6) imports `crm`/`sessions` which are created in Tasks 8/10 — execution order handles this; noted inline.
