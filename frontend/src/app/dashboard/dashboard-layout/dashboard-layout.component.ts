import { Component, signal, inject, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, RouterLink, RouterLinkActive } from '@angular/router';
import { AuthService } from '../../core/services/auth.service';

interface NavItem {
  label: string;
  icon: string;
  route: string;
  roles?: ('manager' | 'instructor')[];
}

@Component({
  selector: 'app-dashboard-layout',
  standalone: true,
  imports: [CommonModule, RouterModule, RouterLink, RouterLinkActive],
  templateUrl: './dashboard-layout.component.html',
  styleUrl: './dashboard-layout.component.css'
})
export class DashboardLayoutComponent {
  public authService = inject(AuthService);
  isSidebarOpen = signal(true);
  isMobile = signal(false);

  readonly navItems: NavItem[] = [
    { label: 'Overview', icon: 'overview', route: '/dashboard', roles: ['manager', 'instructor'] },
    { label: 'Kursus & Harga', icon: 'courses', route: '/dashboard/kursus', roles: ['manager'] },
    { label: 'Instruktur', icon: 'instructors', route: '/dashboard/instruktur', roles: ['manager'] },
    { label: 'Mekanisme SIM', icon: 'sim', route: '/dashboard/mekanisme', roles: ['manager'] },
    { label: 'CRM Siswa', icon: 'crm', route: '/dashboard/crm', roles: ['manager'] },
    { label: 'Sesi Pelatihan', icon: 'sessions', route: '/dashboard/sesi', roles: ['manager', 'instructor'] },
  ];

  get visibleNavItems(): NavItem[] {
    const role = this.authService.currentUser()?.role;
    return this.navItems.filter(item => !item.roles || item.roles.includes(role as any));
  }

  @HostListener('window:resize', ['$event'])
  onResize() {
    this.isMobile.set(window.innerWidth < 1024);
    if (this.isMobile()) {
      this.isSidebarOpen.set(false);
    }
  }

  ngOnInit() {
    this.isMobile.set(window.innerWidth < 1024);
    this.isSidebarOpen.set(window.innerWidth >= 1024);
  }

  toggleSidebar() {
    this.isSidebarOpen.update(v => !v);
  }

  logout() {
    this.authService.logout();
  }
}
