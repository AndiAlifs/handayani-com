"""Pydantic schemas for the AI service.

Field names are camelCase to match the Angular Session interface so payloads
round-trip without transformation. Only the models used by the /analyze
endpoint (and ai.py) remain; the course / mechanism / CRM / instructor models
moved to the Go gateway along with their endpoints.
"""
from typing import Optional
from pydantic import BaseModel


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
