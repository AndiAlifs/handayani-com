import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, of } from 'rxjs';
import { catchError, map } from 'rxjs/operators';
import { Course } from '../models/course.model';
import { Instructor, ScheduleSlot } from '../models/instructor.model';
import { Mechanism } from '../models/mechanism.model';
import { StudentCrm } from '../models/student-crm.model';
import { Session } from '../models/session.model';
import {
  MOCK_COURSES,
  MOCK_INSTRUCTORS,
  MOCK_MECHANISMS,
  MOCK_STUDENTS_CRM,
  MOCK_SESSIONS
} from './mock-data';
import { environment } from '../../../environments/environment';

/**
 * ApiService talks to the Golang backend documented in the PRD (Epic 2-5).
 * Every call degrades gracefully: if the backend is unreachable the read
 * methods fall back to bundled mock data and the write methods echo the
 * payload back (assigning a client-side id for creates) so the dashboard
 * stays fully functional for offline demos.
 */
@Injectable({
  providedIn: 'root'
})
export class ApiService {
  private baseUrl = environment.apiBaseUrl;

  constructor(private http: HttpClient) {}

  // ──────────────────────────────────────────
  // COURSES  (Epic 2)
  // ──────────────────────────────────────────
  getCourses(): Observable<Course[]> {
    return this.http.get<Course[]>(`${this.baseUrl}/api/courses`).pipe(
      catchError(() => of(MOCK_COURSES))
    );
  }

  createCourse(course: Course): Observable<Course> {
    return this.http.post<Course>(`${this.baseUrl}/api/courses`, course).pipe(
      catchError(() => of({ ...course, id: course.id || Date.now() }))
    );
  }

  updateCourse(course: Course): Observable<Course> {
    return this.http.put<Course>(`${this.baseUrl}/api/courses/${course.id}`, course).pipe(
      catchError(() => of(course))
    );
  }

  deleteCourse(id: number): Observable<boolean> {
    return this.http.delete<void>(`${this.baseUrl}/api/courses/${id}`).pipe(
      map(() => true),
      catchError(() => of(true))
    );
  }

  // ──────────────────────────────────────────
  // INSTRUCTORS & SCHEDULES  (Epic 3)
  // ──────────────────────────────────────────
  getInstructorSchedules(): Observable<Instructor[]> {
    return this.http.get<Instructor[]>(`${this.baseUrl}/api/instructors/schedule`).pipe(
      catchError(() => of(MOCK_INSTRUCTORS))
    );
  }

  createInstructor(instructor: Instructor): Observable<Instructor> {
    return this.http.post<Instructor>(`${this.baseUrl}/api/instructors`, instructor).pipe(
      catchError(() => of({ ...instructor, id: instructor.id || Date.now() }))
    );
  }

  updateInstructor(instructor: Instructor): Observable<Instructor> {
    return this.http.put<Instructor>(`${this.baseUrl}/api/instructors/${instructor.id}`, instructor).pipe(
      catchError(() => of(instructor))
    );
  }

  deleteInstructor(id: number): Observable<boolean> {
    return this.http.delete<void>(`${this.baseUrl}/api/instructors/${id}`).pipe(
      map(() => true),
      catchError(() => of(true))
    );
  }

  /** Persist the full weekly schedule matrix for one instructor (Epic 3, AC2/AC3). */
  updateInstructorSchedule(id: number, schedule: ScheduleSlot[]): Observable<ScheduleSlot[]> {
    return this.http.put<ScheduleSlot[]>(`${this.baseUrl}/api/instructors/${id}/schedule`, schedule).pipe(
      catchError(() => of(schedule))
    );
  }

  // ──────────────────────────────────────────
  // MECHANISMS  (Epic 4)
  // ──────────────────────────────────────────
  getMechanisms(): Observable<Mechanism[]> {
    return this.http.get<Mechanism[]>(`${this.baseUrl}/api/mechanisms`).pipe(
      catchError(() => of(MOCK_MECHANISMS))
    );
  }

  createMechanism(mechanism: Mechanism): Observable<Mechanism> {
    return this.http.post<Mechanism>(`${this.baseUrl}/api/mechanisms`, mechanism).pipe(
      catchError(() => of({ ...mechanism, id: mechanism.id || Date.now() }))
    );
  }

  updateMechanism(mechanism: Mechanism): Observable<Mechanism> {
    return this.http.put<Mechanism>(`${this.baseUrl}/api/mechanisms/${mechanism.id}`, mechanism).pipe(
      catchError(() => of(mechanism))
    );
  }

  deleteMechanism(id: number): Observable<boolean> {
    return this.http.delete<void>(`${this.baseUrl}/api/mechanisms/${id}`).pipe(
      map(() => true),
      catchError(() => of(true))
    );
  }

  // ──────────────────────────────────────────
  // CRM & SESSIONS  (internal admin tooling)
  // ──────────────────────────────────────────
  getStudentsCrm(): Observable<StudentCrm[]> {
    return this.http.get<StudentCrm[]>(`${this.baseUrl}/api/crm/students`).pipe(
      catchError(() => of(MOCK_STUDENTS_CRM))
    );
  }

  getSessions(): Observable<Session[]> {
    return this.http.get<Session[]>(`${this.baseUrl}/api/sessions`).pipe(
      catchError(() => of(MOCK_SESSIONS))
    );
  }
}
