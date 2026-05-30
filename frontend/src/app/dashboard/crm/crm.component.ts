import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ApiService } from '../../core/services/api.service';
import { StudentCrm, StudentStatus } from '../../core/models/student-crm.model';

@Component({
  selector: 'app-crm',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './crm.component.html',
  styleUrl: './crm.component.css'
})
export class CrmComponent implements OnInit {
  students: StudentCrm[] = [];

  constructor(private api: ApiService) {}

  ngOnInit() {
    this.api.getStudentsCrm().subscribe(data => {
      this.students = data;
    });
  }

  getStudentsByStatus(status: StudentStatus): StudentCrm[] {
    return this.students.filter(s => s.status === status);
  }

  formatDate(dateStr: Date | string): string {
    return new Date(dateStr).toLocaleDateString('id-ID', {
      day: 'numeric',
      month: 'short',
      year: 'numeric'
    });
  }
}
