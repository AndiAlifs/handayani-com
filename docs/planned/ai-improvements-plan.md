# AI / RAG Improvements Roadmap

**Date:** 2026-06-21 · **Status:** In Progress · **Owner:** Andi Alifsyah

This document tracks a sequenced set of AI and observability improvements to the FastAPI
backend. The work turns four high-value capabilities into a dependency-aware roadmap. Each step
ships and demos independently.

**Recommended order: 1 → 2 → 3 → 4.**

## Background — current state

- **Single LLM call:** `analyze_session_notes()` in `backend/app/ai.py`, called only from
  `POST /api/sessions/{id}/analyze` (`backend/app/routers/sessions.py`). Gemini 2.5 Flash via
  `google-genai`, with a clean graceful-degradation stub when `GEMINI_API_KEY` is empty.
- **RAG:** `GET /api/rag/knowledge-sync` (`backend/app/routers/rag.py`) returns a single markdown
  blob — no citations, no source IDs, no freshness markers.
- **CRM-insights data already exists:** `students_crm` + `sessions` (with `ai_*` columns) in
  `backend/schema.sql`.
- **No scheduler anywhere** — `backend/app/main.py` has no lifespan/startup events; APScheduler is
  not a dependency.
- **Gateway proxy** to the Go/NYAMPE service already exists: `backend/app/routers/gateway.py`
  (httpx, `GO_BACKEND_URL`).
- **Latent bug:** `rag.py` still does `SELECT ... FROM instructors`, but that table was moved to the
  Go service and dropped from `schema.sql`. The instructor section is currently broken.

## Decisions

- **Instructor data in RAG:** fetch live via the Go gateway (not drop the section).
- **Scope:** all four steps, including the Telegram bot.
- **Step 3 frontend:** include the Angular dashboard card.
- **Step 3 scheduling:** APScheduler in-process nightly job + a manual `POST /run` endpoint.

## Step 1 — Source citations in RAG (≈ ½ day) · **Status: Executed** ✅

- File: `backend/app/routers/rag.py`.
- Tag each rendered KB chunk with a stable source ID from the DB row, e.g. `[#kursus-3]`,
  `[#mekanisme-5]`, and append `updated_at` where available (`courses` has it).
- Add a "Sumber / Source" legend section mapping each tag → table + id.
- Fix the instructor section: pull instructor + schedule data through the existing gateway
  (`backend/app/routers/gateway.py` / `GO_BACKEND_URL`) instead of the dropped local table.
- Optional: `GET /api/rag/knowledge-sync.json` returning structured chunks
  `[{sourceId, type, title, body, updatedAt}]` for easier bot citation.

## Step 2 — Langfuse tracing on the existing call (1–2 hrs) · **Status: Planned** 🔲

- Add `langfuse` to `backend/requirements.txt`; add `LANGFUSE_PUBLIC_KEY` / `LANGFUSE_SECRET_KEY` /
  `LANGFUSE_HOST` to `backend/.env.example` and `compose.yml`.
- Wrap the Gemini call in `backend/app/ai.py` in a Langfuse span (prompt, model, output, latency,
  token usage). No-op cleanly when keys are absent, mirroring the existing stub pattern.

## Step 3 — Overnight multi-agent CRM-insights workflow (2–3 days) · **Status: Planned** 🔲

- New module `backend/app/agents/crm_insights.py`: three Gemini calls —
  - **Planner:** reads aggregate stats over `students_crm` + `sessions` → decides report focus.
  - **Generator:** writes the weekly markdown report against that plan.
  - **Evaluator:** scores the draft; if thin, returns critique → regenerate once (capped).
  - Each step is a Langfuse span under one trace (free, given Step 2).
- New table `crm_insight_reports` (id, generated_at, content_md, plan_json, eval_score, eval_notes,
  regenerated) appended to `backend/schema.sql`.
- Scheduler: APScheduler in a FastAPI `lifespan` in `backend/app/main.py`, nightly job. Also add
  `POST /api/crm/insights/run` (manager-only, for live demo) and `GET /api/crm/insights/latest`.
- Graceful degradation: key-less → deterministic stub report, consistent with `ai.py`.
- Frontend: "Laporan Mingguan AI" card on the CRM dashboard + "Generate sekarang" button, with mock
  fallback per the `ApiService` pattern.

## Step 4 — Thin Telegram bot over RAG (optional, ~1 day) · **Status: Planned** 🔲

- Standalone service/poller that calls `knowledge-sync` (citation-aware after Step 1) → Gemini →
  replies with source links. Separate service in `compose.yml`; not entangled with the API.

## Verification

- **Step 1:** hit `GET /api/rag/knowledge-sync`, confirm source tags + legend present and instructor
  section populated via gateway; check `GET /docs`.
- **Step 2:** run `analyze` with Langfuse keys set, confirm a trace with latency/tokens appears.
- **Step 3:** `POST /api/crm/insights/run`, confirm a report row is written, evaluator score
  recorded, regenerate fires on a thin draft; confirm the trace shows all three agent spans; confirm
  the frontend card renders.
- **Step 4:** send a Telegram message, confirm a cited answer.
