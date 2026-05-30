"""SIM mechanism steps CRUD (PRD Epic 4)."""
from fastapi import APIRouter, Depends, status
from pymysql.connections import Connection

from ..database import get_db
from ..models import Mechanism

router = APIRouter(prefix="/api/mechanisms", tags=["mechanisms"])

_SELECT = (
    "SELECT id, requirement_name AS requirementName, issuing_body AS issuingBody, "
    "cost, notes FROM mechanisms ORDER BY sort_order, id"
)


@router.get("", response_model=list[Mechanism])
def list_mechanisms(db: Connection = Depends(get_db)):
    with db.cursor() as cur:
        cur.execute(_SELECT)
        return [Mechanism(**row) for row in cur.fetchall()]


@router.post("", response_model=Mechanism, status_code=status.HTTP_201_CREATED)
def create_mechanism(mechanism: Mechanism, db: Connection = Depends(get_db)):
    with db.cursor() as cur:
        cur.execute(
            "INSERT INTO mechanisms (requirement_name, issuing_body, cost, notes, sort_order) "
            "VALUES (%(requirementName)s, %(issuingBody)s, %(cost)s, %(notes)s, "
            "(SELECT COALESCE(MAX(sort_order), 0) + 1 FROM mechanisms m))",
            mechanism.model_dump(exclude={"id"}),
        )
        mechanism.id = cur.lastrowid
    return mechanism


@router.put("/{mechanism_id}", response_model=Mechanism)
def update_mechanism(mechanism_id: int, mechanism: Mechanism, db: Connection = Depends(get_db)):
    mechanism.id = mechanism_id
    with db.cursor() as cur:
        cur.execute(
            "UPDATE mechanisms SET requirement_name=%(requirementName)s, "
            "issuing_body=%(issuingBody)s, cost=%(cost)s, notes=%(notes)s WHERE id=%(id)s",
            mechanism.model_dump(),
        )
    return mechanism


@router.delete("/{mechanism_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_mechanism(mechanism_id: int, db: Connection = Depends(get_db)):
    with db.cursor() as cur:
        cur.execute("DELETE FROM mechanisms WHERE id=%(id)s", {"id": mechanism_id})
