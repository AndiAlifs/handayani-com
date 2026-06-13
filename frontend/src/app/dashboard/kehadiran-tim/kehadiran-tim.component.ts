import { Component, OnInit, inject } from '@angular/core';
import { CommonModule, DecimalPipe, DatePipe } from '@angular/common';
import { AttendanceService } from '../../core/services/attendance.service';

@Component({
  selector: 'app-kehadiran-tim',
  standalone: true,
  imports: [CommonModule, DecimalPipe, DatePipe],
  templateUrl: './kehadiran-tim.component.html',
  styleUrl: './kehadiran-tim.component.css',
})
export class KehadiranTimComponent implements OnInit {
  private api = inject(AttendanceService);

  dailyAttendance: any[] = [];
  dailySummary: any = {
    total: 0,
    present_ontime: 0,
    present_late: 0,
    on_leave: 0,
    absent: 0,
  };
  minimumWorkHours = 8;
  loading = false;
  error = '';

  ngOnInit(): void {
    this.loadDailyAttendance();
  }

  loadDailyAttendance(): void {
    this.loading = true;
    this.error = '';
    this.api.getDailyAttendance().subscribe({
      next: (response) => {
        this.loading = false;
        this.dailyAttendance = response.data || [];
        this.dailySummary = response.summary || this.dailySummary;
        if (response.minimum_work_hours) {
          this.minimumWorkHours = response.minimum_work_hours;
        }
      },
      error: () => {
        this.loading = false;
        this.error = 'Gagal memuat data absensi harian.';
      },
    });
  }
}
