import { Component, OnInit, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AttendanceService } from '../../core/services/attendance.service';

@Component({
  selector: 'app-absensi',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './absensi.component.html',
  styleUrl: './absensi.component.css',
})
export class AbsensiComponent implements OnInit {
  private api = inject(AttendanceService);
  today = signal<any>(null);
  loading = signal(false);
  message = signal('');
  error = signal('');

  ngOnInit() { this.refresh(); }

  refresh() {
    this.api.getTodayAttendance().subscribe({
      next: (r) => this.today.set(r),
      error: () => this.today.set(null),
    });
  }

  private withPosition(action: (lat: number, lng: number) => void) {
    this.error.set('');
    if (!navigator.geolocation) { this.error.set('Geolokasi tidak didukung browser ini.'); return; }
    this.loading.set(true);
    navigator.geolocation.getCurrentPosition(
      (pos) => action(pos.coords.latitude, pos.coords.longitude),
      () => { this.loading.set(false); this.error.set('Izinkan akses lokasi untuk melakukan absensi.'); },
      { enableHighAccuracy: true, timeout: 10000 },
    );
  }

  clockIn() {
    this.withPosition((latitude, longitude) => {
      this.api.clockIn({ latitude, longitude }).subscribe({
        next: (r) => { this.loading.set(false); this.message.set(`Status: ${r.status}`); this.refresh(); },
        error: (e) => { this.loading.set(false); this.error.set(e?.error?.error ?? 'Gagal clock-in.'); },
      });
    });
  }

  clockOut() {
    this.withPosition((latitude, longitude) => {
      this.api.clockOut({ latitude, longitude }).subscribe({
        next: () => { this.loading.set(false); this.message.set('Clock-out berhasil.'); this.refresh(); },
        error: (e) => { this.loading.set(false); this.error.set(e?.error?.error ?? 'Gagal clock-out.'); },
      });
    });
  }
}
