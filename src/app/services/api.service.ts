import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, of } from 'rxjs';
import { catchError } from 'rxjs/operators';
import {
  Course, Instructor, Mechanism,
  MOCK_COURSES, MOCK_INSTRUCTORS, MOCK_MECHANISMS
} from './mock-data';
import { environment } from '../../environments/environment';

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
}
