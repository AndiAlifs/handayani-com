"""Weekly schedule matrix helpers (PRD Epic 3).

The `schedules` table is kept sparse — only non-default slots are stored.
On read, the full Senin–Minggu x time-slot matrix is reconstructed, defaulting
empty slots to "Tersedia".
"""
from pymysql.connections import Connection

from .models import ScheduleSlot

# Must match the Angular grid (instruktur + instructor-schedule components).
DAYS = ["Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu", "Minggu"]
TIME_SLOTS = ["09.00 - 12.00", "13.00 - 15.00", "15.00 - 17.00"]
DEFAULT_STATUS = "Tersedia"


def load_schedule(db: Connection, instructor_id: int) -> dict[str, str]:
    with db.cursor() as cur:
        cur.execute(
            "SELECT day, time_slot, status FROM schedules WHERE instructor_id=%(id)s",
            {"id": instructor_id},
        )
        rows = cur.fetchall()
    return {f"{r['day']}|{r['time_slot']}": r["status"] for r in rows}


def full_matrix(stored: dict[str, str]) -> list[ScheduleSlot]:
    slots: list[ScheduleSlot] = []
    for day in DAYS:
        for ts in TIME_SLOTS:
            status = stored.get(f"{day}|{ts}", DEFAULT_STATUS)
            slots.append(ScheduleSlot(day=day, timeSlot=ts, status=status))
    return slots


# Statuses safe to expose publicly. A booked slot's status holds the student's
# name (set from the admin dashboard / CRM), so anything else is a booking and
# must be masked before it leaves the server on the public-facing read.
PUBLIC_STATUSES = {DEFAULT_STATUS, "Libur"}
BOOKED_PUBLIC_LABEL = "Terisi"


def public_matrix(stored: dict[str, str]) -> list[ScheduleSlot]:
    """Full matrix with any booking detail (e.g. a customer name) masked to 'Terisi'."""
    return [
        slot if slot.status in PUBLIC_STATUSES
        else ScheduleSlot(day=slot.day, timeSlot=slot.timeSlot, status=BOOKED_PUBLIC_LABEL)
        for slot in full_matrix(stored)
    ]


def persist_schedule(db: Connection, instructor_id: int, slots: list[ScheduleSlot]) -> None:
    """Replace all schedule rows for one instructor (within the caller's transaction)."""
    with db.cursor() as cur:
        cur.execute("DELETE FROM schedules WHERE instructor_id=%(id)s", {"id": instructor_id})
        payload = [
            (instructor_id, s.day, s.timeSlot, s.status)
            for s in slots
            if s.status and s.status != DEFAULT_STATUS
        ]
        if payload:
            cur.executemany(
                "INSERT INTO schedules (instructor_id, day, time_slot, status) "
                "VALUES (%s, %s, %s, %s)",
                payload,
            )
