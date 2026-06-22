import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { forkJoin } from 'rxjs';
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
    // Wait for both reads before computing, so stats are never derived against a
    // half-loaded view (which briefly showed wrong counts when one resolved first).
    forkJoin({
      students: this.api.getStudentsCrm(),
      sessions: this.api.getSessions(),
    }).subscribe(({ students, sessions }) => {
      this.students = students;
      this.sessions = sessions;
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
