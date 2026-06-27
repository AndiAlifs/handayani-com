import { Routes } from '@angular/router';
import { LandingPageComponent } from './landing-page/landing-page.component';
import { authGuard } from './core/guards/auth.guard';
import { roleGuard } from './core/guards/role.guard';

export const routes: Routes = [
  { path: '', component: LandingPageComponent },
  { 
    path: 'login', 
    loadComponent: () => import('./auth/login/login.component').then(m => m.LoginComponent)
  },
  {
    path: 'dashboard',
    canActivate: [authGuard],
    loadComponent: () => import('./dashboard/dashboard-layout/dashboard-layout.component').then(m => m.DashboardLayoutComponent),
    children: [
      {
        path: '',
        loadComponent: () => import('./dashboard/overview/overview.component').then(m => m.OverviewComponent)
      },
      {
        path: 'kursus',
        canActivate: [roleGuard],
        data: { roles: ['manager'] },
        loadComponent: () => import('./dashboard/kursus/kursus.component').then(m => m.KursusComponent)
      },
      {
        path: 'instruktur',
        canActivate: [roleGuard],
        data: { roles: ['manager'] },
        loadComponent: () => import('./dashboard/instruktur/instruktur.component').then(m => m.InstrukturComponent)
      },
      {
        path: 'mekanisme',
        canActivate: [roleGuard],
        data: { roles: ['manager'] },
        loadComponent: () => import('./dashboard/mekanisme/mekanisme.component').then(m => m.MekanismeComponent)
      },
      {
        path: 'crm',
        canActivate: [roleGuard],
        data: { roles: ['manager'] },
        loadComponent: () => import('./dashboard/crm/crm.component').then(m => m.CrmComponent)
      },
      {
        path: 'sesi',
        canActivate: [roleGuard],
        data: { roles: ['manager', 'instructor'] },
        loadComponent: () => import('./dashboard/sesi/sesi.component').then(m => m.SesiComponent)
      },
      {
        path: 'whatsapp',
        canActivate: [roleGuard],
        data: { roles: ['manager'] },
        loadComponent: () => import('./dashboard/whatsapp/whatsapp.component').then(m => m.WhatsappComponent)
      }
    ]
  },
  { path: '**', redirectTo: '' }
];
