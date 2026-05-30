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
