import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from '../services/auth.service';

/**
 * Restricts a route to managers. The backend is the authoritative gate
 * (ManagerMiddleware on /api/admin/*); this just keeps non-managers out of the
 * UI. Stack after authGuard: `canActivate: [authGuard, managerGuard]`.
 */
export const managerGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);
  if (auth.isManager()) {
    return true;
  }
  router.navigate(['/dashboard']);
  return false;
};
