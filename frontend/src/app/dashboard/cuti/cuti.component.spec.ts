import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';
import { CutiComponent } from './cuti.component';

describe('CutiComponent', () => {
  beforeEach(() => TestBed.configureTestingModule({
    imports: [CutiComponent],
    providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
  }));
  it('renders', () => {
    const f = TestBed.createComponent(CutiComponent);
    f.detectChanges();
    expect(f.componentInstance).toBeTruthy();
  });
});
