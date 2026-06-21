# Seeded Accounts & Data Reference

All login accounts live in the **Go / NYAMPE core service** (`core/seed/seed.go`).
The browser authenticates against the Go gateway at `/api/auth/login` (its native `/api/login`).
The content side (`backend/schema.sql` + `backend/seed.sql`, loaded into the shared MySQL) holds
**content only** (courses, mechanisms, CRM students, sessions) — it has no login accounts of its own.

> ⚠️ **Demo data.** Every password equals its username. Do not deploy these to production.

## Login accounts (Go service)

### Managers / Admins
| Username | Password | Role | Notes |
|---|---|---|---|
| `admin` | `admin` | manager | Super admin |
| `admin2` | `admin2` | manager | Regular manager |
| `admin_kendari` | `admin_kendari` | manager | Kendari office manager |

### Instructors
| Username | Password | Full name | Learners seeded |
|---|---|---|---|
| `instructor1` | `instructor1` | Instruktur Utama | 3 |
| `instruktur_bambang` | `instruktur_bambang` | Bambang Wijaya | 4 |
| `instruktur_sari` | `instruktur_sari` | Sari Lestari | 4 |
| `instruktur_joko` | `instruktur_joko` | Joko Susanto | 4 |
| `instruktur_dewi` | `instruktur_dewi` | Dewi Anggraini | 4 |
| `instruktur_agus` | `instruktur_agus` | Agus Setiawan | 4 |
| `instruktur_maya` | `instruktur_maya` | Maya Puspita | 4 |
| `instruktur_hendra` | `instruktur_hendra` | Hendra Gunawan | 4 |
| `instruktur_ratna` | `instruktur_ratna` | Ratna Sari | 4 |

Each instructor is seeded with their learners, learning plans (mix of planned/completed),
and sample sessions (one completed + one active).

### Employees (attendance)
| Username | Password | Role |
|---|---|---|
| `karyawan1` … `karyawan7` | same as username | employee |
| `hidayat` | `hidayat` | employee |

## Content data (FastAPI / MySQL — `backend/seed.sql`)

| Table | Rows | Coverage |
|---|---|---|
| `courses` | 50 | Mengemudi, Menjahit, Komputer, Bahasa Inggris/Mandarin/Jepang, Tata Boga, Kecantikan, Otomotif, Las |
| `mechanisms` | 14 | SIM A & C application pipeline + perpanjangan |
| `students_crm` | 40 | 10 lead · 22 active · 8 completed |
| `sessions` | 45 | completed (with Gemini AI analysis), scheduled, cancelled |

The 8 named instructors above are the same names referenced by `sessions.instructor_name`
in MySQL, so the two backends line up **by name**. (IDs differ: Go uses auto-increment,
FastAPI uses fixed `instructor_id` 1–8.)

## How to load

**FastAPI / MySQL:**
```bash
mysql -u root -p < backend/schema.sql
mysql -u root -p < backend/seed.sql   # truncates the 4 content tables, then reseeds
```

**Go service:** runs `seed.RunAll()` on startup (idempotent — skips rows that already exist).
To re-seed instructor learners from scratch, drop the Go service's tables first.
