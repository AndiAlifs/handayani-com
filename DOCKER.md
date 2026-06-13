# Running the full stack with Docker

One command brings up MySQL, the Go attendance backend, the FastAPI backend, and
the Angular SPA:

```bash
docker compose up --build
```

| Service      | URL / port              | Notes                                              |
|--------------|-------------------------|----------------------------------------------------|
| `web`        | http://localhost:4200   | Angular SPA (nginx-served production build)         |
| `api`        | http://localhost:8080   | FastAPI — CRM/Sessions/RAG + gateway to Go; `/docs` |
| `attendance` | http://localhost:8090   | NYAMPE Go backend (auth + attendance)               |
| `db`         | localhost:3306          | MySQL 8, single `handayani` database                |

Open **http://localhost:4200** and log in with a NYAMPE-seeded account (created by
the Go service's seeder on first boot).

## How it fits together

- The browser calls the API at `http://localhost:8080` (baked into the Angular
  build) and the SPA at `http://localhost:4200`.
- `api` proxies `/api/auth|attendance|admin|instructor` to `attendance` via
  `GO_BACKEND_URL=http://attendance:8090`, and serves CRM/Sessions/RAG itself.
- Both backends share one MySQL database `handayani`. FastAPI's tables
  (`courses`, `mechanisms`, `students_crm`, `sessions`) are loaded from
  `backend/schema.sql` + `backend/seed.sql` on the **first** boot (empty volume);
  the Go service auto-migrates and seeds its own tables.
- `api` and `attendance` share the same `JWT_SECRET`, so FastAPI can validate the
  token the Go service issues.

## Configuration

Defaults work out of the box. Override via environment before `up`:

```bash
# Real Gemini 2.5 session analysis (otherwise a deterministic stub is used):
export GEMINI_API_KEY=your_key_here
# Optionally change the model (default gemini-2.5-flash) or signing secret:
export GEMINI_MODEL=gemini-2.5-pro
export JWT_SECRET=some-long-random-string

docker compose up --build
```

## Common commands

```bash
docker compose up --build -d     # run detached
docker compose logs -f api       # tail one service
docker compose down              # stop (keeps the db volume)
docker compose down -v           # stop and wipe the database (re-seeds next up)
```

## Notes

- Re-seeding: `schema.sql`/`seed.sql` run only on an empty data volume. To reload
  them after changes, `docker compose down -v` then `up` again.
- The SPA build bakes `apiBaseUrl=http://localhost:8080`. If you host the API
  elsewhere, change `frontend/src/environments/environment.ts` and rebuild the
  `web` image.
