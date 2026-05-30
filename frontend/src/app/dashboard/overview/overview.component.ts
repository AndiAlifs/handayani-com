import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ApiService } from '../../core/services/api.service';
import { StudentCrm } from '../../core/models/student-crm.model';
import { Session } from '../../core/models/session.model';

@Component({
  selector: 'app-overview',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './overview.component.html',
  styleUrl: './overview.component.css'
})
export class OverviewComponent implements OnInit {
  students: StudentCrm[] = [];
  sessions: Session[] = [];
  
  stats = {
    activeStudents: 0,
    leads: 0,
    completed: 0,
    upcomingSessions: 0
  };

  constructor(private api: ApiService) {}

  ngOnInit() {
    this.api.getStudentsCrm().subscribe(data => {
      this.students = data;
      this.calculateStats();
    });

    this.api.getSessions().subscribe(data => {
      this.sessions = data;
      this.calculateStats();
    });
  }

  private calculateStats() {
    this.stats.activeStudents = this.students.filter(s => s.status === 'active').length;
    this.stats.leads = this.students.filter(s => s.status === 'lead').length;
    this.stats.completed = this.students.filter(s => s.status === 'completed').length;
    
    const now = new Date();
    this.stats.upcomingSessions = this.sessions.filter(
      s => s.status === 'scheduled' && new Date(s.startTime) >= now
    ).length;
  }
}
