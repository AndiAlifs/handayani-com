# CRM + Sessions Backend, Gemini 2.5 Analysis, NYAMPE-JWT Validation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

> **REVISED 2026-06-13** after the NYAMPE attendance integration merged into `main`.
> That merge already delivered authentication (Go service + gateway proxy) and the
> frontend auth stack (AuthService, interceptor, login). So this plan **no longer
> builds auth** — it only *validates* the Go-issued JWT on the new endpoints. Tasks
> for a `users` table, bcrypt, token minting, a login router, and frontend auth
> rewrites are removed. See `docs/superpowers/specs/2026-06-13-crm-sessions-ai-auth-design.md`
> ("Part 3 (REVISED)").

**Goal:** Add the missing CRM and Sessions REST API plus real Gemini 2.5 session-note analysis to the FastAPI backend, protected by validating the NYAMPE Go-issued JWT, preserving the existing camelCase wire contract and graceful-degradation behavior.

**Architecture:** Follow the existing epic-per-router FastAPI + raw-PyMySQL pattern. New tables (`students_crm`, `sessions`) seeded to mirror `mock-data.ts`. A standalone `ai.py` wraps Gemini 2.5 (`google-genai`) with a deterministic stub fallback. `app/auth.py` is decode-only: it verifies the Go HS256 JWT (shared `JWT_SECRET`) and exposes `require_auth` / `require_manager` dependencies. The frontend already authenticates and attaches the token via its interceptor; the only frontend change is wiring session analysis to the new `/analyze` endpoint.

**Tech Stack:** FastAPI 0.115, PyMySQL, PyJWT (decode only), google-genai (Gemini 2.5), Angular 18, RxJS.

**Key integration constraint:** FastAPI and the Go service **must share the same `JWT_SECRET`** (Go dev fallback: `super-secret-key-default`). Go roles are `employee | manager | instructor`; `manager` is the elevated/admin-equivalent role.

**No backend test suite exists** (per CLAUDE.md). Verification uses Swagger/curl and `ng build`. Protected-endpoint checks use a locally-minted test token (same secret/claims as Go) so they don't require the Go service running.

---

## File Structure

**Backend — create:**
- `backend/app/auth.py` — decode-only Go-JWT validation + `require_auth`/`require_manager`
- `backend/app/ai.py` — Gemini 2.5 analyzer + deterministic stub fallback
- `backend/app/routers/crm.py` — CRM students CRUD (manager-only)
- `backend/app/routers/sessions.py` — sessions CRUD + `/analyze`

**Backend — modify:**
- `backend/schema.sql` — add `students_crm`, `sessions` tables
- `backend/seed.sql` — seed the two new tables
- `backend/app/models.py` — add `StudentCrm`, `AiAnalysis`, `Session`, `AnalyzeRequest`
- `backend/app/main.py` — register `crm` and `sessions` routers
- `backend/requirements.txt` — add `PyJWT`, `google-genai`
- `backend/.env.example` — add `JWT_SECRET`, `GEMINI_API_KEY`, `GEMINI_MODEL`

**Frontend — modify:**
- `frontend/src/app/core/services/api.service.ts` — add `analyzeSession()`
- `frontend/src/app/dashboard/sesi/sesi.component.ts` — call the real `/analyze` endpoint

---

## Phase A — Database, Deps & Models

### Task 1: Add CRM + Sessions tables

**Files:**
- Modify: `backend/schema.sql` (append after the `mechanisms` table)

- [ ] **Step 1: Append the two tables**

Add to the end of `backend/schema.sql`:

```sql
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

- [ ] **Step 2: Verify schema applies**

Run: `mysql -u root -p < backend/schema.sql`
Then: `mysql -u root -p handayani -e "SHOW TABLES;"`
Expected: no errors; `students_crm` and `sessions` listed.

- [ ] **Step 3: Commit**

```bash
git add backend/schema.sql
git commit -m "feat(db): add students_crm and sessions tables"
```

---

### Task 2: Seed CRM + Sessions

**Files:**
- Modify: `backend/seed.sql` (append)

- [ ] **Step 1: Append seed rows**

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

- [ ] **Step 2: Verify seed applies**

Run: `mysql -u root -p < backend/seed.sql`
Then: `mysql -u root -p handayani -e "SELECT COUNT(*) FROM students_crm; SELECT COUNT(*) FROM sessions;"`
Expected: 6 students, 3 sessions.

- [ ] **Step 3: Commit**

```bash
git add backend/seed.sql
git commit -m "feat(db): seed students_crm and sessions"
```

---

### Task 3: Dependencies and env config

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

# MUST match the NYAMPE Go service's JWT_SECRET so FastAPI can validate its
# tokens. The Go dev fallback is "super-secret-key-default".
JWT_SECRET=super-secret-key-default

# Gemini 2.5 session analysis. Leave GEMINI_API_KEY blank to use the built-in
# deterministic stub (no external calls).
GEMINI_API_KEY=
GEMINI_MODEL=gemini-2.5-flash
```

- [ ] **Step 3: Install**

Run from `backend/` (venv active): `pip install -r requirements.txt`
Expected: installs PyJWT and google-genai with no errors.

- [ ] **Step 4: Commit**

```bash
git add backend/requirements.txt backend/.env.example
git commit -m "build(backend): add PyJWT + google-genai; shared JWT_SECRET config"
```

---

### Task 4: Pydantic models

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
```

- [ ] **Step 2: Verify it imports**

Run from `backend/`: `python -c "from app import models; print(list(models.Session.model_fields))"`
Expected: prints Session field names including `aiAnalysis`. No import error.

- [ ] **Step 3: Commit**

```bash
git add backend/app/models.py
git commit -m "feat(models): add StudentCrm, Session, AiAnalysis, AnalyzeRequest"
```

---

## Phase B — Go-JWT validation

### Task 5: Decode-only auth module

**Files:**
- Create: `backend/app/auth.py`

- [ ] **Step 1: Write the module**

Create `backend/app/auth.py`:

```python
"""Validates the JWT issued by the NYAMPE Go service.

FastAPI does NOT mint tokens or store users — the Go service (proxied via
routers/gateway.py) owns authentication. This module only *verifies* the
incoming Bearer token: HS256, signed with the shared JWT_SECRET. The Go token
carries `user_id` and `role` claims (roles: employee | manager | instructor).
`manager` is the elevated/admin-equivalent role.
"""
import os

import jwt
from fastapi import Depends, HTTPException, status
from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer
from pydantic import BaseModel

_ALGORITHM = "HS256"
# Matches the Go service's dev fallback so local dev "just works" with no .env.
_DEFAULT_SECRET = "super-secret-key-default"
_bearer = HTTPBearer(auto_error=False)


class TokenUser(BaseModel):
    userId: int
    role: str


def _secret() -> str:
    return os.getenv("JWT_SECRET", _DEFAULT_SECRET)


def decode_token(token: str) -> TokenUser:
    try:
        payload = jwt.decode(token, _secret(), algorithms=[_ALGORITHM])
    except jwt.PyJWTError:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED, detail="Invalid or expired token"
        )
    return TokenUser(userId=int(payload.get("user_id", 0)), role=payload.get("role", ""))


def require_auth(creds: HTTPAuthorizationCredentials = Depends(_bearer)) -> TokenUser:
    """Any authenticated NYAMPE user."""
    if creds is None:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED, detail="Not authenticated"
        )
    return decode_token(creds.credentials)


def require_manager(user: TokenUser = Depends(require_auth)) -> TokenUser:
    """Manager-only endpoints (the admin-equivalent role)."""
    if user.role != "manager":
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN, detail="Manager role required"
        )
    return user
```

- [ ] **Step 2: Verify validation against a Go-shaped token**

Run from `backend/` (mints a token exactly like the Go service, then decodes it):
```bash
python -c "import jwt, time; from app.auth import decode_token; t=jwt.encode({'user_id':5,'role':'manager','exp':int(time.time())+3600},'super-secret-key-default',algorithm='HS256'); u=decode_token(t); print(u.userId, u.role)"
```
Expected: `5 manager`. A token signed with a different secret raises a 401 (HTTPException).

- [ ] **Step 3: Commit**

```bash
git add backend/app/auth.py
git commit -m "feat(auth): validate NYAMPE Go-issued JWT (decode-only deps)"
```

---

## Phase C — CRM, AI, Sessions

### Task 6: CRM router

**Files:**
- Create: `backend/app/routers/crm.py`

- [ ] **Step 1: Write the router**

Create `backend/app/routers/crm.py`:

```python
"""CRM students CRUD (admin tooling). Manager-only per the auth matrix."""
from fastapi import APIRouter, Depends, status
from pymysql.connections import Connection

from ..auth import require_manager
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
def list_students(db: Connection = Depends(get_db), _: object = Depends(require_manager)):
    with db.cursor() as cur:
        cur.execute(_SELECT)
        return [StudentCrm(**row) for row in cur.fetchall()]


@router.post("", response_model=StudentCrm, status_code=status.HTTP_201_CREATED)
def create_student(student: StudentCrm, db: Connection = Depends(get_db), _: object = Depends(require_manager)):
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
def update_student(student_id: int, student: StudentCrm, db: Connection = Depends(get_db), _: object = Depends(require_manager)):
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
def delete_student(student_id: int, db: Connection = Depends(get_db), _: object = Depends(require_manager)):
    with db.cursor() as cur:
        cur.execute("DELETE FROM students_crm WHERE id=%(id)s", {"id": student_id})
```

> Note the `%%Y-%%m-%%d` double-percent: PyMySQL treats `%` as a parameter marker, so literal percents in SQL must be escaped.

- [ ] **Step 2: Commit**

```bash
git add backend/app/routers/crm.py
git commit -m "feat(crm): add /api/crm/students CRUD (manager-only)"
```

---

### Task 7: Gemini 2.5 analyzer module

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

### Task 8: Sessions router (CRUD + analyze)

**Files:**
- Create: `backend/app/routers/sessions.py`

- [ ] **Step 1: Write the router**

Create `backend/app/routers/sessions.py`:

```python
"""Training sessions CRUD + AI analysis (admin tooling).

Reads/analyze require any authenticated user; writes require manager. The four
ai_* columns map to a nested AiAnalysis object on the wire (`aiAnalysis`, or
null when empty) to match the Angular Session interface. JSON columns
deserialize to Python lists via PyMySQL."""
import json

from fastapi import APIRouter, Depends, HTTPException, status
from pymysql.connections import Connection

from ..ai import analyze_session_notes
from ..auth import require_auth, require_manager
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
def create_session(session: Session, db: Connection = Depends(get_db), _: object = Depends(require_manager)):
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
def update_session(session_id: int, session: Session, db: Connection = Depends(get_db), _: object = Depends(require_manager)):
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
def delete_session(session_id: int, db: Connection = Depends(get_db), _: object = Depends(require_manager)):
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

### Task 9: Register the new routers

**Files:**
- Modify: `backend/app/main.py`

- [ ] **Step 1: Update imports and includes**

In `backend/app/main.py`, change:

```python
from .routers import courses, mechanisms, rag, gateway
```
to:
```python
from .routers import courses, crm, mechanisms, rag, sessions, gateway
```

And after `app.include_router(gateway.router)` (or alongside the other includes), add:

```python
app.include_router(crm.router)
app.include_router(sessions.router)
```

- [ ] **Step 2: Verify the app imports**

Run from `backend/`: `python -c "from app.main import app; print([r.path for r in app.routes if 'crm' in r.path or 'sessions' in r.path])"`
Expected: lists `/api/crm/students` and `/api/sessions` (+ `/api/sessions/{session_id}` and `/analyze`). No import error.

- [ ] **Step 3: Commit**

```bash
git add backend/app/main.py
git commit -m "feat(api): register crm and sessions routers"
```

---

### Task 10: Backend endpoint verification

**Files:** none (verification only)

- [ ] **Step 1: Start the server**

From `backend/` with MySQL running and schema+seed applied:
`uvicorn app.main:app --reload --port 8080`
Expected: starts with no import errors; `/docs` lists `crm` and `sessions` routes.

- [ ] **Step 2: Mint a test token (same secret/claims as Go)**

```bash
python -c "import jwt, time; print(jwt.encode({'user_id':1,'role':'manager','exp':int(time.time())+3600},'super-secret-key-default',algorithm='HS256'))"
```
Copy the printed token.

- [ ] **Step 3: Hit protected endpoints**

```bash
TOKEN=...   # paste the token
curl -s -o /dev/null -w "no-token: %{http_code}\n" http://localhost:8080/api/crm/students
curl -s -w "\n" http://localhost:8080/api/crm/students -H "Authorization: Bearer $TOKEN" | head
curl -s -w "\n" http://localhost:8080/api/sessions -H "Authorization: Bearer $TOKEN" | head
```
Expected: `no-token: 401`; with the manager token, 6 CRM students and 3 sessions (session id 2 has nested `aiAnalysis`, id 1 has `aiAnalysis: null`).

- [ ] **Step 4: Analyze a session (stub path)**

```bash
curl -s -X POST http://localhost:8080/api/sessions/1/analyze -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "{\"rawNotes\":\"Siswa latihan parkir.\"}" | head
```
Expected: returns session 1 with `status: "completed"` and a populated `aiAnalysis`. Re-querying `/api/sessions` confirms it persisted.

- [ ] **Step 5: Commit any fixes**

```bash
git commit -am "fix(backend): address issues found in endpoint verification" --allow-empty
```

---

## Phase D — Frontend wiring

### Task 11: Wire session analysis to the real endpoint

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

> `Observable`, `of`, `catchError`, and `Session` are already imported in this file (used by the existing read methods) — no new imports needed.

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

### Task 12: Full-stack smoke verification

**Files:** none (verification only)

- [ ] **Step 1: Run servers**

Backend: `uvicorn app.main:app --port 8080` (MySQL up, seed applied). Frontend: `npm start`.
For real login, the NYAMPE Go service must be running on `GO_BACKEND_URL` (default `http://localhost:8090`) with the **same `JWT_SECRET`**.

- [ ] **Step 2: Verify analysis flow (manager login)**

Log in (via NYAMPE) as a manager, open the Sesi screen, open a `scheduled` session, enter notes, submit.
Expected: spinner, then the session flips to `completed` with a populated analysis (backend stub, or real Gemini if a key is set in `backend/.env`). Network tab shows `POST /api/sessions/{id}/analyze` carried the `Authorization: Bearer` header (added by the existing interceptor) and returned 200.

- [ ] **Step 3: Verify offline fallback**

Stop the FastAPI backend; submit notes again.
Expected: `analyzeSession` falls back to the local mock analysis; the UI still completes the session.

- [ ] **Step 4: Final commit**

```bash
git commit -am "test: full-stack smoke verification of CRM, sessions, AI analysis" --allow-empty
```

---

## Self-Review Notes

- **Spec coverage:** Part 1 (CRM/Sessions backend) → Tasks 1,2,4,6,8,9. Part 2 (Gemini 2.5) → Tasks 7,8,11. Part 3 REVISED (validate Go JWT) → Tasks 3,5 + per-router deps (6,8). Frontend → Task 11. Verification → Tasks 10,12.
- **Dropped vs. original** (now done by NYAMPE merge): `users` table/seed, bcrypt, FastAPI token minting, login router, instructors.py protection, frontend AuthService/interceptor/login rewrites.
- **Auth model alignment:** gating uses `manager` (Go's elevated role), not `admin` (which no longer exists). CRM → `require_manager`; session reads/analyze → `require_auth`; session writes → `require_manager`.
- **Type consistency:** `StudentCrm`/`Session`/`AiAnalysis` field names match between `models.py` (Task 4), SQL aliases (Tasks 6/8), and the unchanged Angular interfaces. `require_auth`/`require_manager` signatures match between `auth.py` (Task 5) and the routers (Tasks 6,8).
- **PyMySQL `%%` escaping** applied in the CRM and sessions `DATE_FORMAT` SELECTs.
- **Integration risks:** (a) FastAPI and Go must share `JWT_SECRET` — Task 3 sets the matching dev default and documents it. (b) Go `.env.example` sets `PORT=8080` which collides with FastAPI's required 8080; the gateway expects Go on 8090 (`GO_BACKEND_URL`) — a NYAMPE-side config concern, noted but out of scope here. (c) Non-manager users hitting `/api/crm/*` get 403 → ApiService mock fallback keeps the screen functional.
```
