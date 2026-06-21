"""YPA Handayani internal AI service (FastAPI + MySQL).

A slim internal service behind the Go API gateway. It serves only:
  - POST /api/sessions/{id}/analyze   (Gemini session analysis)
  - GET  /api/rag/knowledge-sync[.json]   (RAG knowledge base for the chat bot)
  - GET  /api/health

The Go service (port 8080) is the single public front door and reverse-proxies
these endpoints here (port 8081). The browser never reaches this service
directly, so no CORS middleware is needed — Go terminates the browser request.
Courses / mechanisms / CRM / sessions CRUD moved to the Go gateway.
"""
from fastapi import FastAPI

from .routers import rag, sessions

app = FastAPI(
    title="YPA Handayani AI Service",
    version="2.0.0",
    description="Gemini session analysis + RAG knowledge sync (internal, behind the Go gateway).",
)

app.include_router(rag.router)
app.include_router(sessions.router)


@app.get("/api/health", tags=["health"])
def health():
    return {"status": "ok"}
