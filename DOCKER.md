# Running the full stack with Docker

One command brings up MySQL, the Go gateway, the FastAPI AI service, and the
Angular SPA:

```bash
docker compose up --build
```

| Service | URL / port            | Notes                                                          |
|---------|-----------------------|----------------------------------------------------------------|
| `web`   | http://localhost:4200 | Angular SPA (nginx-served production build)                    |
| `core`  | http://localhost:8080 | NYAMPE Go gateway — auth + attendance + content CRUD + AI proxy |
| `ai`    | (internal `:8081`)    | FastAPI AI service — `/analyze` + RAG; not published by default |
| `db`    | localhost:3306        | MySQL 8, single `handayani` database                           |

Open **http://localhost:4200** and log in with a NYAMPE-seeded account (created by
the Go service's seeder on first boot).

## How it fits together

- The browser calls the API at `http://localhost:8080` (the Go gateway, baked into
  the Angular build) and the SPA at `http://localhost:4200`.
- `core` serves auth, attendance, and CRUD for courses/mechanisms/CRM/sessions
  natively, and reverse-proxies the AI endpoints (`/api/sessions/{id}/analyze`,
  `/api/rag/knowledge-sync`) to `ai` via `AI_SERVICE_URL=http://ai:8081`.
- `ai` reads courses/mechanisms and the `sessions.ai_*` columns directly, and
  fetches the instructor list back from `core` (`GO_BACKEND_URL=http://core:8080`).
- Both backends share one MySQL database `handayani`. The content tables
  (`courses`, `mechanisms`, `students_crm`, `sessions`) are loaded from
  `backend/schema.sql` + `backend/seed.sql` on the **first** boot (empty volume);
  the Go service auto-migrates and seeds its own tables.
- `core` and `ai` share the same `JWT_SECRET`, so the AI service can validate the
  token `core` forwards on `/analyze`.

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
docker compose logs -f core      # tail one service (or `ai`, `db`, `web`)
docker compose down              # stop (keeps the db volume)
docker compose down -v           # stop and wipe the database (re-seeds next up)
```

## Notes

- Re-seeding: `schema.sql`/`seed.sql` run only on an empty data volume. To reload
  them after changes, `docker compose down -v` then `up` again.
- The SPA build bakes `apiBaseUrl=http://localhost:8080`. If you host the API
  elsewhere, change `frontend/src/environments/environment.ts` and rebuild the
  `web` image.
