"""Training session AI analysis (internal AI service).

Session CRUD (list/create/update/delete) now lives in the Go gateway; this
service only runs the Gemini analysis and persists the result. Analyze requires
any authenticated user. The four ai_* columns map to a nested AiAnalysis object
on the wire (`aiAnalysis`, or null when empty) to match the Angular Session
interface. JSON columns deserialize to Python lists via PyMySQL."""
import json
from typing import Optional

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


def _fetch_one(db: Connection, session_id: int) -> Optional[dict]:
    with db.cursor() as cur:
        cur.execute(_SELECT + " WHERE id=%s", (session_id,))
        return cur.fetchone()


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
