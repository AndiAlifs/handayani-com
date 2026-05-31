import { Component, OnInit, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AttendanceService } from '../../core/services/attendance.service';

interface AttendanceRecord {
  id: number;
  clock_in_time: string;
  clock_out_time?: string;
  status: string;
  distance: number;
  work_hours?: number;
  is_late: boolean;
  minutes_late: number;
  latitude: number;
  longitude: number;
  approved_office?: {
    id: number;
    name: string;
    address: string;
  };
}

@Component({
  selector: 'app-riwayat-absensi',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './riwayat-absensi.component.html',
  styleUrl: './riwayat-absensi.component.css',
})
export class RiwayatAbsensiComponent implements OnInit {
  private api = inject(AttendanceService);

  records = signal<AttendanceRecord[]>([]);
  isLoading = signal(true);
  errorMessage = signal('');
  total = signal(0);
  currentPage = signal(1);
  totalPages = signal(1);

  readonly limit = 50;

  ngOnInit() { this.load(); }

  get offset() { return (this.currentPage() - 1) * this.limit; }

  load() {
    this.isLoading.set(true);
    this.errorMessage.set('');
    this.api.getMyAttendanceHistory(this.limit, this.offset).subscribe({
      next: (response) => {
        this.records.set(response.data || []);
        this.total.set(response.total || 0);
        this.totalPages.set(Math.ceil((response.total || 0) / this.limit) || 1);
        this.isLoading.set(false);
      },
      error: () => {
        this.errorMessage.set('Gagal memuat riwayat absensi. Silakan coba lagi.');
        this.isLoading.set(false);
      },
    });
  }

  prevPage() {
    if (this.currentPage() > 1) {
      this.currentPage.update(p => p - 1);
      this.load();
    }
  }

  nextPage() {
    if (this.currentPage() < this.totalPages()) {
      this.currentPage.update(p => p + 1);
      this.load();
    }
  }

  formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString('id-ID', {
      weekday: 'long', year: 'numeric', month: 'long', day: 'numeric',
    });
  }

  formatTime(dateString: string): string {
    return new Date(dateString).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' });
  }

  viewLocation(lat: number, lng: number) {
    window.open(`https://www.google.com/maps?q=${lat},${lng}`, '_blank');
  }
}
