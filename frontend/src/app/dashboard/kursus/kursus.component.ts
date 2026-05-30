import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../core/services/api.service';
import { Course } from '../../core/models/course.model';

@Component({
  selector: 'app-kursus',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './kursus.component.html',
  styleUrl: './kursus.component.css'
})
export class KursusComponent implements OnInit {
  courses: Course[] = [];
  isModalOpen = false;
  editingCourse: Course | null = null;
  
  // Form model
  formData: Partial<Course> = {};

  constructor(private api: ApiService) {}

  ngOnInit() {
    this.api.getCourses().subscribe(data => {
      this.courses = data;
    });
  }

  formatPrice(price: number): string {
    return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', minimumFractionDigits: 0 }).format(price);
  }

  openAddModal() {
    this.editingCourse = null;
    this.formData = {
      category: 'Mengemudi',
      programType: 'Reguler'
    };
    this.isModalOpen = true;
  }

  openEditModal(course: Course) {
    this.editingCourse = course;
    this.formData = { ...course };
    this.isModalOpen = true;
  }

  closeModal() {
    this.isModalOpen = false;
  }

  saveCourse() {
    if (this.editingCourse) {
      // Update existing — persist via API, then sync the local row.
      const updated = { ...this.editingCourse, ...this.formData } as Course;
      this.api.updateCourse(updated).subscribe(saved => {
        const index = this.courses.findIndex(c => c.id === saved.id);
        if (index !== -1) {
          this.courses[index] = saved;
        }
      });
    } else {
      // Create new — let the API assign the persisted record.
      const draft: Course = {
        ...(this.formData as Course),
        id: Math.max(...this.courses.map(c => c.id), 0) + 1
      };
      this.api.createCourse(draft).subscribe(saved => {
        this.courses.push(saved);
      });
    }
    this.closeModal();
  }

  deleteCourse(id: number) {
    if (confirm('Apakah Anda yakin ingin menghapus kursus ini?')) {
      this.api.deleteCourse(id).subscribe(() => {
        this.courses = this.courses.filter(c => c.id !== id);
      });
    }
  }
}
