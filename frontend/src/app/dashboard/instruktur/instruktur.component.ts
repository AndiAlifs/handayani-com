import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../core/services/api.service';
import { Instructor, ScheduleSlot } from '../../core/models/instructor.model';

const DAYS = ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu', 'Minggu'];
const TIME_SLOTS = ['09.00 - 12.00', '13.00 - 15.00', '15.00 - 17.00'];

@Component({
  selector: 'app-instruktur',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './instruktur.component.html',
  styleUrl: './instruktur.component.css'
})
export class InstrukturComponent implements OnInit {
  instructors: Instructor[] = [];

  // Profile form modal
  isFormOpen = false;
  editingInstructor: Instructor | null = null;
  formData: Partial<Instructor> = {};

  // Schedule grid modal
  isScheduleOpen = false;
  scheduleInstructor: Instructor | null = null;
  days = DAYS;
  timeSlots = TIME_SLOTS;
  editingCell: { day: string; timeSlot: string } | null = null;
  cellValue = '';

  constructor(private api: ApiService) {}

  ngOnInit() {
    this.api.getInstructorSchedules().subscribe(data => {
      this.instructors = data;
    });
  }

  // ── Profile CRUD ─────────────────────────────
  openAddModal() {
    this.editingInstructor = null;
    this.formData = { gender: 'Laki-laki', transmission: 'Manual' };
    this.isFormOpen = true;
  }

  openEditModal(instructor: Instructor) {
    this.editingInstructor = instructor;
    this.formData = { ...instructor };
    this.isFormOpen = true;
  }

  closeForm() {
    this.isFormOpen = false;
  }

  saveInstructor() {
    if (this.editingInstructor) {
      const updated = { ...this.editingInstructor, ...this.formData } as Instructor;
      this.api.updateInstructor(updated).subscribe(saved => {
        const index = this.instructors.findIndex(i => i.id === saved.id);
        if (index !== -1) this.instructors[index] = saved;
      });
    } else {
      const draft: Instructor = {
        ...(this.formData as Instructor),
        id: Math.max(...this.instructors.map(i => i.id), 0) + 1,
        schedule: this.buildEmptySchedule()
      };
      this.api.createInstructor(draft).subscribe(saved => {
        this.instructors.push(saved);
      });
    }
    this.closeForm();
  }

  deleteInstructor(id: number) {
    if (confirm('Apakah Anda yakin ingin menghapus instruktur ini?')) {
      this.api.deleteInstructor(id).subscribe(() => {
        this.instructors = this.instructors.filter(i => i.id !== id);
      });
    }
  }

  // ── Weekly schedule grid ─────────────────────
  openScheduleModal(instructor: Instructor) {
    // Work on a deep clone so edits can be cancelled by closing without saving.
    this.scheduleInstructor = {
      ...instructor,
      schedule: this.ensureFullMatrix(instructor.schedule)
    };
    this.editingCell = null;
    this.isScheduleOpen = true;
  }

  closeSchedule() {
    this.isScheduleOpen = false;
    this.scheduleInstructor = null;
    this.editingCell = null;
  }

  getSlotStatus(day: string, timeSlot: string): string {
    const slot = this.scheduleInstructor?.schedule.find(s => s.day === day && s.timeSlot === timeSlot);
    return slot?.status || 'Tersedia';
  }

  getSlotClass(status: string): string {
    if (status === 'Tersedia') return 'slot-available';
    if (status === 'Libur') return 'slot-holiday';
    return 'slot-booked';
  }

  startEditCell(day: string, timeSlot: string) {
    this.editingCell = { day, timeSlot };
    const current = this.getSlotStatus(day, timeSlot);
    this.cellValue = current === 'Tersedia' ? '' : current;
  }

  isEditing(day: string, timeSlot: string): boolean {
    return this.editingCell?.day === day && this.editingCell?.timeSlot === timeSlot;
  }

  /** Quick-set a slot to "Tersedia" (free) or "Libur" (holiday). */
  setQuickStatus(status: string) {
    this.cellValue = status === 'Tersedia' ? '' : status;
    this.commitCell();
  }

  /** Save the typed student name (or status) into the active slot. */
  commitCell() {
    if (!this.editingCell || !this.scheduleInstructor) return;
    const value = this.cellValue.trim() || 'Tersedia';
    const slot = this.scheduleInstructor.schedule.find(
      s => s.day === this.editingCell!.day && s.timeSlot === this.editingCell!.timeSlot
    );
    if (slot) slot.status = value;
    this.editingCell = null;
    this.cellValue = '';
  }

  saveSchedule() {
    if (!this.scheduleInstructor) return;
    const id = this.scheduleInstructor.id;
    const schedule = this.scheduleInstructor.schedule;
    this.api.updateInstructorSchedule(id, schedule).subscribe(saved => {
      const index = this.instructors.findIndex(i => i.id === id);
      if (index !== -1) {
        this.instructors[index] = { ...this.instructors[index], schedule: saved };
      }
      this.closeSchedule();
    });
  }

  // ── Helpers ──────────────────────────────────
  private buildEmptySchedule(): ScheduleSlot[] {
    const slots: ScheduleSlot[] = [];
    for (const day of DAYS) {
      for (const timeSlot of TIME_SLOTS) {
        slots.push({ day, timeSlot, status: day === 'Minggu' ? 'Libur' : 'Tersedia' });
      }
    }
    return slots;
  }

  /** Return a fresh schedule that has every day/time-slot combination. */
  private ensureFullMatrix(existing: ScheduleSlot[]): ScheduleSlot[] {
    const slots: ScheduleSlot[] = [];
    for (const day of DAYS) {
      for (const timeSlot of TIME_SLOTS) {
        const found = existing?.find(s => s.day === day && s.timeSlot === timeSlot);
        slots.push({ day, timeSlot, status: found?.status || 'Tersedia' });
      }
    }
    return slots;
  }
}
