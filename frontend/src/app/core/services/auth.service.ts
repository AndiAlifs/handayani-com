import { Injectable, signal, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router } from '@angular/router';
import { Observable, of } from 'rxjs';
import { map, catchError } from 'rxjs/operators';
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
  private readonly _currentUser = signal<AuthUser | null>(null);
  readonly currentUser = this._currentUser.asReadonly();

  constructor() {
    const stored = localStorage.getItem(this.USER_KEY);
    if (stored) {
      try { this._currentUser.set(JSON.parse(stored)); }
      catch { localStorage.removeItem(this.USER_KEY); }
    }
  }

  login(username: string, password: string): Observable<boolean> {
    return this.http.post<any>(`${environment.apiBaseUrl}/api/auth/login`, { username, password }).pipe(
      map((res) => {
        // Support both nested { user: {...} } and flat { id, username, full_name, role, ... }
        const u = res.user ?? res;
        const user: AuthUser = {
          id: u.id,
          username: u.username,
          name: u.full_name ?? u.username,
          role: (u.role ?? 'employee') as UserRole,
          isSuperAdmin: !!u.is_super_admin,
          officeId: u.office_id ?? null,
        };
        if (!res.token) throw new Error('missing token');
        localStorage.setItem(this.TOKEN_KEY, res.token);
        localStorage.setItem(this.USER_KEY, JSON.stringify(user));
        this._currentUser.set(user);
        return true;
      }),
      catchError(() => of(false)),
    );
  }

  logout(): void {
    this._currentUser.set(null);
    localStorage.removeItem(this.TOKEN_KEY);
    localStorage.removeItem(this.USER_KEY);
    this.router.navigate(['/login']);
  }

  token(): string | null { return localStorage.getItem(this.TOKEN_KEY); }
  isAuthenticated(): boolean {
    const token = this.token();
    if (!token || this.currentUser() === null) return false;
    return !this.isTokenExpired(token);
  }

  /** True if the JWT is malformed or its exp claim is in the past. */
  private isTokenExpired(token: string): boolean {
    try {
      const payload = JSON.parse(atob(token.split('.')[1] ?? ''));
      if (typeof payload.exp !== 'number') return false; // no exp → treat as non-expiring
      return payload.exp * 1000 <= Date.now();
    } catch {
      return true; // malformed token → treat as expired
    }
  }
  isManager(): boolean { return this.currentUser()?.role === 'manager'; }
  isInstructor(): boolean { return this.currentUser()?.role === 'instructor'; }
  isSuperAdmin(): boolean { return this.isManager() && !!this.currentUser()?.isSuperAdmin; }
  /** @deprecated Use isManager() instead. Kept for template compatibility. */
  isAdmin(): boolean { return this.isManager(); }
  hasRole(roles: UserRole[]): boolean {
    const r = this.currentUser()?.role;
    return !!r && roles.includes(r);
  }
}
