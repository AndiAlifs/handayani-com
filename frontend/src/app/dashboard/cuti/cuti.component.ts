import { Component, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ReactiveFormsModule, FormBuilder, Validators } from '@angular/forms';
import { AttendanceService } from '../../core/services/attendance.service';

@Component({
  selector: 'app-cuti',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule],
  templateUrl: './cuti.component.html',
  styleUrl: './cuti.component.css',
})
export class CutiComponent {
  private api = inject(AttendanceService);
  private fb = inject(FormBuilder);

  form = this.fb.group({
    start_date: ['', Validators.required],
    end_date: ['', Validators.required],
    reason: ['', Validators.required],
  });

  submitting = signal(false);
  message = signal('');
  isError = signal(false);

  onSubmit() {
    if (this.form.invalid) return;
    this.submitting.set(true);
    this.message.set('');

    this.api.submitLeave(this.form.value).subscribe({
      next: () => {
        this.message.set('Permohonan cuti berhasil diajukan.');
        this.isError.set(false);
        this.submitting.set(false);
        this.form.reset();
      },
      error: (e) => {
        this.message.set(e?.error?.error ?? 'Gagal mengajukan permohonan cuti.');
        this.isError.set(true);
        this.submitting.set(false);
      },
    });
  }
}
