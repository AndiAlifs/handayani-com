import { TestBed } from '@angular/core/testing';
import { HttpClient, provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';
import { authInterceptor } from './auth.interceptor';

describe('authInterceptor', () => {
  let http: HttpClient;
  let ctrl: HttpTestingController;

  beforeEach(() => {
    localStorage.setItem('token', 'jwt-xyz');
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([authInterceptor])),
        provideHttpClientTesting(),
        provideRouter([]),
      ],
    });
    http = TestBed.inject(HttpClient);
    ctrl = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    localStorage.clear();
    ctrl.verify();
  });

  it('adds Authorization header', () => {
    http.get('/api/attendance/my-attendance/today').subscribe();
    const req = ctrl.expectOne('/api/attendance/my-attendance/today');
    expect(req.request.headers.get('Authorization')).toBe('Bearer jwt-xyz');
    req.flush({});
  });
});
