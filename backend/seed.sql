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

INSERT INTO instructors (id, name, gender, age, vehicle, transmission) VALUES
(1, 'Pak Bambang', 'Laki-laki', 45, 'Avanza', 'Manual'),
(2, 'Bu Sari', 'Perempuan', 38, 'Xenia', 'Matic'),
(3, 'Pak Hendro', 'Laki-laki', 52, 'Avanza', 'Manual'),
(4, 'Pak Dimas', 'Laki-laki', 35, 'Xenia', 'Matic');

-- A couple of representative booked / holiday slots; the rest default to Tersedia
-- and are created lazily by the API when an admin first edits the matrix.
INSERT INTO schedules (instructor_id, day, time_slot, status) VALUES
(1, 'Senin', '09.00 - 12.00', 'Andi S.'),
(1, 'Rabu',  '13.00 - 15.00', 'Rina M.'),
(1, 'Minggu','09.00 - 12.00', 'Libur'),
(2, 'Selasa','09.00 - 12.00', 'Dewi P.'),
(2, 'Kamis', '15.00 - 17.00', 'Budi K.'),
(3, 'Senin', '13.00 - 15.00', 'Tia L.'),
(3, 'Jumat', '09.00 - 12.00', 'Reza F.'),
(4, 'Rabu',  '09.00 - 12.00', 'Yuni W.');

INSERT INTO mechanisms (requirement_name, issuing_body, cost, notes, sort_order) VALUES
('Sertifikat Mengemudi', 'LPK YPA Handayani', 0, 'Diberikan setelah lulus kursus', 1),
('Surat Keterangan Sehat', 'Puskesmas / RS', 50000, 'Berlaku 1 bulan sejak diterbitkan', 2),
('Tes Psikologi', 'Lembaga Psikologi Terakreditasi', 150000, 'Wajib untuk SIM A & SIM C', 3),
('Formulir Permohonan SIM', 'Satpas / Polres', 100000, 'Bawa KTP asli & fotokopi', 4),
('Biaya PNBP SIM Baru', 'Satpas / Polres', 120000, 'SIM A: Rp120.000 | SIM C: Rp100.000', 5);
