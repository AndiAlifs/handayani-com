import { Component, OnInit, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AttendanceService } from '../../core/services/attendance.service';

@Component({
  selector: 'app-riwayat-cuti',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './riwayat-cuti.component.html',
  styleUrl: './riwayat-cuti.component.css',
})
export class RiwayatCutiComponent implements OnInit {
  private api = inject(AttendanceService);

  leaveHistory = signal<any[]>([]);
  loading = signal(false);
  errorMessage = signal('');

  ngOnInit() { this.load(); }

  load() {
    this.loading.set(true);
    this.errorMessage.set('');
    this.api.getMyLeaveHistory().subscribe({
      next: (response) => {
        this.leaveHistory.set(response.data || []);
        this.loading.set(false);
      },
      error: () => {
        this.errorMessage.set('Gagal memuat riwayat cuti. Silakan coba lagi.');
        this.loading.set(false);
      },
    });
  }

  statusClass(status: string): string {
    switch (status) {
      case 'approved': return 'inline-flex px-3 py-1 text-xs font-bold rounded-full bg-green-200 text-green-900 border-2 border-green-400';
      case 'rejected': return 'inline-flex px-3 py-1 text-xs font-bold rounded-full bg-red-200 text-red-900 border-2 border-red-400';
      case 'pending':  return 'inline-flex px-3 py-1 text-xs font-bold rounded-full bg-yellow-200 text-yellow-900 border-2 border-yellow-400';
      default:         return 'inline-flex px-3 py-1 text-xs font-bold rounded-full bg-gray-200 text-gray-900 border-2 border-gray-400';
    }
  }

  statusText(status: string): string {
    switch (status) {
      case 'approved': return 'DISETUJUI';
      case 'rejected': return 'DITOLAK';
      case 'pending':  return 'MENUNGGU';
      default:         return status.toUpperCase();
    }
  }

  cardClass(status: string): string {
    switch (status) {
      case 'approved': return 'border-l-4 border-green-500 bg-green-50';
      case 'rejected': return 'border-l-4 border-red-500 bg-red-50';
      case 'pending':  return 'border-l-4 border-yellow-500 bg-yellow-50';
      default:         return 'border-l-4 border-gray-400 bg-gray-50';
    }
  }

  calculateDuration(startDate: string, endDate: string): number {
    const start = new Date(startDate);
    const end = new Date(endDate);
    return Math.ceil(Math.abs(end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24)) + 1;
  }
}
