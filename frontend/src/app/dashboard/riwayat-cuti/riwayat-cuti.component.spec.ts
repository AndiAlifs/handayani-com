import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';
import { RiwayatCutiComponent } from './riwayat-cuti.component';

describe('RiwayatCutiComponent', () => {
  beforeEach(() => TestBed.configureTestingModule({
    imports: [RiwayatCutiComponent],
    providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
  }));
  it('renders', () => {
    const f = TestBed.createComponent(RiwayatCutiComponent);
    f.detectChanges();
    expect(f.componentInstance).toBeTruthy();
  });
});
