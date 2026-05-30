"""Instructor profile CRUD + weekly schedule matrix (PRD Epic 3)."""
from fastapi import APIRouter, Depends, status
from pymysql.connections import Connection

from ..database import get_db
from ..models import Instructor, ScheduleSlot
from ..schedule import full_matrix, load_schedule, persist_schedule

router = APIRouter(prefix="/api/instructors", tags=["instructors"])

_SELECT_ALL = "SELECT id, name, gender, age, vehicle, transmission FROM instructors ORDER BY id"


@router.get("/schedule", response_model=list[Instructor])
def list_instructors_with_schedule(db: Connection = Depends(get_db)):
    """All instructors, each with their full reconstructed weekly matrix."""
    with db.cursor() as cur:
        cur.execute(_SELECT_ALL)
        rows = cur.fetchall()
    instructors = []
    for row in rows:
        inst = Instructor(**row)
        inst.schedule = full_matrix(load_schedule(db, inst.id))
        instructors.append(inst)
    return instructors


@router.post("", response_model=Instructor, status_code=status.HTTP_201_CREATED)
def create_instructor(instructor: Instructor, db: Connection = Depends(get_db)):
    with db.cursor() as cur:
        cur.execute(
            "INSERT INTO instructors (name, gender, age, vehicle, transmission) "
            "VALUES (%(name)s, %(gender)s, %(age)s, %(vehicle)s, %(transmission)s)",
            instructor.model_dump(exclude={"id", "schedule"}),
        )
        instructor.id = cur.lastrowid
    # Persist any non-default slots that arrived with the new profile.
    persist_schedule(db, instructor.id, instructor.schedule)
    instructor.schedule = full_matrix(load_schedule(db, instructor.id))
    return instructor


@router.put("/{instructor_id}", response_model=Instructor)
def update_instructor(instructor_id: int, instructor: Instructor, db: Connection = Depends(get_db)):
    instructor.id = instructor_id
    with db.cursor() as cur:
        cur.execute(
            "UPDATE instructors SET name=%(name)s, gender=%(gender)s, age=%(age)s, "
            "vehicle=%(vehicle)s, transmission=%(transmission)s WHERE id=%(id)s",
            instructor.model_dump(exclude={"schedule"}),
        )
    instructor.schedule = full_matrix(load_schedule(db, instructor_id))
    return instructor


@router.delete("/{instructor_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_instructor(instructor_id: int, db: Connection = Depends(get_db)):
    with db.cursor() as cur:
        cur.execute("DELETE FROM instructors WHERE id=%(id)s", {"id": instructor_id})


@router.put("/{instructor_id}/schedule", response_model=list[ScheduleSlot])
def update_schedule(instructor_id: int, slots: list[ScheduleSlot], db: Connection = Depends(get_db)):
    """Replace the whole weekly matrix for one instructor (AC2/AC3)."""
    persist_schedule(db, instructor_id, slots)
    return full_matrix(load_schedule(db, instructor_id))
