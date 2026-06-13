"""Pydantic schemas.

Field names are camelCase to match the Angular interfaces in
src/app/core/models, so payloads round-trip without transformation.
"""
from typing import Optional
from pydantic import BaseModel


class Course(BaseModel):
    id: Optional[int] = None
    category: str
    programType: str
    specifics: str
    duration: str
    price: int
    registrationFee: int
    remarks: str = ""


class ScheduleSlot(BaseModel):
    day: str
    timeSlot: str
    status: str = "Tersedia"


class Instructor(BaseModel):
    id: Optional[int] = None
    name: str
    gender: str
    age: int
    vehicle: str
    transmission: str
    schedule: list[ScheduleSlot] = []


class Mechanism(BaseModel):
    id: Optional[int] = None
    requirementName: str
    issuingBody: str
    cost: int
    notes: str = ""


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
