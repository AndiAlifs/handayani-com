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
      // Update existing
      const index = this.courses.findIndex(c => c.id === this.editingCourse!.id);
      if (index !== -1) {
        this.courses[index] = { ...this.editingCourse, ...this.formData } as Course;
      }
    } else {
      // Create new
      const newCourse: Course = {
        ...(this.formData as Course),
        id: Math.max(...this.courses.map(c => c.id), 0) + 1
      };
      this.courses.push(newCourse);
    }
    this.closeModal();
  }

  deleteCourse(id: number) {
    if (confirm('Apakah Anda yakin ingin menghapus kursus ini?')) {
      this.courses = this.courses.filter(c => c.id !== id);
    }
  }
}
