# NYAMPE Attendance → handayani.com Super App — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fold the full NYAMPE field-attendance system into handayani.com as a super app — NYAMPE's Go logic runs behind handayani's FastAPI gateway against one shared `handayani` MySQL database, with NYAMPE's screens ported into handayani's Angular 18 dashboard, all runnable via docker-compose.

**Architecture:** The browser talks only to the FastAPI gateway (:8080). FastAPI serves `courses`/`mechanisms` directly and proxies `/api/auth|attendance|admin|instructor/**` to the vendored Go service (:8090). Go AutoMigrates NYAMPE tables into the shared `handayani` db. Real JWT auth (Go-issued) replaces handayani's mock auth via an HTTP interceptor.

**Tech Stack:** FastAPI + httpx (gateway), Go/Gin + GORM (vendored, NYAMPE logic), Angular 18 standalone, MySQL 8, Docker Compose + nginx.

**Companion spec:** `docs/superpowers/specs/2026-05-31-nyampe-attendance-integration-design.md`

**Source to port from:** `C:\projects\absence_alip` (NYAMPE). Referenced below as `$SRC`.

---

## File Structure

**New top-level dirs/files**
- `attendance-backend/` — vendored NYAMPE Go source (copied from `$SRC/backend`), with `Dockerfile` and modified `main.go`.
- `docker-compose.yml`, `.env.example` — repo root.
- `backend/Dockerfile`, `backend/app/routers/gateway.py`, `backend/tests/test_gateway.py`.
- `frontend/Dockerfile`, `frontend/nginx.conf`.
- `frontend/src/app/core/services/attendance.service.ts`
- `frontend/src/app/core/interceptors/auth.interceptor.ts`
- `frontend/src/app/core/guards/role.guard.ts`
- `frontend/src/app/dashboard/<feature>/*` — one folder per ported screen.

**Modified**
- `backend/app/main.py` — register gateway router, drop `instructors` router.
- `backend/schema.sql`, `backend/seed.sql` — drop `instructors`/`schedules`.
- `backend/requirements.txt` — add `httpx`.
- `frontend/src/app/core/services/auth.service.ts` — real JWT login.
- `frontend/src/app/app.config.ts` — interceptor.
- `frontend/src/app/app.routes.ts` — new dashboard children.
- `frontend/src/app/dashboard/dashboard-layout/dashboard-layout.component.ts` — sidebar nav + roles.
- Landing header — "Login / Staff" link.

**Removed**
- `backend/app/routers/instructors.py`
- `frontend/src/app/dashboard/{instruktur,crm,sesi}/*` (replaced by NYAMPE ports).

---

## PHASE A — Vendor the Go backend

### Task A1: Copy NYAMPE Go source into the repo

**Files:**
- Create: `attendance-backend/**` (copy of `$SRC/backend`)

- [ ] **Step 1: Copy the Go source tree** (exclude any built binary, `.env`, and `vendor/`)

```bash
# from repo root (the worktree)
mkdir -p attendance-backend
cp -r /c/projects/absence_alip/backend/. attendance-backend/
rm -f attendance-backend/.env attendance-backend/attendance-server attendance-backend/*.exe
```

- [ ] **Step 2: Confirm the module + entrypoint exist**

Run: `ls attendance-backend/main.go attendance-backend/go.mod`
Expected: both paths listed (module `field-attendance-system`, go 1.24.3).

- [ ] **Step 3: Commit**

```bash
git add attendance-backend
git commit -m "chore: vendor NYAMPE Go backend into attendance-backend/"
```

### Task A2: Make Go honor PORT and default to the shared db

`$SRC/backend/main.go` hardcodes `r.Run(":8080")` and ignores `PORT`; `database/db.go` defaults `DB_NAME` to `attendance_db`. Change both so compose can place it on 8090 against `handayani`.

**Files:**
- Modify: `attendance-backend/main.go` (final line)
- Modify: `attendance-backend/database/db.go` (DB_NAME default)

- [ ] **Step 1: Read the current tail of main.go**

Run: `tail -5 attendance-backend/main.go`
Expected: ends with `log.Println("Server starting on port 8080...")` and `r.Run(":8080")`.

- [ ] **Step 2: Replace the hardcoded run line**

Add `"os"` to the import block if absent, then replace the final two lines:

```go
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}
	log.Printf("Server starting on port %s...", port)
	r.Run(":" + port)
```

- [ ] **Step 3: Change the DB_NAME default in `database/db.go`**

Replace:

```go
	if dbName == "" {
		dbName = "attendance_db"
	}
```

with:

```go
	if dbName == "" {
		dbName = "handayani"
	}
```

- [ ] **Step 4: Verify it builds** (requires local Go; if unavailable, this is verified in the Docker build at Phase I)

Run: `cd attendance-backend && go build ./... && cd ..`
Expected: no output, exit 0. (If Go is not installed locally, skip and rely on Task I2's Docker build.)

- [ ] **Step 5: Commit**

```bash
git add attendance-backend/main.go attendance-backend/database/db.go
git commit -m "feat(attendance-backend): honor PORT env (default 8090), default DB to handayani"
```

---

## PHASE B — Database (single `handayani` db)

### Task B1: Remove instructors/schedules from handayani SQL

NYAMPE's Go `AutoMigrate` now owns the `instructors` table (and adds students/sessions/learning-plans/attendance/etc.). Handayani's SQL must stop creating the conflicting `instructors`/`schedules` tables.

**Files:**
- Modify: `backend/schema.sql`
- Modify: `backend/seed.sql`

- [ ] **Step 1: Inspect current seed.sql to know what to drop**

Run: `grep -n -i "instructor\|schedule" backend/schema.sql backend/seed.sql`
Expected: lists the `CREATE TABLE instructors`, `CREATE TABLE schedules`, and any `INSERT INTO instructors/schedules` lines.

- [ ] **Step 2: Delete the `instructors` and `schedules` CREATE TABLE blocks** from `backend/schema.sql`

Keep only the `CREATE DATABASE`, `USE handayani`, `courses`, and `mechanisms` blocks. Remove the entire `-- ── Instructors & Schedules (Epic 3) ──` section (both `CREATE TABLE instructors (...)` and `CREATE TABLE schedules (...)`).

- [ ] **Step 3: Delete instructor/schedule INSERTs from `backend/seed.sql`**

Remove every `INSERT INTO instructors ...` and `INSERT INTO schedules ...` statement. Keep course/mechanism seeds.

- [ ] **Step 4: Verify nothing references the dropped tables**

Run: `grep -n -i "instructor\|schedule" backend/schema.sql backend/seed.sql`
Expected: no matches (or only inside comments you intend to keep — there should be none).

- [ ] **Step 5: Commit**

```bash
git add backend/schema.sql backend/seed.sql
git commit -m "feat(db): drop instructors/schedules from handayani SQL (NYAMPE owns them)"
```

---

## PHASE C — FastAPI gateway

### Task C1: Add httpx dependency

**Files:**
- Modify: `backend/requirements.txt`

- [ ] **Step 1: Append httpx**

Add this line to `backend/requirements.txt`:

```
httpx==0.28.1
```

- [ ] **Step 2: Install locally** (for running tests)

Run: `cd backend && pip install -r requirements.txt && cd ..`
Expected: httpx installs successfully.

- [ ] **Step 3: Commit**

```bash
git add backend/requirements.txt
git commit -m "chore(backend): add httpx for gateway proxy"
```

### Task C2: Write the failing gateway test

The gateway must (a) rewrite `/api/auth/login` → Go `/api/login`, (b) rewrite `/api/attendance/clock-in` → Go `/api/clock-in`, (c) pass `/api/admin/*` and `/api/instructor/*` through unchanged, (d) forward the `Authorization` header and body, (e) return Go's status/body, (f) return 502 when Go is unreachable.

**Files:**
- Create: `backend/tests/__init__.py` (empty)
- Create: `backend/tests/test_gateway.py`

- [ ] **Step 1: Write the test using respx to mock the Go service**

Add `respx==0.22.0` and `pytest==8.3.4` to `backend/requirements.txt` (dev). Create `backend/tests/test_gateway.py`:

```python
import httpx
import respx
from fastapi.testclient import TestClient

from app.main import app

client = TestClient(app)
GO = "http://localhost:8090"


@respx.mock
def test_login_rewrites_to_go_login():
    route = respx.post(f"{GO}/api/login").mock(
        return_value=httpx.Response(200, json={"token": "jwt123"})
    )
    r = client.post("/api/auth/login", json={"username": "a", "password": "b"})
    assert r.status_code == 200
    assert r.json() == {"token": "jwt123"}
    assert route.called
    assert route.calls.last.request.content == b'{"username": "a", "password": "b"}'


@respx.mock
def test_attendance_prefix_stripped():
    route = respx.post(f"{GO}/api/clock-in").mock(
        return_value=httpx.Response(200, json={"status": "approved"})
    )
    r = client.post(
        "/api/attendance/clock-in",
        json={"latitude": 1.0, "longitude": 2.0},
        headers={"Authorization": "Bearer jwt123"},
    )
    assert r.status_code == 200
    assert route.called
    assert route.calls.last.request.headers["authorization"] == "Bearer jwt123"


@respx.mock
def test_admin_passthrough_with_query():
    route = respx.get(f"{GO}/api/admin/records").mock(
        return_value=httpx.Response(200, json=[])
    )
    r = client.get("/api/admin/records", headers={"Authorization": "Bearer x"})
    assert r.status_code == 200
    assert route.called


@respx.mock
def test_go_down_returns_502():
    respx.post(f"{GO}/api/clock-in").mock(side_effect=httpx.ConnectError("down"))
    r = client.post("/api/attendance/clock-in", json={}, headers={"Authorization": "Bearer x"})
    assert r.status_code == 502
    assert "tidak tersedia" in r.json()["error"]
```

- [ ] **Step 2: Install dev deps and run the test to confirm it fails**

Run: `cd backend && pip install respx pytest && python -m pytest tests/test_gateway.py -v; cd ..`
Expected: FAIL — `/api/auth/login` and `/api/attendance/*` return 404 (router not yet added).

### Task C3: Implement the gateway router

**Files:**
- Create: `backend/app/routers/gateway.py`

- [ ] **Step 1: Write the proxy router**

```python
"""Reverse-proxy router: forwards auth/attendance/admin/instructor calls to the
vendored Go (NYAMPE) service. The browser only ever talks to FastAPI."""
import os

import httpx
from fastapi import APIRouter, Request, Response

router = APIRouter(tags=["gateway"])

GO_BACKEND_URL = os.getenv("GO_BACKEND_URL", "http://localhost:8090").rstrip("/")
_TIMEOUT = httpx.Timeout(15.0)

# Frontend prefix -> Go path prefix.
_PREFIX_MAP = {
    "/api/auth": "/api",          # /api/auth/login   -> /api/login
    "/api/attendance": "/api",    # /api/attendance/clock-in -> /api/clock-in
    "/api/admin": "/api/admin",   # passthrough
    "/api/instructor": "/api/instructor",
}


def _target(path: str) -> str | None:
    for prefix, go_prefix in _PREFIX_MAP.items():
        if path == prefix or path.startswith(prefix + "/"):
            return go_prefix + path[len(prefix):]
    return None


async def _proxy(request: Request, path: str) -> Response:
    go_path = _target(path)
    if go_path is None:
        return Response(status_code=404)
    url = GO_BACKEND_URL + go_path
    body = await request.body()
    headers = {
        k: v for k, v in request.headers.items()
        if k.lower() in ("authorization", "content-type", "accept")
    }
    try:
        async with httpx.AsyncClient(timeout=_TIMEOUT) as cx:
            resp = await cx.request(
                request.method, url, params=request.query_params,
                content=body, headers=headers,
            )
    except httpx.HTTPError:
        return Response(
            content='{"error": "Layanan absensi tidak tersedia"}',
            status_code=502, media_type="application/json",
        )
    return Response(
        content=resp.content, status_code=resp.status_code,
        media_type=resp.headers.get("content-type"),
    )


@router.api_route(
    "/api/auth/{path:path}",
    methods=["GET", "POST", "PUT", "PATCH", "DELETE"],
)
async def proxy_auth(path: str, request: Request):
    return await _proxy(request, request.url.path)


@router.api_route(
    "/api/attendance/{path:path}",
    methods=["GET", "POST", "PUT", "PATCH", "DELETE"],
)
async def proxy_attendance(path: str, request: Request):
    return await _proxy(request, request.url.path)


@router.api_route(
    "/api/admin/{path:path}",
    methods=["GET", "POST", "PUT", "PATCH", "DELETE"],
)
async def proxy_admin(path: str, request: Request):
    return await _proxy(request, request.url.path)


@router.api_route(
    "/api/instructor/{path:path}",
    methods=["GET", "POST", "PUT", "PATCH", "DELETE"],
)
async def proxy_instructor(path: str, request: Request):
    return await _proxy(request, request.url.path)
```

### Task C4: Register the gateway, drop the instructors router

**Files:**
- Modify: `backend/app/main.py`
- Remove: `backend/app/routers/instructors.py`

- [ ] **Step 1: Edit `backend/app/main.py` imports**

Change:

```python
from .routers import courses, instructors, mechanisms, rag
```

to:

```python
from .routers import courses, mechanisms, rag, gateway
```

- [ ] **Step 2: Replace the router registrations**

Change the `include_router` block:

```python
app.include_router(courses.router)
app.include_router(instructors.router)
app.include_router(mechanisms.router)
app.include_router(rag.router)
```

to:

```python
app.include_router(courses.router)
app.include_router(mechanisms.router)
app.include_router(rag.router)
app.include_router(gateway.router)
```

- [ ] **Step 3: Delete the now-unused router file**

```bash
git rm backend/app/routers/instructors.py
```

- [ ] **Step 4: Confirm nothing else imports it**

Run: `grep -rn "instructors" backend/app`
Expected: no matches in `app/` (router removed). If `rag.py` or others reference it, those references must also be removed — check and clean.

### Task C5: Run the gateway tests green

- [ ] **Step 1: Run the suite**

Run: `cd backend && python -m pytest tests/test_gateway.py -v; cd ..`
Expected: 4 passed.

- [ ] **Step 2: Commit**

```bash
git add backend/app/routers/gateway.py backend/app/main.py backend/tests backend/requirements.txt
git rm --cached backend/app/routers/instructors.py 2>/dev/null; true
git commit -m "feat(gateway): proxy auth/attendance/admin/instructor to Go; drop instructors router"
```

---

## PHASE D — Frontend auth foundation

### Task D1: Real JWT AuthService

Replace mock login with a call to `/api/auth/login` (gateway → Go `/api/login`). Go returns `{ token, user: { id, username, role, is_super_admin, ... } }` (confirm shape by reading `$SRC/backend/handlers/auth.go` `Login`). Store token + user; decode role for guards.

**Files:**
- Modify: `frontend/src/app/core/services/auth.service.ts`
- Test: `frontend/src/app/core/services/auth.service.spec.ts`

- [ ] **Step 1: Read the Go login response shape**

Run: `sed -n '1,120p' /c/projects/absence_alip/backend/handlers/auth.go`
Expected: identify the JSON keys returned by `Login` (token field name, user object fields `role`, `is_super_admin`). Use those exact keys in the service below. (The plan assumes `{ "token": "...", "user": { "id", "username", "full_name", "role", "is_super_admin", "office_id" } }`; adjust property reads if the handler differs.)

- [ ] **Step 2: Write the failing spec**

```typescript
import { TestBed } from '@angular/core/testing';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideRouter } from '@angular/router';
import { AuthService } from './auth.service';

describe('AuthService', () => {
  let service: AuthService;
  let http: HttpTestingController;

  beforeEach(() => {
    localStorage.clear();
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    });
    service = TestBed.inject(AuthService);
    http = TestBed.inject(HttpTestingController);
  });

  it('stores token + user on successful login', (done) => {
    service.login('karyawan1', 'karyawan1').subscribe((ok) => {
      expect(ok).toBeTrue();
      expect(localStorage.getItem('token')).toBe('jwt-abc');
      expect(service.currentUser()?.role).toBe('employee');
      expect(service.isAuthenticated()).toBeTrue();
      done();
    });
    const req = http.expectOne('http://localhost:8080/api/auth/login');
    expect(req.request.body).toEqual({ username: 'karyawan1', password: 'karyawan1' });
    req.flush({ token: 'jwt-abc', user: { id: 3, username: 'karyawan1', full_name: 'Karyawan 1', role: 'employee', is_super_admin: false } });
  });

  it('isManager true for manager+super admin', () => {
    (service as any).currentUser.set({ id: 1, username: 'admin', name: 'Admin', role: 'manager', isSuperAdmin: true });
    expect(service.isManager()).toBeTrue();
    expect(service.isSuperAdmin()).toBeTrue();
  });
});
```

- [ ] **Step 3: Run the spec to confirm it fails**

Run: `cd frontend && npx ng test --watch=false --include='**/auth.service.spec.ts'; cd ..`
Expected: FAIL — `login` currently returns a `boolean`, not an `Observable`, and makes no HTTP call.

- [ ] **Step 4: Rewrite `auth.service.ts`**

```typescript
import { Injectable, signal, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router } from '@angular/router';
import { Observable } from 'rxjs';
import { map, catchError } from 'rxjs/operators';
import { of } from 'rxjs';
import { environment } from '../../../environments/environment';

export type UserRole = 'employee' | 'instructor' | 'manager';

export interface AuthUser {
  id: number;
  username: string;
  name: string;
  role: UserRole;
  isSuperAdmin: boolean;
  officeId?: number | null;
}

@Injectable({ providedIn: 'root' })
export class AuthService {
  private http = inject(HttpClient);
  private router = inject(Router);
  private readonly USER_KEY = 'handayani_auth_user';
  private readonly TOKEN_KEY = 'token';
  public currentUser = signal<AuthUser | null>(null);

  constructor() {
    const stored = localStorage.getItem(this.USER_KEY);
    if (stored) {
      try { this.currentUser.set(JSON.parse(stored)); }
      catch { localStorage.removeItem(this.USER_KEY); }
    }
  }

  login(username: string, password: string): Observable<boolean> {
    return this.http.post<any>(`${environment.apiBaseUrl}/api/auth/login`, { username, password }).pipe(
      map((res) => {
        const u = res.user ?? {};
        const user: AuthUser = {
          id: u.id,
          username: u.username,
          name: u.full_name ?? u.username,
          role: (u.role ?? 'employee') as UserRole,
          isSuperAdmin: !!u.is_super_admin,
          officeId: u.office_id ?? null,
        };
        localStorage.setItem(this.TOKEN_KEY, res.token);
        localStorage.setItem(this.USER_KEY, JSON.stringify(user));
        this.currentUser.set(user);
        return true;
      }),
      catchError(() => of(false)),
    );
  }

  logout(): void {
    this.currentUser.set(null);
    localStorage.removeItem(this.TOKEN_KEY);
    localStorage.removeItem(this.USER_KEY);
    this.router.navigate(['/login']);
  }

  token(): string | null { return localStorage.getItem(this.TOKEN_KEY); }
  isAuthenticated(): boolean { return !!this.token() && this.currentUser() !== null; }
  isManager(): boolean { return this.currentUser()?.role === 'manager'; }
  isInstructor(): boolean { return this.currentUser()?.role === 'instructor'; }
  isSuperAdmin(): boolean { return this.isManager() && !!this.currentUser()?.isSuperAdmin; }
  hasRole(roles: UserRole[]): boolean {
    const r = this.currentUser()?.role;
    return !!r && roles.includes(r);
  }
}
```

- [ ] **Step 5: Update the login component to use the Observable**

In `frontend/src/app/auth/login/login.component.ts`, replace the `setTimeout`/boolean block in `onLogin()` with:

```typescript
    this.isLoading.set(true);
    this.errorMessage.set('');
    this.authService.login(this.username, this.password).subscribe((success) => {
      this.isLoading.set(false);
      if (success) {
        this.router.navigate(['/dashboard']);
      } else {
        this.errorMessage.set('Username atau password salah. Silakan coba lagi.');
      }
    });
```

- [ ] **Step 6: Run the spec green**

Run: `cd frontend && npx ng test --watch=false --include='**/auth.service.spec.ts'; cd ..`
Expected: 2 passed.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/app/core/services/auth.service.ts frontend/src/app/core/services/auth.service.spec.ts frontend/src/app/auth/login/login.component.ts
git commit -m "feat(auth): real JWT login against gateway, unify roles"
```

### Task D2: Auth HTTP interceptor

**Files:**
- Create: `frontend/src/app/core/interceptors/auth.interceptor.ts`
- Test: `frontend/src/app/core/interceptors/auth.interceptor.spec.ts`
- Modify: `frontend/src/app/app.config.ts`

- [ ] **Step 1: Write the failing spec**

```typescript
import { TestBed } from '@angular/core/testing';
import { HttpClient, provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';
import { authInterceptor } from './auth.interceptor';

describe('authInterceptor', () => {
  let http: HttpClient; let ctrl: HttpTestingController;
  beforeEach(() => {
    localStorage.setItem('token', 'jwt-xyz');
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([authInterceptor])),
        provideHttpClientTesting(), provideRouter([]),
      ],
    });
    http = TestBed.inject(HttpClient); ctrl = TestBed.inject(HttpTestingController);
  });
  it('adds Authorization header', () => {
    http.get('/api/attendance/my-attendance/today').subscribe();
    const req = ctrl.expectOne('/api/attendance/my-attendance/today');
    expect(req.request.headers.get('Authorization')).toBe('Bearer jwt-xyz');
    req.flush({});
  });
});
```

- [ ] **Step 2: Run to confirm failure**

Run: `cd frontend && npx ng test --watch=false --include='**/auth.interceptor.spec.ts'; cd ..`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement the interceptor**

```typescript
import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { catchError, throwError } from 'rxjs';

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const router = inject(Router);
  const token = localStorage.getItem('token');
  const authReq = token
    ? req.clone({ setHeaders: { Authorization: `Bearer ${token}` } })
    : req;
  return next(authReq).pipe(
    catchError((err) => {
      if (err.status === 401) {
        localStorage.removeItem('token');
        router.navigate(['/login']);
      }
      return throwError(() => err);
    }),
  );
};
```

- [ ] **Step 4: Wire it into `app.config.ts`**

Change `provideHttpClient()` to:

```typescript
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { authInterceptor } from './core/interceptors/auth.interceptor';
// ...
    provideHttpClient(withInterceptors([authInterceptor])),
```

- [ ] **Step 5: Run green + commit**

Run: `cd frontend && npx ng test --watch=false --include='**/auth.interceptor.spec.ts'; cd ..`
Expected: 1 passed.

```bash
git add frontend/src/app/core/interceptors frontend/src/app/app.config.ts
git commit -m "feat(auth): attach JWT via interceptor, redirect 401 to login"
```

### Task D3: Role guard

**Files:**
- Create: `frontend/src/app/core/guards/role.guard.ts`

- [ ] **Step 1: Implement the data-driven role guard**

```typescript
import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService, UserRole } from '../services/auth.service';

export const roleGuard: CanActivateFn = (route) => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const allowed = (route.data?.['roles'] as UserRole[] | undefined) ?? [];
  if (!auth.isAuthenticated()) { router.navigate(['/login']); return false; }
  if (allowed.length === 0 || auth.hasRole(allowed)) return true;
  router.navigate(['/dashboard']);
  return false;
};
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/app/core/guards/role.guard.ts
git commit -m "feat(auth): add data-driven role guard"
```

### Task D4: AttendanceService (gateway client)

Single Angular service wrapping every NYAMPE endpoint through the gateway. The interceptor adds the token, so no per-call headers. Port the full method list from `$SRC/frontend/src/app/services/api.service.ts`, repointing URLs to the gateway prefixes.

**Files:**
- Create: `frontend/src/app/core/services/attendance.service.ts`

- [ ] **Step 1: Read the full NYAMPE api.service to enumerate every method**

Run: `cat /c/projects/absence_alip/frontend/src/app/services/api.service.ts`
Expected: the complete method list (attendance, leave, admin, office, settings, students, learning-plans, instructor). Each maps to a gateway URL per the table below.

- [ ] **Step 2: Implement the service** (URL mapping rules: NYAMPE `/login`→`/api/auth/login`; `/clock-in`,`/clock-out`,`/my-*`,`/office-location`,`/leave`→`/api/attendance/...`; `/admin/*`→`/api/admin/*`; `/instructor/*`→`/api/instructor/*`)

```typescript
import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';

@Injectable({ providedIn: 'root' })
export class AttendanceService {
  private http = inject(HttpClient);
  private base = environment.apiBaseUrl;
  private a = (p: string) => `${this.base}/api/attendance${p}`;
  private adm = (p: string) => `${this.base}/api/admin${p}`;
  private ins = (p: string) => `${this.base}/api/instructor${p}`;

  // Attendance (employee)
  clockIn(d: { latitude: number; longitude: number }): Observable<any> { return this.http.post(this.a('/clock-in'), d); }
  clockOut(d: { latitude: number; longitude: number }): Observable<any> { return this.http.post(this.a('/clock-out'), d); }
  getTodayAttendance(): Observable<any> { return this.http.get(this.a('/my-attendance/today')); }
  getMyAttendanceHistory(limit = 50, offset = 0): Observable<any> { return this.http.get(this.a(`/my-attendance/history?limit=${limit}&offset=${offset}`)); }
  getMyOffices(): Observable<any> { return this.http.get(this.a('/my-offices')); }
  getOfficeLocation(): Observable<any> { return this.http.get(this.a('/office-location')); }

  // Leave (employee)
  submitLeave(d: any): Observable<any> { return this.http.post(this.a('/leave'), d); }
  getTodayLeave(): Observable<any> { return this.http.get(this.a('/my-leave/today')); }
  getMyLeaveHistory(): Observable<any> { return this.http.get(this.a('/my-leave/history')); }

  // Manager / admin
  getAllRecords(): Observable<any> { return this.http.get(this.adm('/records')); }
  getDailyAttendance(): Observable<any> { return this.http.get(this.adm('/daily-attendance')); }
  getPendingClockIns(): Observable<any> { return this.http.get(this.adm('/pending-clockins')); }
  updateClockInStatus(id: number, status: string): Observable<any> { return this.http.patch(this.adm(`/clockin/${id}`), { status }); }
  getAllLeaveRequests(): Observable<any> { return this.http.get(this.adm('/leaves')); }
  updateLeaveStatus(id: number, status: string): Observable<any> { return this.http.patch(this.adm(`/leave/${id}`), { status }); }
  getAllEmployees(): Observable<any> { return this.http.get(this.adm('/employees')); }
  createEmployee(d: any): Observable<any> { return this.http.post(this.adm('/employees'), d); }
  updateEmployee(id: number, d: any): Observable<any> { return this.http.put(this.adm(`/employees/${id}`), d); }
  deleteEmployee(id: number): Observable<any> { return this.http.delete(this.adm(`/employees/${id}`)); }
  getAllOffices(): Observable<any> { return this.http.get(this.adm('/offices')); }
  createOffice(d: any): Observable<any> { return this.http.post(this.adm('/offices'), d); }
  updateOffice(id: number, d: any): Observable<any> { return this.http.put(this.adm(`/offices/${id}`), d); }
  deleteOffice(id: number): Observable<any> { return this.http.delete(this.adm(`/offices/${id}`)); }
  getManagerOffices(): Observable<any> { return this.http.get(this.adm('/my-offices')); }
  assignOffice(d: any): Observable<any> { return this.http.post(this.adm('/offices/assign'), d); }
  unassignOffice(d: any): Observable<any> { return this.http.post(this.adm('/offices/unassign'), d); }
  getSettings(): Observable<any> { return this.http.get(this.adm('/settings')); }
  getMinimumWorkHours(): Observable<any> { return this.http.get(this.adm('/settings/minimum-work-hours')); }
  updateMinimumWorkHours(d: any): Observable<any> { return this.http.put(this.adm('/settings/minimum-work-hours'), d); }
  getSessionDuration(): Observable<any> { return this.http.get(this.adm('/settings/session-duration')); }
  updateSessionDuration(d: any): Observable<any> { return this.http.put(this.adm('/settings/session-duration'), d); }
  getQuotaPresets(): Observable<any> { return this.http.get(this.adm('/settings/quota-presets')); }
  updateQuotaPresets(d: any): Observable<any> { return this.http.put(this.adm('/settings/quota-presets'), d); }

  // Students / learning plans (admin-owned)
  getStudents(): Observable<any> { return this.http.get(this.adm('/students')); }
  createStudent(d: any): Observable<any> { return this.http.post(this.adm('/students'), d); }
  updateStudent(id: number, d: any): Observable<any> { return this.http.put(this.adm(`/students/${id}`), d); }
  archiveStudent(id: number): Observable<any> { return this.http.put(this.adm(`/students/${id}/archive`), {}); }
  getStudentSessions(id: number): Observable<any> { return this.http.get(this.adm(`/students/${id}/sessions`)); }
  getLearningPlans(): Observable<any> { return this.http.get(this.adm('/learning-plans')); }
  createLearningPlan(d: any): Observable<any> { return this.http.post(this.adm('/learning-plans'), d); }
  updateLearningPlan(id: number, d: any): Observable<any> { return this.http.put(this.adm(`/learning-plans/${id}`), d); }
  deleteLearningPlan(id: number): Observable<any> { return this.http.delete(this.adm(`/learning-plans/${id}`)); }
  getAdminInstructors(): Observable<any> { return this.http.get(this.adm('/instructors')); }
  getInstructorLoad(): Observable<any> { return this.http.get(this.adm('/instructor-load')); }

  // Instructor self-service
  insGetStudents(): Observable<any> { return this.http.get(this.ins('/students')); }
  insGetSchedule(): Observable<any> { return this.http.get(this.ins('/schedule')); }
  insStartSession(d: any): Observable<any> { return this.http.post(this.ins('/session/start'), d); }
  insEndSession(d: any): Observable<any> { return this.http.post(this.ins('/session/end'), d); }
  insActiveSession(): Observable<any> { return this.http.get(this.ins('/session/active')); }
}
```

- [ ] **Step 3: Build-check (compiles) + commit**

Run: `cd frontend && npx tsc -p tsconfig.app.json --noEmit; cd ..`
Expected: no errors.

```bash
git add frontend/src/app/core/services/attendance.service.ts
git commit -m "feat(attendance): Angular gateway client service"
```

---

## PHASE E — Employee screens (port)

### Port Recipe (applies to every component task in Phases E–G)

For each NYAMPE component you port:

1. **Create folder** `frontend/src/app/dashboard/<feature>/` with `<feature>.component.ts`, `.html`, `.css`.
2. **Copy** the logic from the NYAMPE source `.ts` (and inline template if present) into the new files.
3. **Make it standalone:** add `standalone: true` and `imports: [CommonModule, FormsModule]` (+ `ReactiveFormsModule` if the source uses it) to the `@Component`. Split any inline `template:`/`styles:` into `.html`/`.css` and use `templateUrl`/`styleUrl`.
4. **Swap the data layer:** replace `ApiService`/`api.service` injection and its method calls with `AttendanceService` (Phase D4). Method names match the table in D4.
5. **Drop token plumbing:** delete any `getHeaders()`/manual `Authorization` usage — the interceptor handles it.
6. **Restyle:** replace NYAMPE's standalone-page chrome (its own navbar/full-page wrappers) with handayani's dashboard look — the component renders *inside* `dashboard-layout`'s `<router-outlet>`, so remove top-level nav/headers and keep just the feature content. Reuse Tailwind/utility classes already present in `frontend/src/app/dashboard/kursus/kursus.component.html` for cards/tables/buttons.
7. **Smoke test:** a spec that creates the component and asserts it renders (`fixture.detectChanges(); expect(component).toBeTruthy()`), with `provideHttpClient`, `provideHttpClientTesting`, `provideRouter([])`.

### Task E1: Clock-in screen (`absensi`) — fully worked exemplar

**Files:**
- Create: `frontend/src/app/dashboard/absensi/absensi.component.{ts,html,css}`
- Test: `frontend/src/app/dashboard/absensi/absensi.component.spec.ts`
- Source: `$SRC/frontend/src/app/components/clock-in/clock-in.component.ts`

- [ ] **Step 1: Read the source component**

Run: `cat /c/projects/absence_alip/frontend/src/app/components/clock-in/clock-in.component.ts`
Expected: see its geolocation capture (`navigator.geolocation.getCurrentPosition`), `clockIn`/`clockOut` calls, and today-status display logic.

- [ ] **Step 2: Write the smoke spec (failing)**

```typescript
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';
import { AbsensiComponent } from './absensi.component';

describe('AbsensiComponent', () => {
  beforeEach(() => TestBed.configureTestingModule({
    imports: [AbsensiComponent],
    providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
  }));
  it('renders', () => {
    const f = TestBed.createComponent(AbsensiComponent);
    f.detectChanges();
    expect(f.componentInstance).toBeTruthy();
  });
});
```

- [ ] **Step 3: Run to confirm failure**

Run: `cd frontend && npx ng test --watch=false --include='**/absensi.component.spec.ts'; cd ..`
Expected: FAIL — component does not exist.

- [ ] **Step 4: Create `absensi.component.ts`** (port using the Recipe; standalone, `AttendanceService`)

```typescript
import { Component, OnInit, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AttendanceService } from '../../core/services/attendance.service';

@Component({
  selector: 'app-absensi',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './absensi.component.html',
  styleUrl: './absensi.component.css',
})
export class AbsensiComponent implements OnInit {
  private api = inject(AttendanceService);
  today = signal<any>(null);
  loading = signal(false);
  message = signal('');
  error = signal('');

  ngOnInit() { this.refresh(); }

  refresh() {
    this.api.getTodayAttendance().subscribe({
      next: (r) => this.today.set(r),
      error: () => this.today.set(null),
    });
  }

  private withPosition(action: (lat: number, lng: number) => void) {
    this.error.set('');
    if (!navigator.geolocation) { this.error.set('Geolokasi tidak didukung browser ini.'); return; }
    this.loading.set(true);
    navigator.geolocation.getCurrentPosition(
      (pos) => action(pos.coords.latitude, pos.coords.longitude),
      () => { this.loading.set(false); this.error.set('Izinkan akses lokasi untuk melakukan absensi.'); },
      { enableHighAccuracy: true, timeout: 10000 },
    );
  }

  clockIn() {
    this.withPosition((latitude, longitude) => {
      this.api.clockIn({ latitude, longitude }).subscribe({
        next: (r) => { this.loading.set(false); this.message.set(`Status: ${r.status}`); this.refresh(); },
        error: (e) => { this.loading.set(false); this.error.set(e?.error?.error ?? 'Gagal clock-in.'); },
      });
    });
  }

  clockOut() {
    this.withPosition((latitude, longitude) => {
      this.api.clockOut({ latitude, longitude }).subscribe({
        next: () => { this.loading.set(false); this.message.set('Clock-out berhasil.'); this.refresh(); },
        error: (e) => { this.loading.set(false); this.error.set(e?.error?.error ?? 'Gagal clock-out.'); },
      });
    });
  }
}
```

- [ ] **Step 5: Create `absensi.component.html`** (dashboard-content style; no page chrome)

```html
<section class="p-6 space-y-4">
  <h1 class="text-2xl font-semibold">Absensi</h1>

  <div class="rounded-lg border bg-white p-4">
    <p class="text-sm text-gray-500">Status hari ini</p>
    <p class="text-lg font-medium" *ngIf="today() as t; else noRec">
      {{ t.status }} <span *ngIf="t.is_late">(Terlambat {{ t.minutes_late }} menit)</span>
    </p>
    <ng-template #noRec><p class="text-lg font-medium">Belum absen</p></ng-template>
  </div>

  <div class="flex gap-3">
    <button (click)="clockIn()" [disabled]="loading()"
      class="rounded bg-blue-600 px-4 py-2 text-white disabled:opacity-50">Clock In</button>
    <button (click)="clockOut()" [disabled]="loading()"
      class="rounded bg-gray-700 px-4 py-2 text-white disabled:opacity-50">Clock Out</button>
  </div>

  <p *ngIf="message()" class="text-green-600">{{ message() }}</p>
  <p *ngIf="error()" class="text-red-600">{{ error() }}</p>
</section>
```

- [ ] **Step 6: Create `absensi.component.css`** (empty is fine; Tailwind utilities used inline)

```css
/* feature-specific overrides only */
```

- [ ] **Step 7: Run the smoke spec green**

Run: `cd frontend && npx ng test --watch=false --include='**/absensi.component.spec.ts'; cd ..`
Expected: 1 passed.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/app/dashboard/absensi
git commit -m "feat(absensi): port clock-in/out screen into dashboard"
```

### Task E2: My attendance history (`riwayat-absensi`)

Apply the **Port Recipe**.
- Source: `$SRC/frontend/src/app/components/my-attendance-history/my-attendance-history.component.{ts,html,css}`
- Dest: `frontend/src/app/dashboard/riwayat-absensi/riwayat-absensi.component.{ts,html,css}`, selector `app-riwayat-absensi`.
- Service calls: `getMyAttendanceHistory(limit, offset)`.

- [ ] Step 1: `cat` the source `.ts` and `.html`.
- [ ] Step 2: Write failing smoke spec (template from Recipe step 7, class `RiwayatAbsensiComponent`).
- [ ] Step 3: Run → FAIL.
- [ ] Step 4: Create the three files; standalone; render the history rows in a `<table>` styled like `kursus.component.html`.
- [ ] Step 5: Run → PASS.
- [ ] Step 6: `git add frontend/src/app/dashboard/riwayat-absensi && git commit -m "feat(riwayat-absensi): port attendance history"`

### Task E3: Leave request (`cuti`)

Apply the **Port Recipe**.
- Source: `$SRC/frontend/src/app/components/leave-request/leave-request.component.*`
- Dest: `frontend/src/app/dashboard/cuti/cuti.component.*`, selector `app-cuti`.
- Service calls: `submitLeave({ start_date, end_date, reason })`, `getTodayLeave()`.
- Imports: `CommonModule, FormsModule` (or `ReactiveFormsModule` if source uses reactive forms — match source).

- [ ] Step 1: `cat` source. Step 2: failing smoke spec (`CutiComponent`). Step 3: FAIL. Step 4: port (form with start/end date + reason → `submitLeave`). Step 5: PASS. Step 6: commit `feat(cuti): port leave request form`.

### Task E4: Leave history (`riwayat-cuti`)

Apply the **Port Recipe**.
- Source: `$SRC/frontend/src/app/components/leave-history/leave-history.component.*`
- Dest: `frontend/src/app/dashboard/riwayat-cuti/riwayat-cuti.component.*`, selector `app-riwayat-cuti`.
- Service calls: `getMyLeaveHistory()`.

- [ ] Step 1–6 per Recipe; commit `feat(riwayat-cuti): port leave history`.

---

## PHASE F — Manager screens (port)

### Task F1: Team daily dashboard (`kehadiran-tim`)

Apply the **Port Recipe**. NYAMPE's `manager-dashboard` uses a large inline template — split it into `.html`.
- Source: `$SRC/frontend/src/app/components/manager-dashboard/manager-dashboard.component.ts`
- Dest: `frontend/src/app/dashboard/kehadiran-tim/kehadiran-tim.component.*`, selector `app-kehadiran-tim`.
- Service calls: `getDailyAttendance()`, plus pending list if the source combines it (else see F2).

- [ ] Step 1: `cat` source (note inline template). Step 2: failing smoke spec (`KehadiranTimComponent`). Step 3: FAIL. Step 4: port; move inline template → `.html`; swap to `AttendanceService`. Step 5: PASS. Step 6: commit `feat(kehadiran-tim): port manager daily dashboard`.

### Task F2: Pending approvals (`persetujuan`)

Apply the **Port Recipe**.
- Source: pending-clockin UI within `$SRC/.../manager-dashboard` or its own section.
- Dest: `frontend/src/app/dashboard/persetujuan/persetujuan.component.*`, selector `app-persetujuan`.
- Service calls: `getPendingClockIns()`, `updateClockInStatus(id, 'approved'|'rejected')`.

- [ ] Step 1–6 per Recipe; commit `feat(persetujuan): port pending clock-in approvals`.

### Task F3: Attendance reports (`laporan`)

Apply the **Port Recipe**.
- Source: `$SRC/frontend/src/app/components/attendance-reports/attendance-reports.component.*`
- Dest: `frontend/src/app/dashboard/laporan/laporan.component.*`, selector `app-laporan`.
- Service calls: `getAllRecords()`.

- [ ] Step 1–6 per Recipe; commit `feat(laporan): port attendance reports`.

### Task F4: Leave management (`manajemen-cuti`)

Apply the **Port Recipe**.
- Source: `$SRC/frontend/src/app/components/leave-management/leave-management.component.*`
- Dest: `frontend/src/app/dashboard/manajemen-cuti/manajemen-cuti.component.*`, selector `app-manajemen-cuti`.
- Service calls: `getAllLeaveRequests()`, `updateLeaveStatus(id, status)`.

- [ ] Step 1–6 per Recipe; commit `feat(manajemen-cuti): port leave management`.

### Task F5: Employee management (`karyawan`)

Apply the **Port Recipe**. NYAMPE handles employee CRUD inside the manager dashboard UI ("Manajemen Karyawan"); extract it into its own screen.
- Dest: `frontend/src/app/dashboard/karyawan/karyawan.component.*`, selector `app-karyawan`.
- Service calls: `getAllEmployees()`, `createEmployee(d)`, `updateEmployee(id, d)`, `deleteEmployee(id)`.

- [ ] Step 1: locate the employee-management markup/logic in `$SRC` (`grep -rn "employees" $SRC/frontend/src/app`). Step 2: failing smoke spec (`KaryawanComponent`). Step 3: FAIL. Step 4: build a table + create/edit form calling the service. Step 5: PASS. Step 6: commit `feat(karyawan): employee management screen`.

### Task F6: Office management (`kantor`)

Apply the **Port Recipe**. NYAMPE's `office-management` is already standalone — porting is mostly path + service swap + restyle.
- Source: `$SRC/frontend/src/app/components/office-management/office-management.component.ts`
- Dest: `frontend/src/app/dashboard/kantor/kantor.component.*`, selector `app-kantor`.
- Service calls: `getAllOffices()`, `createOffice`, `updateOffice`, `deleteOffice`, `getManagerOffices`, `assignOffice`, `unassignOffice`.

- [ ] Step 1–6 per Recipe; commit `feat(kantor): port office management`.

### Task F7: System settings (`pengaturan`)

Apply the **Port Recipe**. Super-admin only.
- Dest: `frontend/src/app/dashboard/pengaturan/pengaturan.component.*`, selector `app-pengaturan`.
- Service calls: `getSettings`, `getMinimumWorkHours`/`updateMinimumWorkHours`, `getSessionDuration`/`updateSessionDuration`, `getQuotaPresets`/`updateQuotaPresets`.

- [ ] Step 1: `grep -rn "settings" $SRC/frontend/src/app` to find the settings UI. Step 2–6 per Recipe; commit `feat(pengaturan): port system settings`.

---

## PHASE G — Replace instruktur / crm / sesi with NYAMPE

### Task G1: Remove handayani's old instruktur/crm/sesi components

**Files:**
- Remove: `frontend/src/app/dashboard/{instruktur,crm,sesi}/`
- Modify: `frontend/src/app/app.routes.ts` (their route entries — re-added in G5)

- [ ] **Step 1: Delete the three component folders**

```bash
git rm -r frontend/src/app/dashboard/instruktur frontend/src/app/dashboard/crm frontend/src/app/dashboard/sesi
```

- [ ] **Step 2: Remove their three `loadComponent` route entries** from `app.routes.ts` (the `kursus`/`mekanisme`/overview entries stay).

- [ ] **Step 3: Remove now-orphaned models/mock-data** they referenced

Run: `grep -rln "student-crm.model\|session.model\|MOCK_STUDENTS_CRM\|MOCK_SESSIONS" frontend/src/app`
Delete the unused model files and mock entries flagged (only if no remaining importer). Keep `course.model`, `mechanism.model`.

- [ ] **Step 4: Commit**

```bash
git commit -am "refactor: remove handayani instruktur/crm/sesi (replaced by NYAMPE)"
```

### Task G2: Students screen (`crm`)

Apply the **Port Recipe**. NYAMPE's `admin-students` is standalone.
- Source: `$SRC/frontend/src/app/components/admin-students/admin-students.component.ts`
- Dest: `frontend/src/app/dashboard/crm/crm.component.*`, selector `app-crm`.
- Service calls: `getStudents`, `createStudent`, `updateStudent`, `archiveStudent`, `getStudentSessions`.

- [ ] Step 1–6 per Recipe; commit `feat(crm): NYAMPE student management replaces handayani CRM`.

### Task G3: Learning plans screen (`sesi`)

Apply the **Port Recipe**. NYAMPE's `admin-learning-plans` is standalone.
- Source: `$SRC/frontend/src/app/components/admin-learning-plans/admin-learning-plans.component.ts`
- Dest: `frontend/src/app/dashboard/sesi/sesi.component.*`, selector `app-sesi`.
- Service calls: `getLearningPlans`, `createLearningPlan`, `updateLearningPlan`, `deleteLearningPlan`.

- [ ] Step 1–6 per Recipe; commit `feat(sesi): NYAMPE learning plans replace handayani sessions`.

### Task G4: Instructors insight screen (`instruktur`)

Apply the **Port Recipe**. NYAMPE's `admin-instructors` is standalone.
- Source: `$SRC/frontend/src/app/components/admin-instructors/admin-instructors.component.ts`
- Dest: `frontend/src/app/dashboard/instruktur/instruktur.component.*`, selector `app-instruktur`.
- Service calls: `getAdminInstructors`, `getInstructorLoad`.

- [ ] Step 1–6 per Recipe; commit `feat(instruktur): NYAMPE instructor insight replaces handayani instruktur`.

### Task G5: Wire all new routes + sidebar

**Files:**
- Modify: `frontend/src/app/app.routes.ts`
- Modify: `frontend/src/app/dashboard/dashboard-layout/dashboard-layout.component.ts`

- [ ] **Step 1: Add the dashboard children** in `app.routes.ts` under the existing `dashboard` route's `children`, each guarded by `roleGuard` with `data: { roles: [...] }`:

```typescript
import { roleGuard } from './core/guards/role.guard';
// ... inside dashboard children: (keep '', kursus, mekanisme)
{ path: 'absensi', loadComponent: () => import('./dashboard/absensi/absensi.component').then(m => m.AbsensiComponent) },
{ path: 'riwayat-absensi', loadComponent: () => import('./dashboard/riwayat-absensi/riwayat-absensi.component').then(m => m.RiwayatAbsensiComponent) },
{ path: 'cuti', loadComponent: () => import('./dashboard/cuti/cuti.component').then(m => m.CutiComponent) },
{ path: 'riwayat-cuti', loadComponent: () => import('./dashboard/riwayat-cuti/riwayat-cuti.component').then(m => m.RiwayatCutiComponent) },
{ path: 'kehadiran-tim', canActivate: [roleGuard], data: { roles: ['manager'] }, loadComponent: () => import('./dashboard/kehadiran-tim/kehadiran-tim.component').then(m => m.KehadiranTimComponent) },
{ path: 'persetujuan', canActivate: [roleGuard], data: { roles: ['manager'] }, loadComponent: () => import('./dashboard/persetujuan/persetujuan.component').then(m => m.PersetujuanComponent) },
{ path: 'laporan', canActivate: [roleGuard], data: { roles: ['manager'] }, loadComponent: () => import('./dashboard/laporan/laporan.component').then(m => m.LaporanComponent) },
{ path: 'manajemen-cuti', canActivate: [roleGuard], data: { roles: ['manager'] }, loadComponent: () => import('./dashboard/manajemen-cuti/manajemen-cuti.component').then(m => m.ManajemenCutiComponent) },
{ path: 'karyawan', canActivate: [roleGuard], data: { roles: ['manager'] }, loadComponent: () => import('./dashboard/karyawan/karyawan.component').then(m => m.KaryawanComponent) },
{ path: 'kantor', canActivate: [roleGuard], data: { roles: ['manager'] }, loadComponent: () => import('./dashboard/kantor/kantor.component').then(m => m.KantorComponent) },
{ path: 'pengaturan', canActivate: [roleGuard], data: { roles: ['manager'] }, loadComponent: () => import('./dashboard/pengaturan/pengaturan.component').then(m => m.PengaturanComponent) },
{ path: 'crm', canActivate: [roleGuard], data: { roles: ['manager','instructor'] }, loadComponent: () => import('./dashboard/crm/crm.component').then(m => m.CrmComponent) },
{ path: 'sesi', canActivate: [roleGuard], data: { roles: ['manager','instructor'] }, loadComponent: () => import('./dashboard/sesi/sesi.component').then(m => m.SesiComponent) },
{ path: 'instruktur', canActivate: [roleGuard], data: { roles: ['manager'] }, loadComponent: () => import('./dashboard/instruktur/instruktur.component').then(m => m.InstrukturComponent) },
```

- [ ] **Step 2: Update the sidebar `NavItem` type + `navItems`** in `dashboard-layout.component.ts` to the unified roles and new entries:

```typescript
interface NavItem { label: string; icon: string; route: string; roles?: ('employee'|'instructor'|'manager')[]; }

readonly navItems: NavItem[] = [
  { label: 'Overview', icon: 'overview', route: '/dashboard', roles: ['employee','instructor','manager'] },
  { label: 'Absensi', icon: 'clock', route: '/dashboard/absensi', roles: ['employee','instructor','manager'] },
  { label: 'Riwayat Absensi', icon: 'history', route: '/dashboard/riwayat-absensi', roles: ['employee','instructor','manager'] },
  { label: 'Cuti', icon: 'leave', route: '/dashboard/cuti', roles: ['employee','instructor'] },
  { label: 'Riwayat Cuti', icon: 'history', route: '/dashboard/riwayat-cuti', roles: ['employee','instructor'] },
  { label: 'Kehadiran Tim', icon: 'team', route: '/dashboard/kehadiran-tim', roles: ['manager'] },
  { label: 'Persetujuan', icon: 'check', route: '/dashboard/persetujuan', roles: ['manager'] },
  { label: 'Laporan', icon: 'report', route: '/dashboard/laporan', roles: ['manager'] },
  { label: 'Manajemen Cuti', icon: 'leave', route: '/dashboard/manajemen-cuti', roles: ['manager'] },
  { label: 'Karyawan', icon: 'users', route: '/dashboard/karyawan', roles: ['manager'] },
  { label: 'Kantor', icon: 'office', route: '/dashboard/kantor', roles: ['manager'] },
  { label: 'Pengaturan', icon: 'settings', route: '/dashboard/pengaturan', roles: ['manager'] },
  { label: 'CRM Siswa', icon: 'crm', route: '/dashboard/crm', roles: ['manager','instructor'] },
  { label: 'Sesi Pelatihan', icon: 'sessions', route: '/dashboard/sesi', roles: ['manager','instructor'] },
  { label: 'Instruktur', icon: 'instructors', route: '/dashboard/instruktur', roles: ['manager'] },
  { label: 'Kursus & Harga', icon: 'courses', route: '/dashboard/kursus', roles: ['manager'] },
  { label: 'Mekanisme SIM', icon: 'sim', route: '/dashboard/mekanisme', roles: ['manager'] },
];
```

(`visibleNavItems` already filters by `currentUser().role` — keep it.)

- [ ] **Step 3: Build the whole frontend to catch route/type errors**

Run: `cd frontend && npx ng build; cd ..`
Expected: build succeeds.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/app/app.routes.ts frontend/src/app/dashboard/dashboard-layout/dashboard-layout.component.ts
git commit -m "feat(dashboard): wire attendance/leave/manager/student routes + sidebar"
```

---

## PHASE H — Landing-page entry

### Task H1: "Login / Staff" link on the landing header

**Files:**
- Modify: landing header — `frontend/src/app/shared/components/navbar/navbar.component.html` (or `landing-page/hero-section` if the nav lives there; confirm by grep).

- [ ] **Step 1: Locate the landing header**

Run: `grep -rln "routerLink\|nav\|header" frontend/src/app/shared/components/navbar frontend/src/app/landing-page/hero-section`
Expected: find the markup that renders the top nav of the public landing page.

- [ ] **Step 2: Add a login link** (ensure `RouterLink` is imported in that standalone component's `imports`)

```html
<a routerLink="/login" class="rounded bg-blue-600 px-4 py-2 text-white">Login / Staff</a>
```

- [ ] **Step 3: Build + commit**

Run: `cd frontend && npx ng build; cd ..`
Expected: build succeeds.

```bash
git add frontend/src/app
git commit -m "feat(landing): add Login / Staff entry point"
```

---

## PHASE I — Docker Compose

### Task I1: Dockerfiles + nginx config

**Files:**
- Create: `backend/Dockerfile`, `attendance-backend/Dockerfile`, `frontend/Dockerfile`, `frontend/nginx.conf`

- [ ] **Step 1: `backend/Dockerfile` (FastAPI gateway)**

```dockerfile
FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY app ./app
EXPOSE 8080
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8080"]
```

- [ ] **Step 2: `attendance-backend/Dockerfile` (Go)**

```dockerfile
FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /attendance-server .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /attendance-server /attendance-server
EXPOSE 8090
ENV PORT=8090
CMD ["/attendance-server"]
```

- [ ] **Step 3: `frontend/nginx.conf`**

```nginx
server {
  listen 80;
  root /usr/share/nginx/html;
  index index.html;
  location /api/ { proxy_pass http://gateway:8080; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; }
  location / { try_files $uri $uri/ /index.html; }
}
```

- [ ] **Step 4: `frontend/Dockerfile`**

```dockerfile
FROM node:20 AS build
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npx ng build --configuration production

FROM nginx:alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist/*/browser /usr/share/nginx/html
EXPOSE 80
```

> Note: confirm the Angular build output path (`dist/<project>/browser` for Angular 18) — adjust the `COPY --from=build` glob if `angular.json`'s `outputPath` differs. Since `environment.prod.ts` sets `apiBaseUrl: ''`, prod calls are same-origin `/api/...` and nginx proxies them to the gateway.

- [ ] **Step 5: Commit**

```bash
git add backend/Dockerfile attendance-backend/Dockerfile frontend/Dockerfile frontend/nginx.conf
git commit -m "chore(docker): add Dockerfiles + nginx proxy config"
```

### Task I2: docker-compose.yml + .env.example

**Files:**
- Create: `docker-compose.yml`, `.env.example` (repo root)

- [ ] **Step 1: `.env.example`**

```dotenv
MYSQL_ROOT_PASSWORD=changeme
DB_USER=root
DB_PASSWORD=changeme
DB_NAME=handayani
JWT_SECRET=replace-with-a-long-random-secret
```

- [ ] **Step 2: `docker-compose.yml`**

```yaml
services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD}
      MYSQL_DATABASE: ${DB_NAME}
    ports: ["3306:3306"]
    volumes:
      - db_data:/var/lib/mysql
      - ./backend/schema.sql:/docker-entrypoint-initdb.d/01-schema.sql:ro
      - ./backend/seed.sql:/docker-entrypoint-initdb.d/02-seed.sql:ro
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-p${MYSQL_ROOT_PASSWORD}"]
      interval: 5s
      timeout: 5s
      retries: 20

  go-backend:
    build: ./attendance-backend
    environment:
      PORT: "8090"
      DB_HOST: mysql
      DB_PORT: "3306"
      DB_USER: ${DB_USER}
      DB_PASSWORD: ${DB_PASSWORD}
      DB_NAME: ${DB_NAME}
      JWT_SECRET: ${JWT_SECRET}
    depends_on:
      mysql:
        condition: service_healthy

  gateway:
    build: ./backend
    environment:
      GO_BACKEND_URL: http://go-backend:8090
      DB_HOST: mysql
      DB_PORT: "3306"
      DB_USER: ${DB_USER}
      DB_PASSWORD: ${DB_PASSWORD}
      DB_NAME: ${DB_NAME}
    ports: ["8080:8080"]
    depends_on:
      mysql:
        condition: service_healthy
      go-backend:
        condition: service_started

  frontend:
    build: ./frontend
    ports: ["4200:80"]
    depends_on:
      gateway:
        condition: service_started

volumes:
  db_data: {}
```

- [ ] **Step 3: Build all images (also validates Go + Angular builds)**

Run: `cp .env.example .env && docker compose build`
Expected: all four build stages succeed (this is the authoritative Go-build check if Go isn't installed locally).

- [ ] **Step 4: Commit**

```bash
git add docker-compose.yml .env.example
git commit -m "chore(docker): compose for mysql + go + gateway + frontend"
```

---

## PHASE J — End-to-end verification

### Task J1: Bring up the stack and verify the acceptance criteria

- [ ] **Step 1: Start everything**

Run: `docker compose up -d --build`
Expected: `mysql` healthy, `go-backend`, `gateway`, `frontend` running. Check: `docker compose ps`.

- [ ] **Step 2: Verify Go migrated into the shared db**

Run: `docker compose exec mysql mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -e "USE handayani; SHOW TABLES;"`
Expected: `courses`, `mechanisms` (from SQL) **and** `users`, `attendances`, `leave_requests`, `office_locations`, `manager_offices`, `system_settings`, `students`, `student_sessions`, `learning_plans`, `instructors` (from Go AutoMigrate).

- [ ] **Step 3: Verify login through the gateway**

Run: `curl -s -X POST localhost:8080/api/auth/login -H 'Content-Type: application/json' -d '{"username":"admin","password":"admin"}'`
Expected: JSON containing a `token`.

- [ ] **Step 4: Verify a protected proxied call**

Run: `TOKEN=...; curl -s localhost:8080/api/admin/records -H "Authorization: Bearer $TOKEN"`
Expected: 200 with a JSON array (not 401/404/502).

- [ ] **Step 5: Verify handayani's own endpoint still works**

Run: `curl -s localhost:8080/api/courses`
Expected: 200 JSON array of courses.

- [ ] **Step 6: Browser smoke (manual)**

Open `http://localhost:4200` → landing renders → "Login / Staff" → log in as `karyawan1`/`karyawan1` → `/dashboard/absensi` clock-in (use DevTools → Sensors to set location) → log in as `admin`/`admin` → `/dashboard/kehadiran-tim`, approve a pending clock-in, open `/dashboard/kantor`.

- [ ] **Step 7: Run all automated tests**

Run: `cd backend && python -m pytest -q; cd ../frontend && npx ng test --watch=false; cd ..`
Expected: backend + frontend suites pass.

- [ ] **Step 8: Final commit (if any verification fixups were needed)**

```bash
git commit -am "test: end-to-end verification fixups" || true
```

---

## Self-Review Notes (author)

- **Spec coverage:** §4.1 gateway → C2–C5; §4.2 single-db → A2, B1, I2; §5 auth/roles → D1–D3; §6 component table → E1–E4, F1–F7, G2–G4; §7 routing map → D4, G5; §8 docker → I1–I2; §9 error handling → C3 (502), E1 (geo), D2 (401); §10 testing → C2, D1–D2, E1 smoke, J1; §11 acceptance → J1.
- **Ports:** repetitive component tasks use the shared **Port Recipe** with per-component concretes (source path, dest path, selector, exact `AttendanceService` methods) rather than re-pasting full component code; clock-in (E1) is fully worked as the exemplar.
- **Type consistency:** `AuthService` exposes `isManager()/isSuperAdmin()/hasRole()`, `roleGuard` reads `route.data['roles']`, `AttendanceService` method names referenced in component tasks all exist in D4.
- **Known confirm-at-runtime points (flagged in-task, not placeholders):** exact Go login JSON keys (D1 step 1), Angular prod `dist` output path (I1 step 4), location of employee/settings UI in NYAMPE source (F5/F7 step 1), landing nav file (H1 step 1).
