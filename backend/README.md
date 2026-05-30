# YPA Handayani — Knowledge Base API (Python / FastAPI)

REST backend for the YPA Handayani Knowledge Base, implementing PRD Epics 2–5.
Stack: Python (FastAPI) + MySQL (SQLAlchemy Core + PyMySQL).

> Note: the PRD names Golang, but this project implements the backend in Python
> per the team's decision. The REST contract is unchanged.

## Prerequisites

- Python 3.10+
- MySQL 8+

## Setup

```bash
# 1. Create schema + seed data
mysql -u root -p < schema.sql
mysql -u root -p < seed.sql

# 2. Install dependencies (use a virtualenv)
python -m venv .venv
.venv\Scripts\activate        # Windows PowerShell
# source .venv/bin/activate    # macOS / Linux
pip install -r requirements.txt

# 3. Configure connection (defaults target local root / handayani)
copy .env.example .env         # then edit, or export the vars

# 4. Run (port 8080 to match the Angular environment config)
uvicorn app.main:app --reload --port 8080
```

Interactive API docs are available at `http://localhost:8080/docs` (requires
Python 3.10+; the OpenAPI generator trips over a typing bug in Python 3.9.0 —
the REST endpoints themselves work on 3.9 regardless).
The Angular `ApiService` already points at `http://localhost:8080` in
`src/environments/environment.ts`, and CORS is open for `localhost:4200`.

## Endpoints

| Method | Path | Epic | Purpose |
|--------|------|------|---------|
| GET | `/api/health` | — | Liveness check |
| GET | `/api/courses` | 2 | List courses |
| POST | `/api/courses` | 2 | Create course |
| PUT | `/api/courses/{id}` | 2 | Update course |
| DELETE | `/api/courses/{id}` | 2 | Delete course |
| GET | `/api/instructors/schedule` | 3 | List instructors with full weekly matrix |
| POST | `/api/instructors` | 3 | Create instructor |
| PUT | `/api/instructors/{id}` | 3 | Update instructor profile |
| DELETE | `/api/instructors/{id}` | 3 | Delete instructor |
| PUT | `/api/instructors/{id}/schedule` | 3 | Replace the weekly schedule matrix |
| GET | `/api/mechanisms` | 4 | List SIM mechanism steps |
| POST | `/api/mechanisms` | 4 | Create step |
| PUT | `/api/mechanisms/{id}` | 4 | Update step |
| DELETE | `/api/mechanisms/{id}` | 4 | Delete step |
| GET | `/api/rag/knowledge-sync` | 5 | Flattened Markdown of the whole KB for the RAG bot |

## Project layout

```
backend/
  app/
    main.py            FastAPI app, CORS, router registration
    database.py        SQLAlchemy engine + get_db dependency
    models.py          Pydantic schemas (camelCase, matches Angular)
    schedule.py        Weekly matrix helpers (sparse storage + reconstruction)
    routers/
      courses.py
      instructors.py
      mechanisms.py
      rag.py
  schema.sql
  seed.sql
  requirements.txt
```

## Schedule storage

The `schedules` table is kept sparse — only non-`Tersedia` slots are persisted.
On read, `GET /api/instructors/schedule` reconstructs the complete
Senin–Minggu × {09.00–12.00, 13.00–15.00, 15.00–17.00} matrix, defaulting empty
slots to `Tersedia`. `PUT /api/instructors/{id}/schedule` replaces the whole
matrix inside a transaction.

## RAG knowledge sync

`GET /api/rag/knowledge-sync` returns `text/markdown` (not JSON): a single
descriptive document covering courses, instructor availability, and SIM costs,
plus the official WhatsApp contacts. The bot polls this on an interval — there
are no webhooks (PRD §4.2).
