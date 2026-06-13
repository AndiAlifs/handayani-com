import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';

@Injectable({ providedIn: 'root' })
export class AttendanceService {
  private http = inject(HttpClient);
  private base = environment.apiBaseUrl;
  private a = (p: string) => `${this.base}/api/attendance${p}`;
  private adm = (p: string) => `${this.base}/api/admin${p}`;
  private ins = (p: string) => `${this.base}/api/instructor${p}`;

  // Attendance (employee)
  clockIn(d: { latitude: number; longitude: number }): Observable<any> { return this.http.post(this.a('/clock-in'), d); }
  clockOut(d: { latitude: number; longitude: number }): Observable<any> { return this.http.post(this.a('/clock-out'), d); }
  getTodayAttendance(): Observable<any> { return this.http.get(this.a('/my-attendance/today')); }
  getMyAttendanceHistory(limit = 50, offset = 0): Observable<any> { return this.http.get(this.a(`/my-attendance/history?limit=${limit}&offset=${offset}`)); }
  getMyOffices(): Observable<any> { return this.http.get(this.a('/my-offices')); }
  getOfficeLocation(): Observable<any> { return this.http.get(this.a('/office-location')); }

  // Leave (employee)
  submitLeave(d: any): Observable<any> { return this.http.post(this.a('/leave'), d); }
  getTodayLeave(): Observable<any> { return this.http.get(this.a('/my-leave/today')); }
  getMyLeaveHistory(): Observable<any> { return this.http.get(this.a('/my-leave/history')); }

  // Manager / admin
  getAllRecords(): Observable<any> { return this.http.get(this.adm('/records')); }
  getDailyAttendance(): Observable<any> { return this.http.get(this.adm('/daily-attendance')); }
  getPendingClockIns(): Observable<any> { return this.http.get(this.adm('/pending-clockins')); }
  updateClockInStatus(id: number, status: string): Observable<any> { return this.http.patch(this.adm(`/clockin/${id}`), { status }); }
  getAllLeaveRequests(): Observable<any> { return this.http.get(this.adm('/leaves')); }
  updateLeaveStatus(id: number, status: string): Observable<any> { return this.http.patch(this.adm(`/leave/${id}`), { status }); }
  getAllEmployees(): Observable<any> { return this.http.get(this.adm('/employees')); }
  createEmployee(d: any): Observable<any> { return this.http.post(this.adm('/employees'), d); }
  updateEmployee(id: number, d: any): Observable<any> { return this.http.put(this.adm(`/employees/${id}`), d); }
  deleteEmployee(id: number): Observable<any> { return this.http.delete(this.adm(`/employees/${id}`)); }
  getAllOffices(): Observable<any> { return this.http.get(this.adm('/offices')); }
  createOffice(d: any): Observable<any> { return this.http.post(this.adm('/offices'), d); }
  updateOffice(id: number, d: any): Observable<any> { return this.http.put(this.adm(`/offices/${id}`), d); }
  deleteOffice(id: number): Observable<any> { return this.http.delete(this.adm(`/offices/${id}`)); }
  getManagerOffices(): Observable<any> { return this.http.get(this.adm('/my-offices')); }
  assignOffice(d: any): Observable<any> { return this.http.post(this.adm('/offices/assign'), d); }
  unassignOffice(d: any): Observable<any> { return this.http.post(this.adm('/offices/unassign'), d); }
  getSettings(): Observable<any> { return this.http.get(this.adm('/settings')); }
  getMinimumWorkHours(): Observable<any> { return this.http.get(this.adm('/settings/minimum-work-hours')); }
  updateMinimumWorkHours(d: any): Observable<any> { return this.http.put(this.adm('/settings/minimum-work-hours'), d); }
  getSessionDuration(): Observable<any> { return this.http.get(this.adm('/settings/session-duration')); }
  updateSessionDuration(d: any): Observable<any> { return this.http.put(this.adm('/settings/session-duration'), d); }
  getQuotaPresets(): Observable<any> { return this.http.get(this.adm('/settings/quota-presets')); }
  updateQuotaPresets(d: any): Observable<any> { return this.http.put(this.adm('/settings/quota-presets'), d); }

  // Students / learning plans (admin-owned)
  getStudents(): Observable<any> { return this.http.get(this.adm('/students')); }
  createStudent(d: any): Observable<any> { return this.http.post(this.adm('/students'), d); }
  updateStudent(id: number, d: any): Observable<any> { return this.http.put(this.adm(`/students/${id}`), d); }
  archiveStudent(id: number): Observable<any> { return this.http.put(this.adm(`/students/${id}/archive`), {}); }
  getStudentSessions(id: number): Observable<any> { return this.http.get(this.adm(`/students/${id}/sessions`)); }
  adjustStudentQuota(id: number, remainingQuotaHours: number): Observable<any> { return this.http.put(this.adm(`/students/${id}/adjust-quota`), { remaining_quota_hours: remainingQuotaHours }); }
  reassignStudent(id: number, instructorId: number): Observable<any> { return this.http.put(this.adm(`/students/${id}/reassign`), { instructor_id: instructorId }); }
  getLearningPlans(params?: { instructor_id?: number; student_id?: number; period?: 'week' | 'month'; start_date?: string; end_date?: string }): Observable<any> {
    const parts: string[] = [];
    if (params) {
      if (params.instructor_id != null) parts.push(`instructor_id=${params.instructor_id}`);
      if (params.student_id != null) parts.push(`student_id=${params.student_id}`);
      if (params.period) parts.push(`period=${params.period}`);
      if (params.start_date) parts.push(`start_date=${params.start_date}`);
      if (params.end_date) parts.push(`end_date=${params.end_date}`);
    }
    const query = parts.length ? '?' + parts.join('&') : '';
    return this.http.get(this.adm(`/learning-plans${query}`));
  }
  createLearningPlan(d: any): Observable<any> { return this.http.post(this.adm('/learning-plans'), d); }
  updateLearningPlan(id: number, d: any): Observable<any> { return this.http.put(this.adm(`/learning-plans/${id}`), d); }
  deleteLearningPlan(id: number): Observable<any> { return this.http.delete(this.adm(`/learning-plans/${id}`)); }
  bulkCreateLearningPlans(d: { student_id: number; days_of_week: number[]; start_time: string; end_time: string; from_date: string; to_date: string; force?: boolean }): Observable<any> { return this.http.post(this.adm('/learning-plans/bulk'), d); }
  getAdminInstructors(): Observable<any> { return this.http.get(this.adm('/instructors')); }
  getInstructorLoad(): Observable<any> { return this.http.get(this.adm('/instructor-load')); }
  downloadStudentRoster(): Observable<Blob> { return this.http.get(this.adm('/students/roster.xlsx'), { responseType: 'blob' }); }

  // Instructor self-service (students)
  insGetStudents(active?: string): Observable<any> {
    const params = active ? `?active=${active}` : '';
    return this.http.get(this.ins(`/students${params}`));
  }
  insCreateStudent(d: { name: string; total_quota_hours: number; whatsapp: string; gender: string; meeting_point?: string; initial_schedule_date?: string; initial_start_time?: string; initial_end_time?: string }): Observable<any> { return this.http.post(this.ins('/students'), d); }
  insUpdateStudent(id: number, d: { name?: string; whatsapp?: string; gender?: string; meeting_point?: string }): Observable<any> { return this.http.put(this.ins(`/students/${id}`), d); }
  insAdjustQuota(id: number, remainingQuotaHours: number): Observable<any> { return this.http.put(this.ins(`/students/${id}/adjust-quota`), { remaining_quota_hours: remainingQuotaHours }); }
  insArchiveStudent(id: number): Observable<any> { return this.http.put(this.ins(`/students/${id}/archive`), {}); }
  insGetStudentSessions(id: number): Observable<any> { return this.http.get(this.ins(`/students/${id}/sessions`)); }

  // Instructor self-service (schedule/learning plans)
  insGetSchedule(period: 'week' | 'month' = 'month', startDate?: string, endDate?: string): Observable<any> {
    let params = `?period=${period}`;
    if (startDate) params += `&start_date=${startDate}`;
    if (endDate) params += `&end_date=${endDate}`;
    return this.http.get(this.ins(`/schedule${params}`));
  }
  insCreateLearningPlan(d: { student_id: number; scheduled_date: string; start_time: string; end_time: string; status?: string }): Observable<any> { return this.http.post(this.ins('/schedule'), d); }
  insUpdateLearningPlan(id: number, d: { scheduled_date?: string; start_time?: string; end_time?: string; status?: string }): Observable<any> { return this.http.put(this.ins(`/schedule/${id}`), d); }
  insDeleteLearningPlan(id: number): Observable<any> { return this.http.delete(this.ins(`/schedule/${id}`)); }
  insBulkCreateLearningPlans(d: { student_id: number; days_of_week: number[]; start_time: string; end_time: string; from_date: string; to_date: string; force?: boolean }): Observable<any> { return this.http.post(this.ins('/schedule/bulk'), d); }

  // Instructor self-service (sessions)
  insStartSession(d: { student_id: number; latitude: number; longitude: number }): Observable<any> { return this.http.post(this.ins('/session/start'), d); }
  insEndSession(d: { session_id?: number; student_id?: number; notes?: string; custom_check_in_time?: string; custom_check_out_time?: string }): Observable<any> { return this.http.post(this.ins('/session/end'), d); }
  insActiveSession(): Observable<any> { return this.http.get(this.ins('/session/active')); }
  insGetQuotaPresets(): Observable<any> { return this.http.get(this.ins('/quota-presets')); }
}
