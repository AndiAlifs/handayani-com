import { Injectable, signal } from '@angular/core';
import { Router } from '@angular/router';

export type UserRole = 'admin' | 'instructor';

export interface AuthUser {
  id: number;
  username: string;
  name: string;
  role: UserRole;
}

interface MockCredential {
  username: string;
  password: string;
  user: AuthUser;
}

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private readonly STORAGE_KEY = 'handayani_auth_user';
  public currentUser = signal<AuthUser | null>(null);

  private readonly MOCK_USERS: MockCredential[] = [
    {
      username: 'admin',
      password: 'admin123',
      user: { id: 1, username: 'admin', name: 'Administrator', role: 'admin' }
    },
    {
      username: 'instruktur',
      password: 'instruktur123',
      user: { id: 2, username: 'instruktur', name: 'Pak Bambang', role: 'instructor' }
    }
  ];

  constructor(private router: Router) {
    this.restoreSession();
  }

  private restoreSession(): void {
    const stored = localStorage.getItem(this.STORAGE_KEY);
    if (stored) {
      try {
        this.currentUser.set(JSON.parse(stored));
      } catch {
        localStorage.removeItem(this.STORAGE_KEY);
      }
    }
  }

  login(username: string, password: string): boolean {
    const match = this.MOCK_USERS.find(
      u => u.username === username && u.password === password
    );
    if (match) {
      this.currentUser.set(match.user);
      localStorage.setItem(this.STORAGE_KEY, JSON.stringify(match.user));
      return true;
    }
    return false;
  }

  logout(): void {
    this.currentUser.set(null);
    localStorage.removeItem(this.STORAGE_KEY);
    this.router.navigate(['/login']);
  }

  isAuthenticated(): boolean {
    return this.currentUser() !== null;
  }

  isAdmin(): boolean {
    return this.currentUser()?.role === 'admin';
  }
}
