export interface Course {
  id: number;
  category: string;
  programType: string;
  specifics: string;
  duration: string;
  price: number;
  registrationFee: number;
  remarks: string;
}

export interface Instructor {
  id: number;
  name: string;
  gender: string;
  age: number;
  vehicle: string;
  transmission: string;
  schedule: ScheduleSlot[];
}

export interface ScheduleSlot {
  day: string;
  timeSlot: string;
  status: string;
}

export interface Mechanism {
  id: number;
  requirementName: string;
  issuingBody: string;
  cost: number;
  notes: string;
}

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

const DAYS = ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu', 'Minggu'];
const TIME_SLOTS = ['09.00 - 12.00', '13.00 - 15.00', '15.00 - 17.00'];

function generateSchedule(bookedSlots: { day: string; time: string; student: string }[], holidays: string[]): ScheduleSlot[] {
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

export const MOCK_MECHANISMS: Mechanism[] = [
  { id: 1, requirementName: 'Sertifikat Mengemudi', issuingBody: 'LPK YPA Handayani', cost: 0, notes: 'Diberikan setelah lulus kursus' },
  { id: 2, requirementName: 'Surat Keterangan Sehat', issuingBody: 'Puskesmas / RS', cost: 50000, notes: 'Berlaku 1 bulan sejak diterbitkan' },
  { id: 3, requirementName: 'Tes Psikologi', issuingBody: 'Lembaga Psikologi Terakreditasi', cost: 150000, notes: 'Wajib untuk SIM A & SIM C' },
  { id: 4, requirementName: 'Formulir Permohonan SIM', issuingBody: 'Satpas / Polres', cost: 100000, notes: 'Bawa KTP asli & fotokopi' },
  { id: 5, requirementName: 'Ujian Teori SIM', issuingBody: 'Satpas / Polres', cost: 0, notes: 'Minimal skor kelulusan 70%' },
  { id: 6, requirementName: 'Ujian Praktik SIM', issuingBody: 'Satpas / Polres', cost: 0, notes: 'Tes mengemudi di area uji' },
  { id: 7, requirementName: 'Biaya Pembuatan SIM A', issuingBody: 'Satpas / Polres', cost: 120000, notes: 'SIM baru, berlaku 5 tahun' },
];
