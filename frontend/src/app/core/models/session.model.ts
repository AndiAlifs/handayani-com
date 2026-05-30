export type SessionStatus = 'scheduled' | 'completed' | 'cancelled';

export interface Session {
  id: number;
  studentId: number;
  studentName: string;
  instructorId: number;
  instructorName: string;
  courseId: number;
  courseName: string;
  startTime: Date;
  endTime: Date;
  status: SessionStatus;
  sessionNumber: number; // e.g. 3 of 10
  totalSessions: number;
  rawNotes?: string;
  aiAnalysis?: AiAnalysis;
}

export interface AiAnalysis {
  strengths: string[];
  weaknesses: string[];
  recommendedNextFocus: string;
  upsellRecommendation?: string;
}
