import { Course } from '../models/course.model';
import { Instructor, ScheduleSlot } from '../models/instructor.model';
import { Mechanism } from '../models/mechanism.model';
import { StudentCrm } from '../models/student-crm.model';
import { Session, AiAnalysis } from '../models/session.model';
import { WhatsAppStatus, WhatsAppMessage } from '../models/whatsapp.model';

export type { Course, Instructor, ScheduleSlot, Mechanism, StudentCrm, Session, AiAnalysis };
export type { WhatsAppStatus, WhatsAppMessage };

// ──────────────────────────────────────────
// WHATSAPP (WAHA) — offline fallbacks
// ──────────────────────────────────────────
export const MOCK_WHATSAPP_STATUS: WhatsAppStatus = {
  sessionName: 'default',
  status: 'STOPPED',
  phoneNumber: '',
  pairedAt: null,
  lastSyncedAt: new Date().toISOString(),
};

export const MOCK_WHATSAPP_MESSAGES: WhatsAppMessage[] = [];

// ──────────────────────────────────────────
// COURSES
// ──────────────────────────────────────────
export const MOCK_COURSES: Course[] = [
  { id: 1, category: 'Mengemudi', programType: 'Reguler', specifics: 'Manual - Avanza/Xenia', duration: '10 Pertemuan', price: 1500000, registrationFee: 200000, remarks: 'Belum termasuk SIM & Sertifikat' },
  { id: 2, category: 'Mengemudi', programType: 'Reguler', specifics: 'Matic - Avanza/Xenia', duration: '10 Pertemuan', price: 1600000, registrationFee: 200000, remarks: 'Belum termasuk SIM & Sertifikat' },
  { id: 3, category: 'Mengemudi', programType: 'Weekend', specifics: 'Manual - Avanza/Xenia', duration: '10 Pertemuan', price: 1800000, registrationFee: 200000, remarks: 'Belum termasuk SIM & Sertifikat' },
  { id: 4, category: 'Mengemudi', programType: 'Weekend', specifics: 'Matic - Avanza/Xenia', duration: '10 Pertemuan', price: 1900000, registrationFee: 200000, remarks: 'Belum termasuk SIM & Sertifikat' },
  { id: 5, category: 'Mengemudi', programType: 'Reguler', specifics: 'Hybrid - Avanza/Xenia', duration: '10 Pertemuan', price: 2000000, registrationFee: 200000, remarks: 'Manual + Matic' },
  { id: 6, category: 'Menjahit', programType: 'Reguler', specifics: 'Dasar', duration: '3 Bulan', price: 1200000, registrationFee: 150000, remarks: 'Termasuk bahan praktik' },
  { id: 7, category: 'Menjahit', programType: 'Reguler', specifics: 'Mahir', duration: '6 Bulan', price: 2200000, registrationFee: 150000, remarks: 'Termasuk bahan praktik & sertifikat' },
  { id: 8, category: 'Komputer', programType: 'Reguler', specifics: 'Microsoft Office', duration: '2 Bulan', price: 800000, registrationFee: 100000, remarks: 'Word, Excel, PowerPoint' },
  { id: 9, category: 'Komputer', programType: 'Reguler', specifics: 'Desain Grafis', duration: '3 Bulan', price: 1500000, registrationFee: 100000, remarks: 'Photoshop, Illustrator, CorelDraw' },
  { id: 10, category: 'Bahasa Inggris', programType: 'Reguler', specifics: 'Basic Level', duration: '3 Bulan', price: 900000, registrationFee: 100000, remarks: 'Speaking & Grammar' },
  { id: 11, category: 'Bahasa Inggris', programType: 'Reguler', specifics: 'Intermediate Level', duration: '3 Bulan', price: 1100000, registrationFee: 100000, remarks: 'Conversation & Writing' },
  { id: 12, category: 'Bahasa Mandarin', programType: 'Reguler', specifics: 'HSK 1', duration: '3 Bulan', price: 1200000, registrationFee: 150000, remarks: 'Dasar percakapan & karakter' },
  { id: 13, category: 'Bahasa Mandarin', programType: 'Reguler', specifics: 'HSK 2', duration: '3 Bulan', price: 1400000, registrationFee: 150000, remarks: 'Lanjutan percakapan & tulisan' },
];

// ──────────────────────────────────────────
// INSTRUCTORS
// ──────────────────────────────────────────
const DAYS = ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu', 'Minggu'];
const TIME_SLOTS = ['09.00 - 12.00', '13.00 - 15.00', '15.00 - 17.00'];

function generateSchedule(
  bookedSlots: { day: string; time: string; student: string }[],
  holidays: string[]
): ScheduleSlot[] {
  const slots: ScheduleSlot[] = [];
  for (const day of DAYS) {
    for (const time of TIME_SLOTS) {
      const booked = bookedSlots.find(b => b.day === day && b.time === time);
      const isHoliday = holidays.includes(day);
      slots.push({
        day,
        timeSlot: time,
        status: booked ? booked.student : isHoliday ? 'Libur' : 'Tersedia'
      });
    }
  }
  return slots;
}

export const MOCK_INSTRUCTORS: Instructor[] = [
  {
    id: 1, name: 'Pak Bambang', gender: 'Laki-laki', age: 45, vehicle: 'Avanza', transmission: 'Manual',
    schedule: generateSchedule(
      [{ day: 'Senin', time: '09.00 - 12.00', student: 'Andi S.' }, { day: 'Rabu', time: '13.00 - 15.00', student: 'Rina M.' }],
      ['Minggu']
    )
  },
  {
    id: 2, name: 'Bu Sari', gender: 'Perempuan', age: 38, vehicle: 'Xenia', transmission: 'Matic',
    schedule: generateSchedule(
      [{ day: 'Selasa', time: '09.00 - 12.00', student: 'Dewi P.' }, { day: 'Kamis', time: '15.00 - 17.00', student: 'Budi K.' }],
      ['Minggu']
    )
  },
  {
    id: 3, name: 'Pak Hendro', gender: 'Laki-laki', age: 52, vehicle: 'Avanza', transmission: 'Manual',
    schedule: generateSchedule(
      [{ day: 'Senin', time: '13.00 - 15.00', student: 'Tia L.' }, { day: 'Jumat', time: '09.00 - 12.00', student: 'Reza F.' }],
      ['Minggu']
    )
  },
  {
    id: 4, name: 'Pak Dimas', gender: 'Laki-laki', age: 35, vehicle: 'Xenia', transmission: 'Matic',
    schedule: generateSchedule(
      [{ day: 'Rabu', time: '09.00 - 12.00', student: 'Yuni W.' }],
      ['Sabtu', 'Minggu']
    )
  },
];

// ──────────────────────────────────────────
// MECHANISMS
// ──────────────────────────────────────────
export const MOCK_MECHANISMS: Mechanism[] = [
  { id: 1, requirementName: 'Sertifikat Mengemudi', issuingBody: 'LPK YPA Handayani', cost: 0, notes: 'Diberikan setelah lulus kursus' },
  { id: 2, requirementName: 'Surat Keterangan Sehat', issuingBody: 'Puskesmas / RS', cost: 50000, notes: 'Berlaku 1 bulan sejak diterbitkan' },
  { id: 3, requirementName: 'Tes Psikologi', issuingBody: 'Lembaga Psikologi Terakreditasi', cost: 150000, notes: 'Wajib untuk SIM A & SIM C' },
  { id: 4, requirementName: 'Formulir Permohonan SIM', issuingBody: 'Satpas / Polres', cost: 100000, notes: 'Bawa KTP asli & fotokopi' },
  { id: 5, requirementName: 'Biaya PNBP SIM Baru', issuingBody: 'Satpas / Polres', cost: 120000, notes: 'SIM A: Rp120.000 | SIM C: Rp100.000' },
];

// ──────────────────────────────────────────
// CRM DATA
// ──────────────────────────────────────────
export const MOCK_STUDENTS_CRM: StudentCrm[] = [
  { id: 1, name: 'Andi Setiawan', phone: '08123456789', courseId: 1, courseName: 'Manual - Avanza/Xenia', status: 'active', progressScore: 60, notes: 'Sudah menguasai pengereman. Perlu latihan parkir paralel.', createdAt: new Date('2026-05-01') },
  { id: 2, name: 'Rina Marlina', phone: '08234567890', courseId: 2, courseName: 'Matic - Avanza/Xenia', status: 'active', progressScore: 80, notes: 'Progres sangat baik. Siap ujian minggu depan.', createdAt: new Date('2026-05-05') },
  { id: 3, name: 'Dewi Pertiwi', phone: '08345678901', courseId: 1, courseName: 'Manual - Avanza/Xenia', status: 'lead', progressScore: 0, notes: 'Tertarik kursus manual. Menunggu konfirmasi jadwal.', createdAt: new Date('2026-05-20') },
  { id: 4, name: 'Budi Kurniawan', phone: '08456789012', courseId: 4, courseName: 'Matic Weekend - Avanza/Xenia', status: 'completed', progressScore: 100, notes: 'Lulus. SIM A diterbitkan 15 Mei 2026.', createdAt: new Date('2026-04-01') },
  { id: 5, name: 'Tia Lestari', phone: '08567890123', courseId: 1, courseName: 'Manual - Avanza/Xenia', status: 'active', progressScore: 40, notes: 'Masih kesulitan kopling. Perlu fokus pada perpindahan gigi.', createdAt: new Date('2026-05-10') },
  { id: 6, name: 'Yuni Wahyuni', phone: '08678901234', courseId: 2, courseName: 'Matic - Avanza/Xenia', status: 'lead', progressScore: 0, notes: 'Chatbot lead — ingin mendaftar kursus matic.', createdAt: new Date('2026-05-28') },
];

// ──────────────────────────────────────────
// SESSION DATA
// ──────────────────────────────────────────
const today = new Date();

export const MOCK_SESSIONS: Session[] = [
  {
    id: 1, studentId: 1, studentName: 'Andi Setiawan', instructorId: 1, instructorName: 'Pak Bambang',
    courseId: 1, courseName: 'Manual - Avanza/Xenia',
    startTime: new Date(today.getFullYear(), today.getMonth(), today.getDate(), 9, 0),
    endTime: new Date(today.getFullYear(), today.getMonth(), today.getDate(), 12, 0),
    status: 'scheduled', sessionNumber: 7, totalSessions: 10,
  },
  {
    id: 2, studentId: 2, studentName: 'Rina Marlina', instructorId: 2, instructorName: 'Bu Sari',
    courseId: 2, courseName: 'Matic - Avanza/Xenia',
    startTime: new Date(today.getFullYear(), today.getMonth(), today.getDate(), 13, 0),
    endTime: new Date(today.getFullYear(), today.getMonth(), today.getDate(), 15, 0),
    status: 'completed', sessionNumber: 9, totalSessions: 10,
    rawNotes: 'Rina sudah sangat baik dalam mengemudi di jalan raya. Parkir mundur masih perlu sedikit penyesuaian, tapi kopling sudah sempurna.',
    aiAnalysis: {
      strengths: ['Pengendalian kemudi di jalan raya', 'Kontrol kecepatan yang baik', 'Disiplin rambu-rambu lalu lintas'],
      weaknesses: ['Parkir mundur masih kurang presisi'],
      recommendedNextFocus: 'Latihan parkir mundur dan parkir paralel intensif untuk sesi terakhir.',
      upsellRecommendation: undefined
    }
  },
  {
    id: 3, studentId: 5, studentName: 'Tia Lestari', instructorId: 1, instructorName: 'Pak Bambang',
    courseId: 1, courseName: 'Manual - Avanza/Xenia',
    startTime: new Date(today.getFullYear(), today.getMonth(), today.getDate() + 1, 13, 0),
    endTime: new Date(today.getFullYear(), today.getMonth(), today.getDate() + 1, 15, 0),
    status: 'scheduled', sessionNumber: 4, totalSessions: 10,
    rawNotes: 'Tia masih kesulitan dengan kopling saat di tanjakan. Perpindahan gigi 1 ke 2 masih ragu-ragu.',
    aiAnalysis: {
      strengths: ['Pengereman sudah baik', 'Steering control mulai membaik'],
      weaknesses: ['Penggunaan kopling di tanjakan', 'Perpindahan gigi 1→2 masih ragu'],
      recommendedNextFocus: 'Latihan khusus tanjakan dengan teknik setengah kopling dan hill-start assist.',
      upsellRecommendation: 'Siswa menunjukkan kesulitan signifikan pada sesi ke-4. Direkomendasikan penambahan 2 sesi khusus tanjakan.'
    }
  },
];
