import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, of } from 'rxjs';
import { catchError } from 'rxjs/operators';
import { Course } from '../models/course.model';
import { Instructor } from '../models/instructor.model';
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

@Injectable({
  providedIn: 'root'
})
export class ApiService {
  private baseUrl = environment.apiBaseUrl;

  constructor(private http: HttpClient) {}

  getCourses(): Observable<Course[]> {
    return this.http.get<Course[]>(`${this.baseUrl}/api/courses`).pipe(
      catchError(() => of(MOCK_COURSES))
    );
  }

  getInstructorSchedules(): Observable<Instructor[]> {
    return this.http.get<Instructor[]>(`${this.baseUrl}/api/instructors/schedule`).pipe(
      catchError(() => of(MOCK_INSTRUCTORS))
    );
  }

  getMechanisms(): Observable<Mechanism[]> {
    return this.http.get<Mechanism[]>(`${this.baseUrl}/api/mechanisms`).pipe(
      catchError(() => of(MOCK_MECHANISMS))
    );
  }

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
