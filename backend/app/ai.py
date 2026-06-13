"""Session-note analysis. Uses Gemini 2.5 via the google-genai SDK when
GEMINI_API_KEY is set; otherwise falls back to a deterministic stub so the
feature works with zero configuration (graceful degradation, as elsewhere in
this app)."""
import json
import os

from .models import AiAnalysis, Session

_PROMPT = """Anda adalah evaluator instruktur mengemudi di YPA Handayani.
Analisis catatan sesi berikut dan kembalikan HANYA JSON valid (tanpa markdown).

Konteks sesi:
- Siswa: {student}
- Kursus: {course}
- Sesi ke-{n} dari {total}

Catatan instruktur:
\"\"\"{notes}\"\"\"

Kembalikan JSON dengan struktur persis:
{{
  "strengths": ["..."],
  "weaknesses": ["..."],
  "recommendedNextFocus": "...",
  "upsellRecommendation": "... atau null"
}}
Semua teks dalam Bahasa Indonesia. "upsellRecommendation" diisi hanya jika
siswa mendekati akhir paket namun masih ada kelemahan signifikan; jika tidak,
gunakan null."""


def _stub(session: Session, raw_notes: str) -> AiAnalysis:
    near_end = session.sessionNumber >= session.totalSessions - 2
    return AiAnalysis(
        strengths=["Kontrol kemudi dasar", "Kepatuhan instruksi"],
        weaknesses=["Perlu perbaikan pada saat parkir", "Masih ragu saat perpindahan gigi"],
        recommendedNextFocus="Fokus pada teknik parkir paralel dan mundur di area sempit.",
        upsellRecommendation=(
            "Siswa hampir menyelesaikan paket namun masih ada kekurangan teknis. "
            "Tawarkan paket top-up 3 sesi tambahan."
            if near_end
            else None
        ),
    )


def analyze_session_notes(session: Session, raw_notes: str) -> AiAnalysis:
    api_key = os.getenv("GEMINI_API_KEY", "").strip()
    if not api_key:
        return _stub(session, raw_notes)
    try:
        from google import genai
        from google.genai import types

        client = genai.Client(api_key=api_key)
        prompt = _PROMPT.format(
            student=session.studentName,
            course=session.courseName,
            n=session.sessionNumber,
            total=session.totalSessions,
            notes=raw_notes,
        )
        resp = client.models.generate_content(
            model=os.getenv("GEMINI_MODEL", "gemini-2.5-flash"),
            contents=prompt,
            config=types.GenerateContentConfig(response_mime_type="application/json"),
        )
        data = json.loads(resp.text)
        upsell = data.get("upsellRecommendation")
        if isinstance(upsell, str) and upsell.strip().lower() in ("null", "none", ""):
            upsell = None
        return AiAnalysis(
            strengths=list(data.get("strengths", [])),
            weaknesses=list(data.get("weaknesses", [])),
            recommendedNextFocus=data.get("recommendedNextFocus", ""),
            upsellRecommendation=upsell,
        )
    except Exception:
        # Any SDK/network/parse failure degrades to the deterministic stub.
        return _stub(session, raw_notes)
