import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';
import { RiwayatAbsensiComponent } from './riwayat-absensi.component';

describe('RiwayatAbsensiComponent', () => {
  beforeEach(() => TestBed.configureTestingModule({
    imports: [RiwayatAbsensiComponent],
    providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
  }));
  it('renders', () => {
    const f = TestBed.createComponent(RiwayatAbsensiComponent);
    f.detectChanges();
    expect(f.componentInstance).toBeTruthy();
  });
});
