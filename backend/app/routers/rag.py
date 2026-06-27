"""RAG knowledge-sync endpoint (PRD Epic 5).

Returns the entire knowledge base flattened into one descriptive Markdown
document optimised for LLM semantic search. The bot polls this on an interval
(no webhooks, per PRD 4.2).

Every fact is tagged with a stable source id (e.g. ``[#kursus-3]``) and a
``## Sumber`` legend maps each tag back to its origin table/row so the chat bot
can cite where an answer came from. ``GET /knowledge-sync.json`` exposes the same
chunks as structured JSON for bots that prefer machine-readable citation.

Instructor data lives in the NYAMPE Go service now, so it is fetched server-side
via a manager service account (``RAG_SERVICE_USERNAME`` / ``RAG_SERVICE_PASSWORD``).
Leave those unset to disable the instructor section — the rest of the KB still
builds (graceful degradation, consistent with the Gemini stub fallback).
"""
import logging
import os
import time
from dataclasses import dataclass
from datetime import datetime
from typing import Optional

import httpx
from fastapi import APIRouter, Depends
from fastapi.responses import PlainTextResponse
from pymysql.connections import Connection

from ..database import get_db

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/rag", tags=["rag"])

# The Go service is now the front door; RAG calls it server-side for the
# instructor list. (Formerly imported from the now-removed gateway router.)
GO_BACKEND_URL = os.getenv("GO_BACKEND_URL", "http://localhost:8080").rstrip("/")

_GO_TIMEOUT = httpx.Timeout(10.0)
# Cached manager token for the instructor fetch (shared across requests).
_token_cache: dict = {"token": None, "exp": 0.0}


def rupiah(value: int) -> str:
    if value == 0:
        return "Gratis"
    return "Rp" + f"{value:,}".replace(",", ".")


@dataclass
class KbChunk:
    """One citable fact in the knowledge base."""

    sourceId: str          # e.g. "kursus-3"
    type: str              # "kursus" | "mekanisme" | "instruktur"
    title: str
    body: str
    updatedAt: Optional[str] = None

    @property
    def tag(self) -> str:
        return f"[#{self.sourceId}]"


def _fmt_date(value) -> Optional[str]:
    if isinstance(value, datetime):
        return value.strftime("%Y-%m-%d")
    return None


# ── Section builders (each returns a list of citable chunks) ────────────────

def _course_chunks(db: Connection) -> list[KbChunk]:
    with db.cursor() as cur:
        cur.execute(
            "SELECT id, category, program_type, specifics, duration, price, "
            "registration_fee, remarks, updated_at FROM courses ORDER BY category, id"
        )
        rows = cur.fetchall()
    return [
        KbChunk(
            sourceId=f"kursus-{r['id']}",
            type="kursus",
            title=f"{r['category']} ({r['program_type']})",
            body=(
                f"Kursus {r['category']} ({r['program_type']}) — {r['specifics']}. "
                f"Durasi: {r['duration']}. Biaya kursus: {rupiah(r['price'])}. "
                f"Biaya pendaftaran: {rupiah(r['registration_fee'])}. "
                f"Keterangan: {r['remarks'] or '-'}."
            ),
            updatedAt=_fmt_date(r.get("updated_at")),
        )
        for r in rows
    ]


def _mechanism_chunks(db: Connection) -> tuple[list[KbChunk], int]:
    """Returns the mechanism chunks and the summed administrative cost."""
    with db.cursor() as cur:
        cur.execute(
            "SELECT id, requirement_name, issuing_body, cost, notes "
            "FROM mechanisms ORDER BY sort_order, id"
        )
        rows = cur.fetchall()
    chunks = [
        KbChunk(
            sourceId=f"mekanisme-{r['id']}",
            type="mekanisme",
            title=r["requirement_name"],
            body=(
                f"{r['requirement_name']} — diterbitkan oleh {r['issuing_body']}. "
                f"Biaya: {rupiah(r['cost'])}. Catatan: {r['notes'] or '-'}."
            ),
        )
        for r in rows
    ]
    return chunks, sum(r["cost"] for r in rows)


def _service_token(cx: httpx.Client, user: str, pwd: str) -> Optional[str]:
    """Log into the Go service as the manager service account and cache the JWT."""
    now = time.time()
    if _token_cache["token"] and _token_cache["exp"] > now + 60:
        return _token_cache["token"]
    resp = cx.post(
        GO_BACKEND_URL + "/api/login",
        json={"username": user, "password": pwd, "remember_me": True},
    )
    resp.raise_for_status()
    token = resp.json().get("token")
    if token:
        # Go issues a 7-day token for remember_me; refresh well before then.
        _token_cache["token"] = token
        _token_cache["exp"] = now + 6 * 24 * 60 * 60
    return token


def _instructor_chunks() -> list[KbChunk]:
    """Fetch instructors from the Go service. Returns [] if unconfigured/unreachable."""
    user = os.getenv("RAG_SERVICE_USERNAME", "").strip()
    pwd = os.getenv("RAG_SERVICE_PASSWORD", "").strip()
    if not user or not pwd:
        return []
    try:
        with httpx.Client(timeout=_GO_TIMEOUT) as cx:
            token = _service_token(cx, user, pwd)
            if not token:
                logger.warning(
                    "RAG instructor section omitted: service-account login to %s returned no token",
                    GO_BACKEND_URL,
                )
                return []
            url = GO_BACKEND_URL + "/api/admin/instructors"
            headers = {"Authorization": f"Bearer {token}"}
            resp = cx.get(url, headers=headers)
            if resp.status_code == 401:  # stale token — re-login once
                _token_cache["token"] = None
                token = _service_token(cx, user, pwd)
                resp = cx.get(url, headers=headers | {"Authorization": f"Bearer {token}"})
            resp.raise_for_status()
            rows = resp.json().get("data") or []
    except (httpx.HTTPError, ValueError, KeyError) as exc:
        # Silent degradation is intentional, but log so a misconfigured service
        # account (e.g. a non-manager → HTTP 403) is diagnosable rather than
        # vanishing without a trace.
        logger.warning("RAG instructor section omitted: %s", exc)
        return []

    chunks: list[KbChunk] = []
    for r in rows:
        name = r.get("full_name") or r.get("username") or "Instruktur"
        office = (r.get("office") or {}).get("name") if r.get("office") else None
        body = f"Instruktur {name} mengajar di YPA Handayani."
        if office:
            body = f"Instruktur {name} (kantor {office}) mengajar di YPA Handayani."
        chunks.append(
            KbChunk(
                sourceId=f"instruktur-{r.get('id')}",
                type="instruktur",
                title=name,
                body=body,
                updatedAt=_fmt_date_str(r.get("updated_at")),
            )
        )
    return chunks


def _fmt_date_str(value) -> Optional[str]:
    """Trim an ISO 8601 timestamp string (Go's JSON) to its date part."""
    if isinstance(value, str) and len(value) >= 10:
        return value[:10]
    return None


# ── Markdown assembly ───────────────────────────────────────────────────────

def _render_section(heading: str, chunks: list[KbChunk]) -> str:
    lines = [f"## {heading}", ""]
    for c in chunks:
        suffix = f" _(diperbarui {c.updatedAt})_" if c.updatedAt else ""
        lines.append(f"- {c.body}{suffix} {c.tag}")
    lines.append("")
    return "\n".join(lines)


def _render_legend(chunks: list[KbChunk]) -> str:
    lines = ["## Sumber (Source)", "", "Setiap fakta di atas ditandai sumbernya:"]
    for c in chunks:
        lines.append(f"- {c.tag} → tabel `{c.type}`, id `{c.sourceId.split('-')[-1]}` — {c.title}")
    lines.append("")
    return "\n".join(lines)


def _build_kb(db: Connection) -> tuple[str, list[KbChunk]]:
    courses = _course_chunks(db)
    instructors = _instructor_chunks()
    mechanisms, mech_total = _mechanism_chunks(db)
    all_chunks = courses + instructors + mechanisms

    synced = datetime.now().strftime("%Y-%m-%d %H:%M")
    header = (
        "# Basis Pengetahuan YPA Handayani\n\n"
        f"Disinkronkan pada {synced}. "
        "Dokumen ini berisi seluruh informasi resmi YPA Handayani: katalog kursus & harga, "
        "profil instruktur, dan mekanisme pembuatan SIM. Setiap fakta diberi tanda sumber "
        "(misalnya `[#kursus-3]`); lihat bagian Sumber di bawah untuk menelusurinya. "
        "Gunakan informasi ini untuk menjawab pertanyaan calon siswa secara akurat dan "
        "sebutkan sumbernya bila relevan.\n\n"
        "Kontak pendaftaran resmi (WhatsApp): 082191927620 dan 082193234971.\n\n"
    )

    sections = [_render_section("Katalog Kursus & Harga", courses)]
    if instructors:
        sections.append(_render_section("Instruktur", instructors))

    mech_section = _render_section("Mekanisme & Biaya Pembuatan SIM A", mechanisms)
    mech_section += f"\nPerkiraan total biaya administrasi SIM: {rupiah(mech_total)}.\n"
    sections.append(mech_section)

    sections.append(_render_legend(all_chunks))

    return header + "\n".join(sections), all_chunks


@router.get("/knowledge-sync", response_class=PlainTextResponse)
def knowledge_sync(db: Connection = Depends(get_db)):
    markdown, _ = _build_kb(db)
    return PlainTextResponse(content=markdown, media_type="text/markdown; charset=utf-8")


@router.get("/knowledge-sync.json")
def knowledge_sync_json(db: Connection = Depends(get_db)):
    """Same knowledge base as structured, citable chunks for bots."""
    _, chunks = _build_kb(db)
    return {
        "syncedAt": datetime.now().isoformat(timespec="seconds"),
        "chunks": [
            {
                "sourceId": c.sourceId,
                "type": c.type,
                "title": c.title,
                "body": c.body,
                "updatedAt": c.updatedAt,
            }
            for c in chunks
        ],
    }
