export type StudentStatus = 'lead' | 'active' | 'completed';

export interface StudentCrm {
  id: number;
  name: string;
  phone: string;
  courseId: number;
  courseName: string;
  status: StudentStatus;
  progressScore: number; // 0-100
  notes: string;
  createdAt: Date;
}
