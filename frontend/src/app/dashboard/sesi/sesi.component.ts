import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService } from '../../core/services/api.service';
import { Session, SessionStatus } from '../../core/models/session.model';
import { AuthService } from '../../core/services/auth.service';

@Component({
  selector: 'app-sesi',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './sesi.component.html',
  styleUrl: './sesi.component.css'
})
export class SesiComponent implements OnInit {
  sessions: Session[] = [];
  selectedSession: Session | null = null;
  newNotes = '';
  isAnalyzing = false;

  constructor(
    private api: ApiService,
    public authService: AuthService
  ) {}

  ngOnInit() {
    this.api.getSessions().subscribe(data => {
      // If instructor, filter by their ID (mock ID 2 for demo purposes, 
      // or filter based on auth user name matching instructor name)
      const user = this.authService.currentUser();
      if (user?.role === 'instructor') {
        this.sessions = data.filter(s => s.instructorName === user.name);
      } else {
        this.sessions = data; // Admin sees all
      }
    });
  }

  getSessionsByStatus(status: SessionStatus): Session[] {
    return this.sessions.filter(s => s.status === status);
  }

  selectSession(session: Session) {
    this.selectedSession = session;
    this.newNotes = session.rawNotes || '';
  }

  closeModal() {
    this.selectedSession = null;
    this.newNotes = '';
  }

  formatDate(dateStr: Date | string): string {
    return new Date(dateStr).toLocaleDateString('id-ID', {
      weekday: 'long',
      day: 'numeric',
      month: 'long',
      year: 'numeric'
    });
  }

  formatTime(dateStr: Date | string): string {
    return new Date(dateStr).toLocaleTimeString('id-ID', {
      hour: '2-digit',
      minute: '2-digit'
    });
  }

  submitNotes() {
    if (!this.selectedSession || !this.newNotes.trim()) return;

    this.isAnalyzing = true;
    this.api.analyzeSession(this.selectedSession, this.newNotes).subscribe(updated => {
      const idx = this.sessions.findIndex(s => s.id === updated.id);
      if (idx !== -1) this.sessions[idx] = updated;
      this.selectedSession = updated;
      this.newNotes = updated.rawNotes || '';
      this.isAnalyzing = false;
    });
  }
}
