-- YPA Handayani Knowledge Base — MySQL schema
-- Fulfils PRD §5 (Core Entities) and WBS Tasks 2.1, 3.1, 4.1.

CREATE DATABASE IF NOT EXISTS handayani
  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE handayani;

-- ── Courses & Pricing (Epic 2) ──────────────────────────────
CREATE TABLE IF NOT EXISTS courses (
  id               INT AUTO_INCREMENT PRIMARY KEY,
  category         VARCHAR(64)  NOT NULL,
  program_type     VARCHAR(32)  NOT NULL,
  specifics        VARCHAR(255) NOT NULL,
  duration         VARCHAR(64)  NOT NULL,
  price            BIGINT       NOT NULL DEFAULT 0,
  registration_fee BIGINT       NOT NULL DEFAULT 0,
  remarks          VARCHAR(255) NOT NULL DEFAULT '',
  created_at       TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
  updated_at       TIMESTAMP    DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- ── Instructors & Schedules (Epic 3) ────────────────────────
CREATE TABLE IF NOT EXISTS instructors (
  id           INT AUTO_INCREMENT PRIMARY KEY,
  name         VARCHAR(128) NOT NULL,
  gender       VARCHAR(16)  NOT NULL,
  age          INT          NOT NULL,
  vehicle      VARCHAR(64)  NOT NULL,
  transmission VARCHAR(32)  NOT NULL,
  created_at   TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
  updated_at   TIMESTAMP    DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS schedules (
  id            INT AUTO_INCREMENT PRIMARY KEY,
  instructor_id INT          NOT NULL,
  day           VARCHAR(16)  NOT NULL,
  time_slot     VARCHAR(32)  NOT NULL,
  status        VARCHAR(128) NOT NULL DEFAULT 'Tersedia',
  CONSTRAINT fk_schedule_instructor
    FOREIGN KEY (instructor_id) REFERENCES instructors(id) ON DELETE CASCADE,
  UNIQUE KEY uq_slot (instructor_id, day, time_slot)
) ENGINE=InnoDB;

-- ── Administrative Mechanisms (Epic 4) ──────────────────────
CREATE TABLE IF NOT EXISTS mechanisms (
  id               INT AUTO_INCREMENT PRIMARY KEY,
  requirement_name VARCHAR(128) NOT NULL,
  issuing_body     VARCHAR(128) NOT NULL,
  cost             BIGINT       NOT NULL DEFAULT 0,
  notes            VARCHAR(255) NOT NULL DEFAULT '',
  sort_order       INT          NOT NULL DEFAULT 0
) ENGINE=InnoDB;
