# Payroll & Employee-Database Module — Design Spec

**Date:** 2026-06-21 · **Status:** Approved design, pre-implementation · **Owner:** Andi Alifsyah

## 1. Context & motivation

A market signal (a public post by an Indonesian business owner shopping for a payroll
product) prompted the question of whether YPA Handayani's app should grow a payroll
capability. After scoping we settled on the highest-value, lowest-risk interpretation:

> **Add an internal payroll + employee-database module to *this* app so YPA Handayani can
> pay its own instructors and office staff.**

This is a natural fit because the codebase already owns the *unglamorous half* of payroll:
an employee database (`models.User`: role, office, work-hours config) and the work-hour /
attendance / session data that payroll consumes (`models.Attendance`, `models.LeaveRequest`,
`models.StudentSession`). What's missing is the **compensation + statutory layer**.

This is explicitly **not** a repackaging of the app to sell to that external buyer — they
asked for a payroll-only, attendance-free product, which is the opposite of this app's
identity.

## 2. Goals & non-goals

**Goals**
- Manage employee payroll-administrative data (biodata, NPWP, PTKP, BPJS numbers, bank, etc.).
- Run monthly payroll for **both** payee types:
  - **Office staff** (`employee` / `manager`) — fixed monthly base + allowances/deductions.
  - **Instructors** — variable pay derived from sessions/hours taught (`StudentSession`).
- Full Indonesian statutory calculation: **PPh 21 (TER scheme)** and **BPJS** (Kesehatan +
  Ketenagakerjaan).
- Generate **per-employee bukti potong PDF** and a **Coretax bulk-import file**.
- A **payroll simulation/preview** step before figures are committed.
- Deliver payslips via **three channels**: in-dashboard PDF, email, WhatsApp.

**Non-goals (this iteration)**
- Selling the module as a standalone product.
- Attendance/leave/scheduling changes (already exist; payroll only *reads* them).
- Direct programmatic submission to Coretax (DJP offers no general public payroll API; we
  produce an import-ready file a manager uploads manually).
- Multi-company / multi-tenant payroll.
- Loans/cash-advance lifecycle beyond a simple recurring-deduction component.

## 3. Architecture decision

**All-Go**, inside `attendance-backend`. Payroll data, the calculation engine, document
generation, and delivery all live in the Go gateway. This honours the project's stated
boundary ("Go gateway = everything; Python FastAPI = AI only") and matches the in-progress
migration of CRUD out of Python into Go. The FastAPI service is **not touched**.

The cost of all-Go is weaker document tooling than Python; we close that gap with deliberate
library choices:

| Concern | Library | Notes |
|---|---|---|
| PDF (payslip + bukti potong) | `github.com/johnfercher/maroto/v2` | Grid-based, pure Go, no external binary |
| Coretax import file (Excel) | `github.com/xuri/excelize/v2` | Pure Go, mature |
| Coretax import file (XML, if required) | stdlib `encoding/xml` | Format TBD — see §11 research item |
| Email | `gopkg.in/gomail.v2` (or `gomail` maintained fork) | SMTP |
| WhatsApp | stdlib `net/http` | Thin client to a configured gateway |

New code organised as:
- `attendance-backend/models/payroll.go` — GORM structs (added to AutoMigrate).
- `attendance-backend/payroll/` — the isolated, pure-function calculation engine + its tests.
- `attendance-backend/handlers/payroll.go` — HTTP handlers (CRUD + run lifecycle).
- `attendance-backend/payroll/documents/` — PDF + Coretax generators.
- `attendance-backend/payroll/delivery/` — delivery interface + adapters.
- `attendance-backend/seed/` — statutory-config seed (rates as data).

## 4. Data model

All tables are **owned by the Go gateway and added to `AutoMigrate`** (unlike the
`models/knowledge.go` content tables, whose DDL lives in `backend/schema.sql`). JSON is
**camelCase** to match the existing wire contract and Angular models.

### 4.1 `EmployeeProfile` — payroll-administrative layer (1:1 with `User`)
| Field | Type | Notes |
|---|---|---|
| `id` | uint PK | |
| `userId` | uint, unique, FK→users | |
| `nik` | string | KTP number |
| `npwp` | string | empty ⇒ +20% PPh 21 surcharge |
| `ptkpStatus` | enum `TK/0…TK/3, K/0…K/3, K/I/0…` | drives PTKP amount + TER category |
| `employmentType` | enum `permanent, contract, freelance` | informational + default tax category |
| `pph21Category` | enum `pegawai_tetap, bukan_pegawai, pegawai_tidak_tetap` | drives engine path (see §5) |
| `bankName`, `bankAccountNo`, `bankAccountName` | string | for transfer + payslip |
| `bpjsKesehatanNo`, `bpjsTkNo` | string | statutory IDs |
| `email`, `whatsapp` | string | delivery channels (User has no email today) |
| `joinDate` | date | |
| `isActive` | bool | excluded from runs when false |

### 4.2 `EmployeeCompensation` — pay basis (effective-dated)
| Field | Type | Notes |
|---|---|---|
| `id` | uint PK | |
| `userId` | uint FK | |
| `payBasis` | enum `monthly_fixed, per_session, per_hour` | |
| `baseSalary` | int64 (IDR) | for `monthly_fixed` |
| `rate` | int64 (IDR) | per session/hour for variable basis |
| `effectiveFrom`, `effectiveTo` | date | nullable `effectiveTo` = current |

### 4.3 `PayComponent` — master list of earning/deduction types
| Field | Type | Notes |
|---|---|---|
| `id` | uint PK | |
| `code` | string unique | e.g. `TUNJ_JABATAN`, `POT_PINJAMAN` |
| `name` | string | display label |
| `componentType` | enum `earning, deduction` | |
| `taxable` | bool | included in PPh 21 gross |
| `isBpjsBase` | bool | included in BPJS calculation base |
| `defaultCalc` | enum `fixed, manual` | |

### 4.4 `EmployeeComponent` — recurring components assigned to an employee
`id`, `userId` FK, `componentId` FK, `amount` int64, `effectiveFrom/To`.

### 4.5 `PayrollRun` — a period batch (the unit of work)
| Field | Type | Notes |
|---|---|---|
| `id` | uint PK | |
| `periodMonth`, `periodYear` | int | |
| `runType` | enum `regular, december_annual` | December triggers annual PPh 21 recalc |
| `status` | enum `draft, calculated, finalized, paid` | state machine (§6) |
| `payDate` | date | |
| `totalGross`, `totalDeductions`, `totalNet`, `totalPph21`, `totalBpjs` | int64 | run-level totals |
| `createdBy` | uint FK→users | |
| `notes` | string | |
| `calculatedAt`, `finalizedAt`, `paidAt` | datetime nullable | |

Unique constraint on (`periodMonth`, `periodYear`, `runType`) to prevent duplicate runs.

### 4.6 `Payslip` — one frozen snapshot per employee per run
Snapshots so historical slips never change when config/master data later does.
| Field | Type | Notes |
|---|---|---|
| `id` | uint PK | |
| `payrollRunId` | uint FK | |
| `userId` | uint FK | |
| `employeeNameSnapshot`, `npwpSnapshot`, `ptkpSnapshot` | string | frozen |
| `basisQty` | decimal | hours/sessions used (variable pay) |
| `grossEarnings` | int64 | |
| `bpjsKesEmployee`, `bpjsJhtEmployee`, `bpjsJpEmployee` | int64 | employee-side deductions |
| `bpjsKesEmployer`, `bpjsJhtEmployer`, `bpjsJpEmployer`, `bpjsJkk`, `bpjsJkm` | int64 | employer cost (reported, some added to bruto) |
| `biayaJabatan` | int64 | |
| `taxableIncome` | int64 | |
| `pph21` | int64 | |
| `totalDeductions`, `netPay` | int64 | |
| `calcStatus` | enum `ok, error` | a run can't finalize with any `error` |
| `calcError` | string | reason when `error` |
| `pdfPath` | string | generated at finalize |

### 4.7 `PayslipLine` — itemised lines for audit
`id`, `payslipId` FK, `componentCode`, `description`, `lineType` enum `earning, deduction, statutory`, `amount` int64, `sortOrder`.

### 4.8 `DeliveryLog` — per-channel delivery outcome
`id`, `payslipId` FK, `channel` enum `dashboard, email, whatsapp`, `status` enum `pending, sent, failed`, `detail` string, `sentAt` datetime. Retryable; never rolls back a finalized run.

### 4.9 `StatutoryConfig` — rates as data, effective-dated (**critical**)
Rather than hardcode tax tables, store them as seeded, effective-dated config so a regulation
change is a data edit. Concretely a small set of tables:
- `ptkp_amounts` — `status`, `annualAmount`, `effectiveFrom`.
- `ter_rates` — `category` (A/B/C), `lowerBound`, `upperBound`, `rate`, `effectiveFrom`.
- `progressive_brackets` — `lowerBound`, `upperBound`, `rate`, `effectiveFrom` (December recalc).
- `bpjs_config` — component (`kesehatan`, `jht`, `jp`, `jkk`, `jkm`), `employeeRate`,
  `employerRate`, `salaryCap` (nullable), `effectiveFrom`.
- `payroll_constants` — `biaya_jabatan_rate` (5%), `biaya_jabatan_max_month`, `no_npwp_surcharge` (20%).

Seed values for 2026 go in `seed/` and **must be verified against current regulation at
build time** (PMK 168/2023 for TER; current BPJS rates/caps; UU HPP brackets). See §11.

## 5. Calculation engine (`attendance-backend/payroll/`)

An isolated package of **pure functions** — config in, numbers out, **no DB/HTTP inside** —
so the legally load-bearing logic is fully unit-testable. Handlers load data + config, call
the engine, then persist.

```
CalcVariablePay(sessions, rate)        → grossVariable        // reads period's StudentSession hours
CalcEarnings(comp, components, varPay) → {gross, taxableGross, bpjsBase, lines[]}
CalcBPJS(bpjsBase, cfg)                → {kesEmp, kesEr, jhtEmp, jhtEr, jpEmp, jpEr, jkk, jkm}
CalcPPh21(ctx)                         → pph21                 // dispatches on pph21Category
BuildPayslip(employee, comp, run, cfg) → Payslip + PayslipLine[]
```

### 5.1 PPh 21 — dispatch by category
- **`pegawai_tetap` (office staff; default):**
  - **Jan–Nov:** `pph21 = TER_rate(category, monthlyGross) × monthlyGross`. TER category
    derived from PTKP status (A: TK/0,TK/1,K/0 · B: TK/2,TK/3,K/1,K/2 · C: K/3).
  - **December (`runType=december_annual`):** progressive annual recalculation —
    `annualNet = annualGross − biayaJabatan − employeeJHT/JP − PTKP`; apply progressive
    brackets; `decemberPph21 = annualTax − Σ(Jan–Nov TER withheld)`.
- **`bukan_pegawai` (per-session instructors, if applicable):** different scheme
  (cumulative `50% × gross` basis × progressive rate). Engine supports this path; **which
  category instructors fall under is confirmed during build** (open item ⚠#1).
- **No-NPWP surcharge:** PPh 21 × 1.2 when `npwp` empty.

### 5.2 BPJS (employee-side reduces net; employer-side reported, some added to bruto)
Current reference values (**verify**): Kesehatan 1% employee / 4% employer, cap IDR 12,000,000;
JHT 2% / 3.7%, no cap; JP 1% / 2%, cap (annually-adjusted, config); JKK 0.24–1.74% employer
(risk class); JKM 0.30% employer. Treatment of employer contributions in the PPh 21 bruto
(JKK/JKM/Kesehatan-employer added; JHT/JP-employee deductible) is encoded in the engine and
pinned by golden tests.

### 5.3 Variable pay
`CalcVariablePay` aggregates the employee's `StudentSession` records (deducted hours / session
counts) whose `check_out_time` falls within the run period, × the instructor's `rate`.

## 6. Run lifecycle (state machine)

```
draft ──calculate──▶ calculated ──finalize──▶ finalized ──mark-paid──▶ paid
  ▲          │             │ (re-runnable)         │ (locked; docs + delivery)
  └──────────┴─────────────┘
```

- **draft → calculated** (`POST /runs/{id}/calculate`): the **simulation**. Computes every
  payslip + lines, writes snapshots, sets per-payslip `calcStatus`. Re-runnable while in
  `draft`/`calculated` (recompute wipes & rebuilds payslips for the run).
- **calculated → finalized** (`POST /runs/{id}/finalize`): **blocked if any payslip has
  `calcStatus=error`**. Locks the run, generates payslip + bukti potong PDFs and the Coretax
  file, enqueues delivery.
- **finalized → paid** (`POST /runs/{id}/mark-paid`): records payment done.
- Post-finalize corrections go through a **new adjustment run**, never silent edits.

## 7. API surface (Go gateway)

Conventions match existing handlers: bare-body JSON (no `{"data":…}` envelope), camelCase,
`AuthMiddleware`-gated. **Manager-only** for setup + run management (like CRM); **employees
read only their own** payslips (JWT `userId` ownership check).

| Method | Path | Auth | Purpose |
|---|---|---|---|
| GET/POST/PUT/DELETE | `/api/payroll/employees` | manager | EmployeeProfile + Compensation CRUD |
| GET/POST/PUT/DELETE | `/api/payroll/components` | manager | PayComponent master |
| POST | `/api/payroll/runs` | manager | create draft run for a period |
| POST | `/api/payroll/runs/{id}/calculate` | manager | compute/simulate payslips |
| GET | `/api/payroll/runs` · `/runs/{id}` | manager | list / review run + payslips |
| POST | `/api/payroll/runs/{id}/finalize` | manager | lock + generate docs + deliver |
| POST | `/api/payroll/runs/{id}/mark-paid` | manager | mark paid |
| GET | `/api/payroll/runs/{id}/payslips` | manager | all payslips in a run |
| GET | `/api/payroll/payslips/{id}/pdf` | manager or owner | payslip PDF |
| GET | `/api/payroll/me/payslips` | any auth | caller's own finalized payslips |
| GET | `/api/payroll/runs/{id}/coretax` | manager | zip: Coretax import file + bukti potong PDFs |
| POST | `/api/payroll/payslips/{id}/redeliver` | manager | retry a failed delivery channel |

## 8. Documents

- **Payslip PDF** (maroto v2): employer header, employee + period, itemised earnings/
  deductions/statutory lines, net pay, BPJS + PPh 21 breakdown.
- **Bukti potong PDF** (maroto v2): laid out to resemble the official **e-Bupot 21**
  withholding slip. Exact field set verified against the current DJP form (§11).
- **Coretax import file** (excelize v2): the DJP **e-Bupot 21 bulk-import** template. Exact
  columns/format is a research item (§11) — built to a confirmed template, not guessed.

## 9. Delivery

`PayslipDeliverer` interface with three adapters, each toggled by config; results recorded in
`DeliveryLog`; failures retryable and isolated from run state.
- **Dashboard:** no-op — slip already visible to the owner via `/api/payroll/me/payslips`.
- **Email** (gomail/SMTP): payslip PDF attached. Requires SMTP env config.
- **WhatsApp** (`net/http`): message + link/PDF via a configured gateway (e.g. Cloud API or a
  local gateway). Requires gateway env config.

## 10. Frontend (Angular)

New dashboard area at Indonesian path `/dashboard/penggajian`, guarded by `authGuard`.
- **Manager:** employee compensation setup; component master; run list; run detail/review
  (per-employee table with full breakdown + error flags); finalize; download Coretax/bukti
  potong; redeliver.
- **Employee:** *"Slip Gaji Saya"* — list of own finalized payslips + PDF download.
- New `ApiService` methods + camelCase TS models in `core/models/`; `id`/`en` i18n strings;
  **graceful-degradation mock-data fallback preserved** per existing pattern.

## 11. Open items & research tasks

- **⚠#1 Instructor PPh 21 category** — confirm whether instructors are `pegawai_tetap` (TER)
  or `bukan_pegawai` (50%-basis scheme). Engine supports both; resolved during build.
- **⚠#2 Coretax template** — obtain and confirm the **current e-Bupot 21 mass-import
  template** (column layout / XML vs Excel) before building the generator. Treat as a
  build-time research task; do not guess the format.
- **Statutory rate verification** — verify all seeded 2026 values (TER tables, PTKP, BPJS
  rates/caps, JP cap, biaya jabatan, brackets) against current regulation before trusting any
  real payslip.
- **WhatsApp gateway choice** — pick the gateway (Cloud API vs local) and its env contract.

## 12. Error handling & edge cases

- Employee with no compensation profile / inactive ⇒ skipped, surfaced in run summary.
- No NPWP ⇒ +20% PPh 21 (handled, not an error).
- Missing BPJS number ⇒ still computed, flagged for follow-up.
- Per-payslip `calcStatus=error` blocks finalize; the run summary lists all errors.
- Document/delivery failures recorded per channel; retryable; never roll back a finalized run.
- Finalized runs are immutable; corrections via adjustment run.

## 13. Testing

The repo currently has **no test suite**; payroll is where one must start, because the
engine's correctness is legally load-bearing.
- **Go unit tests (golden cases)** for the `payroll` package: TER Jan–Nov, December annual
  recalc, no-NPWP surcharge, BPJS with/without caps, variable-pay aggregation — validated
  against worked DJP examples.
- **Handler tests** for the run state machine and permission gating (manager vs owner vs
  other).

## 14. Phasing (one coherent module, sequenced for incremental delivery)

1. **Data model + master CRUD** — models, AutoMigrate, EmployeeProfile/Compensation/
   PayComponent endpoints + statutory-config seed. (≈2–3 days) · **Status: Planned** 🔲
2. **Calculation engine + tests** — pure functions + golden tests; rate verification. (≈3–4 days) · **Status: Planned** 🔲
3. **Run lifecycle** — runs, calculate (simulation), finalize, mark-paid; payslip/line
   persistence. (≈2–3 days) · **Status: Planned** 🔲
4. **Documents** — payslip PDF, bukti potong PDF, Coretax import file. (≈3–5 days, gated on ⚠#2) · **Status: Planned** 🔲
5. **Delivery** — interface + dashboard/email/WhatsApp adapters + redeliver. (≈2 days) · **Status: Planned** 🔲
6. **Frontend** — manager payroll area + employee "Slip Gaji Saya". (≈3–4 days) · **Status: Planned** 🔲

Each phase ships and is demoable independently.
