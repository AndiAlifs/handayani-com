import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ApiService } from '../../core/services/api.service';
import { Course } from '../../core/models/course.model';

@Component({
  selector: 'app-course-pricing',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './course-pricing.component.html',
  styleUrl: './course-pricing.component.css'
})
export class CoursePricingComponent implements OnInit {
  courses: Course[] = [];
  categories: string[] = [];
  activeCategory = '';

  constructor(private api: ApiService) {}

  ngOnInit(): void {
    this.api.getCourses().subscribe(courses => {
      this.courses = courses;
      this.categories = [...new Set(courses.map(c => c.category))];
      this.activeCategory = this.categories[0] || '';
    });
  }

  get filteredCourses(): Course[] {
    return this.courses.filter(c => c.category === this.activeCategory);
  }

  setCategory(cat: string): void {
    this.activeCategory = cat;
  }

  formatPrice(price: number): string {
    return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', minimumFractionDigits: 0 }).format(price);
  }

  getCategoryIcon(cat: string): string {
    const icons: Record<string, string> = {
      'Mengemudi': '🚗',
      'Menjahit': '🧵',
      'Komputer': '💻',
      'Bahasa Inggris': '🇬🇧',
      'Bahasa Mandarin': '🇨🇳'
    };
    return icons[cat] || '📚';
  }

  getCourseImage(course: Course): string {
    if (course.category === 'Mengemudi') {
      if (course.specifics?.includes('Manual')) return 'assets/images/courses/manual.png';
      if (course.specifics?.includes('Matic')) return 'assets/images/courses/matic.png';
      if (course.specifics?.includes('Hybrid')) return 'assets/images/courses/hybrid.png';
    }
    if (course.category === 'Menjahit') return 'assets/images/courses/sewing.png';
    if (course.category === 'Komputer') return 'assets/images/courses/computer.png';
    if (course.category === 'Bahasa Inggris') return 'assets/images/courses/english.png';
    if (course.category === 'Bahasa Mandarin') return 'assets/images/courses/mandarin.png';
    return 'assets/images/courses/manual.png';
  }
}
