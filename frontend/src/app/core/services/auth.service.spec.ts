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
