"""YPA Handayani Knowledge Base API (FastAPI + MySQL).

Implements PRD Epics 2–5. The Angular ApiService points at
http://localhost:8080 and consumes these endpoints, falling back to bundled
mock data when the API is unavailable.
"""
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from .routers import courses, crm, mechanisms, rag, sessions, gateway

app = FastAPI(
    title="YPA Handayani Knowledge Base API",
    version="1.0.0",
    description="Courses, instructors & schedules, SIM mechanisms, and RAG knowledge sync.",
)

# Open CORS so the Angular dev server (localhost:4200) and the RAG bot poller
# can call the API from another origin.
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(courses.router)
app.include_router(mechanisms.router)
app.include_router(rag.router)
app.include_router(gateway.router)
app.include_router(crm.router)
app.include_router(sessions.router)


@app.get("/api/health", tags=["health"])
def health():
    return {"status": "ok"}
