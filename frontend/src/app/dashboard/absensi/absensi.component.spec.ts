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
