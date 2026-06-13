"""CRM students CRUD (admin tooling). Manager-only per the auth matrix."""
from fastapi import APIRouter, Depends, status
from pymysql.connections import Connection

from ..auth import require_manager
from ..database import get_db
from ..models import StudentCrm

router = APIRouter(prefix="/api/crm/students", tags=["crm"])

_SELECT = (
    "SELECT id, name, phone, course_id AS courseId, course_name AS courseName, "
    "status, progress_score AS progressScore, notes, "
    "DATE_FORMAT(created_at, '%%Y-%%m-%%d') AS createdAt "
    "FROM students_crm ORDER BY created_at DESC, id DESC"
)


@router.get("", response_model=list[StudentCrm])
def list_students(db: Connection = Depends(get_db), _: object = Depends(require_manager)):
    with db.cursor() as cur:
        cur.execute(_SELECT)
        return [StudentCrm(**row) for row in cur.fetchall()]


@router.post("", response_model=StudentCrm, status_code=status.HTTP_201_CREATED)
def create_student(student: StudentCrm, db: Connection = Depends(get_db), _: object = Depends(require_manager)):
    with db.cursor() as cur:
        cur.execute(
            "INSERT INTO students_crm "
            "(name, phone, course_id, course_name, status, progress_score, notes) "
            "VALUES (%(name)s, %(phone)s, %(courseId)s, %(courseName)s, "
            "%(status)s, %(progressScore)s, %(notes)s)",
            student.model_dump(exclude={"id", "createdAt"}),
        )
        student.id = cur.lastrowid
    return student


@router.put("/{student_id}", response_model=StudentCrm)
def update_student(student_id: int, student: StudentCrm, db: Connection = Depends(get_db), _: object = Depends(require_manager)):
    student.id = student_id
    with db.cursor() as cur:
        cur.execute(
            "UPDATE students_crm SET name=%(name)s, phone=%(phone)s, "
            "course_id=%(courseId)s, course_name=%(courseName)s, status=%(status)s, "
            "progress_score=%(progressScore)s, notes=%(notes)s WHERE id=%(id)s",
            student.model_dump(exclude={"createdAt"}),
        )
    return student


@router.delete("/{student_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_student(student_id: int, db: Connection = Depends(get_db), _: object = Depends(require_manager)):
    with db.cursor() as cur:
        cur.execute("DELETE FROM students_crm WHERE id=%(id)s", {"id": student_id})
