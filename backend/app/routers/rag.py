"""RAG knowledge-sync endpoint (PRD Epic 5).

Returns the entire knowledge base flattened into one descriptive Markdown
document optimised for LLM semantic search. The bot polls this on an interval
(no webhooks, per PRD 4.2).
"""
from fastapi import APIRouter, Depends
from fastapi.responses import PlainTextResponse
from pymysql.connections import Connection

from ..database import get_db
from ..schedule import DEFAULT_STATUS, full_matrix, load_schedule

router = APIRouter(prefix="/api/rag", tags=["rag"])


def rupiah(value: int) -> str:
    if value == 0:
        return "Gratis"
    return "Rp" + f"{value:,}".replace(",", ".")


def _courses_section(db: Connection) -> str:
    with db.cursor() as cur:
        cur.execute(
            "SELECT category, program_type, specifics, duration, price, "
            "registration_fee, remarks FROM courses ORDER BY category, id"
        )
        rows = cur.fetchall()
    lines = ["## Katalog Kursus & Harga", ""]
    for r in rows:
        lines.append(
            f"- Kursus {r['category']} ({r['program_type']}) — {r['specifics']}. "
            f"Durasi: {r['duration']}. Biaya kursus: {rupiah(r['price'])}. "
            f"Biaya pendaftaran: {rupiah(r['registration_fee'])}. "
            f"Keterangan: {r['remarks'] or '-'}."
        )
    lines.append("")
    return "\n".join(lines)


def _instructors_section(db: Connection) -> str:
    with db.cursor() as cur:
        cur.execute("SELECT id, name, gender, age, vehicle, transmission FROM instructors ORDER BY id")
        rows = cur.fetchall()
    lines = ["## Instruktur & Jadwal Mingguan", ""]
    for r in rows:
        lines.append(f"### {r['name']}")
        lines.append(
            f"Jenis kelamin: {r['gender']}. Usia: {r['age']} tahun. "
            f"Kendaraan: {r['vehicle']}. Transmisi: {r['transmission']}."
        )
        booked, holiday = [], []
        for slot in full_matrix(load_schedule(db, r["id"])):
            if slot.status == DEFAULT_STATUS:
                continue
            if slot.status == "Libur":
                holiday.append(f"{slot.day} {slot.timeSlot}")
            else:
                booked.append(f"{slot.day} {slot.timeSlot} ({slot.status})")
        if booked:
            lines.append("Slot terisi: " + "; ".join(booked) + ".")
        if holiday:
            lines.append("Libur: " + "; ".join(holiday) + ".")
        lines.append(
            "Slot lain pada Senin–Sabtu (09.00–12.00, 13.00–15.00, 15.00–17.00) "
            "tersedia untuk booking."
        )
        lines.append("")
    return "\n".join(lines)


def _mechanisms_section(db: Connection) -> str:
    with db.cursor() as cur:
        cur.execute(
            "SELECT requirement_name, issuing_body, cost, notes "
            "FROM mechanisms ORDER BY sort_order, id"
        )
        rows = cur.fetchall()
    lines = ["## Mekanisme & Biaya Pembuatan SIM A", ""]
    total = 0
    for r in rows:
        total += r["cost"]
        lines.append(
            f"- {r['requirement_name']} — diterbitkan oleh {r['issuing_body']}. "
            f"Biaya: {rupiah(r['cost'])}. Catatan: {r['notes'] or '-'}."
        )
    lines.append("")
    lines.append(f"Perkiraan total biaya administrasi SIM: {rupiah(total)}.")
    lines.append("")
    return "\n".join(lines)


@router.get("/knowledge-sync", response_class=PlainTextResponse)
def knowledge_sync(db: Connection = Depends(get_db)):
    header = (
        "# Basis Pengetahuan YPA Handayani\n\n"
        "Dokumen ini berisi seluruh informasi resmi YPA Handayani: katalog kursus & harga, "
        "profil instruktur beserta jadwal mingguan, dan mekanisme pembuatan SIM. "
        "Gunakan informasi ini untuk menjawab pertanyaan calon siswa secara akurat.\n\n"
        "Kontak pendaftaran resmi (WhatsApp): 082191927620 dan 082193234971.\n\n"
    )
    body = "\n".join(
        [_courses_section(db), _instructors_section(db), _mechanisms_section(db)]
    )
    return PlainTextResponse(content=header + body, media_type="text/markdown; charset=utf-8")
