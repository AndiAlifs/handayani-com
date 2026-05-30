"""Course catalog CRUD (PRD Epic 2)."""
from fastapi import APIRouter, Depends, status
from pymysql.connections import Connection

from ..database import get_db
from ..models import Course

router = APIRouter(prefix="/api/courses", tags=["courses"])

_SELECT = (
    "SELECT id, category, program_type AS programType, specifics, duration, "
    "price, registration_fee AS registrationFee, remarks "
    "FROM courses ORDER BY category, id"
)


@router.get("", response_model=list[Course])
def list_courses(db: Connection = Depends(get_db)):
    with db.cursor() as cur:
        cur.execute(_SELECT)
        return [Course(**row) for row in cur.fetchall()]


@router.post("", response_model=Course, status_code=status.HTTP_201_CREATED)
def create_course(course: Course, db: Connection = Depends(get_db)):
    with db.cursor() as cur:
        cur.execute(
            "INSERT INTO courses "
            "(category, program_type, specifics, duration, price, registration_fee, remarks) "
            "VALUES (%(category)s, %(programType)s, %(specifics)s, %(duration)s, "
            "%(price)s, %(registrationFee)s, %(remarks)s)",
            course.model_dump(exclude={"id"}),
        )
        course.id = cur.lastrowid
    return course


@router.put("/{course_id}", response_model=Course)
def update_course(course_id: int, course: Course, db: Connection = Depends(get_db)):
    course.id = course_id
    with db.cursor() as cur:
        cur.execute(
            "UPDATE courses SET category=%(category)s, program_type=%(programType)s, "
            "specifics=%(specifics)s, duration=%(duration)s, price=%(price)s, "
            "registration_fee=%(registrationFee)s, remarks=%(remarks)s WHERE id=%(id)s",
            course.model_dump(),
        )
    return course


@router.delete("/{course_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_course(course_id: int, db: Connection = Depends(get_db)):
    with db.cursor() as cur:
        cur.execute("DELETE FROM courses WHERE id=%(id)s", {"id": course_id})
