-- Seed data mirroring the Angular mock-data so the public landing page and the
-- RAG knowledge-sync endpoint have realistic content out of the box.
USE handayani;

INSERT INTO courses (category, program_type, specifics, duration, price, registration_fee, remarks) VALUES
('Mengemudi', 'Reguler', 'Manual - Avanza/Xenia', '10 Pertemuan', 1500000, 200000, 'Belum termasuk SIM & Sertifikat'),
('Mengemudi', 'Reguler', 'Matic - Avanza/Xenia', '10 Pertemuan', 1600000, 200000, 'Belum termasuk SIM & Sertifikat'),
('Mengemudi', 'Weekend', 'Manual - Avanza/Xenia', '10 Pertemuan', 1800000, 200000, 'Belum termasuk SIM & Sertifikat'),
('Mengemudi', 'Weekend', 'Matic - Avanza/Xenia', '10 Pertemuan', 1900000, 200000, 'Belum termasuk SIM & Sertifikat'),
('Mengemudi', 'Reguler', 'Hybrid - Avanza/Xenia', '10 Pertemuan', 2000000, 200000, 'Manual + Matic'),
('Menjahit', 'Reguler', 'Dasar', '3 Bulan', 1200000, 150000, 'Termasuk bahan praktik'),
('Menjahit', 'Reguler', 'Mahir', '6 Bulan', 2200000, 150000, 'Termasuk bahan praktik & sertifikat'),
('Komputer', 'Reguler', 'Microsoft Office', '2 Bulan', 800000, 100000, 'Word, Excel, PowerPoint'),
('Komputer', 'Reguler', 'Desain Grafis', '3 Bulan', 1500000, 100000, 'Photoshop, Illustrator, CorelDraw'),
('Bahasa Inggris', 'Reguler', 'Basic Level', '3 Bulan', 900000, 100000, 'Speaking & Grammar'),
('Bahasa Inggris', 'Reguler', 'Intermediate Level', '3 Bulan', 1100000, 100000, 'Conversation & Writing'),
('Bahasa Mandarin', 'Reguler', 'HSK 1', '3 Bulan', 1200000, 150000, 'Dasar percakapan & karakter'),
('Bahasa Mandarin', 'Reguler', 'HSK 2', '3 Bulan', 1400000, 150000, 'Lanjutan percakapan & tulisan');

INSERT INTO mechanisms (requirement_name, issuing_body, cost, notes, sort_order) VALUES
('Sertifikat Mengemudi', 'LPK YPA Handayani', 0, 'Diberikan setelah lulus kursus', 1),
('Surat Keterangan Sehat', 'Puskesmas / RS', 50000, 'Berlaku 1 bulan sejak diterbitkan', 2),
('Tes Psikologi', 'Lembaga Psikologi Terakreditasi', 150000, 'Wajib untuk SIM A & SIM C', 3),
('Formulir Permohonan SIM', 'Satpas / Polres', 100000, 'Bawa KTP asli & fotokopi', 4),
('Biaya PNBP SIM Baru', 'Satpas / Polres', 120000, 'SIM A: Rp120.000 | SIM C: Rp100.000', 5);

INSERT INTO students_crm (id, name, phone, course_id, course_name, status, progress_score, notes, created_at) VALUES
(1, 'Andi Setiawan', '08123456789', 1, 'Manual - Avanza/Xenia', 'active', 60, 'Sudah menguasai pengereman. Perlu latihan parkir paralel.', '2026-05-01'),
(2, 'Rina Marlina', '08234567890', 2, 'Matic - Avanza/Xenia', 'active', 80, 'Progres sangat baik. Siap ujian minggu depan.', '2026-05-05'),
(3, 'Dewi Pertiwi', '08345678901', 1, 'Manual - Avanza/Xenia', 'lead', 0, 'Tertarik kursus manual. Menunggu konfirmasi jadwal.', '2026-05-20'),
(4, 'Budi Kurniawan', '08456789012', 4, 'Matic Weekend - Avanza/Xenia', 'completed', 100, 'Lulus. SIM A diterbitkan 15 Mei 2026.', '2026-04-01'),
(5, 'Tia Lestari', '08567890123', 1, 'Manual - Avanza/Xenia', 'active', 40, 'Masih kesulitan kopling. Perlu fokus pada perpindahan gigi.', '2026-05-10'),
(6, 'Yuni Wahyuni', '08678901234', 2, 'Matic - Avanza/Xenia', 'lead', 0, 'Chatbot lead — ingin mendaftar kursus matic.', '2026-05-28');

INSERT INTO sessions
(id, student_id, student_name, instructor_id, instructor_name, course_id, course_name,
 start_time, end_time, status, session_number, total_sessions, raw_notes,
 ai_strengths, ai_weaknesses, ai_recommended_next_focus, ai_upsell_recommendation) VALUES
(1, 1, 'Andi Setiawan', 1, 'Pak Bambang', 1, 'Manual - Avanza/Xenia',
 '2026-06-13 09:00:00', '2026-06-13 12:00:00', 'scheduled', 7, 10, NULL,
 NULL, NULL, NULL, NULL),
(2, 2, 'Rina Marlina', 2, 'Bu Sari', 2, 'Matic - Avanza/Xenia',
 '2026-06-13 13:00:00', '2026-06-13 15:00:00', 'completed', 9, 10,
 'Rina sudah sangat baik dalam mengemudi di jalan raya. Parkir mundur masih perlu sedikit penyesuaian, tapi kopling sudah sempurna.',
 '["Pengendalian kemudi di jalan raya", "Kontrol kecepatan yang baik", "Disiplin rambu-rambu lalu lintas"]',
 '["Parkir mundur masih kurang presisi"]',
 'Latihan parkir mundur dan parkir paralel intensif untuk sesi terakhir.', NULL),
(3, 5, 'Tia Lestari', 1, 'Pak Bambang', 1, 'Manual - Avanza/Xenia',
 '2026-06-14 13:00:00', '2026-06-14 15:00:00', 'scheduled', 4, 10,
 'Tia masih kesulitan dengan kopling saat di tanjakan. Perpindahan gigi 1 ke 2 masih ragu-ragu.',
 '["Pengereman sudah baik", "Steering control mulai membaik"]',
 '["Penggunaan kopling di tanjakan", "Perpindahan gigi 1→2 masih ragu"]',
 'Latihan khusus tanjakan dengan teknik setengah kopling dan hill-start assist.',
 'Siswa menunjukkan kesulitan signifikan pada sesi ke-4. Direkomendasikan penambahan 2 sesi khusus tanjakan.');
