import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ApiService } from '../../core/services/api.service';
import { Instructor } from '../../core/models/instructor.model';

@Component({
  selector: 'app-instructor-schedule',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './instructor-schedule.component.html',
  styleUrl: './instructor-schedule.component.css'
})
export class InstructorScheduleComponent implements OnInit {
  instructors: Instructor[] = [];
  selectedInstructor: Instructor | null = null;
  days = ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu', 'Minggu'];
  timeSlots = ['09.00 - 12.00', '13.00 - 15.00', '15.00 - 17.00'];

  constructor(private api: ApiService) {}

  ngOnInit(): void {
    this.api.getPublicInstructorSchedules().subscribe(data => {
      this.instructors = data;
      if (data.length > 0) this.selectedInstructor = data[0];
    });
  }

  selectInstructor(instructor: Instructor): void {
    this.selectedInstructor = instructor;
  }

  getSlotStatus(day: string, timeSlot: string): string {
    if (!this.selectedInstructor) return '';
    const slot = this.selectedInstructor.schedule.find(
      s => s.day === day && s.timeSlot === timeSlot
    );
    return slot?.status || 'Tersedia';
  }

  getSlotClass(status: string): string {
    if (status === 'Tersedia') return 'slot-available';
    if (status === 'Libur') return 'slot-holiday';
    return 'slot-booked';
  }

  // Public page must not leak customer names — any booked slot reads "Terisi".
  getSlotLabel(status: string): string {
    if (status === 'Tersedia' || status === 'Libur') return status;
    return 'Terisi';
  }
}
