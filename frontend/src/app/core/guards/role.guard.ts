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
