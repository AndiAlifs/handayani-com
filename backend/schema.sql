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

-- ── Administrative Mechanisms (Epic 4) ──────────────────────
CREATE TABLE IF NOT EXISTS mechanisms (
  id               INT AUTO_INCREMENT PRIMARY KEY,
  requirement_name VARCHAR(128) NOT NULL,
  issuing_body     VARCHAR(128) NOT NULL,
  cost             BIGINT       NOT NULL DEFAULT 0,
  notes            VARCHAR(255) NOT NULL DEFAULT '',
  sort_order       INT          NOT NULL DEFAULT 0
) ENGINE=InnoDB;

-- ── CRM (admin tooling) ─────────────────────────────────────
CREATE TABLE IF NOT EXISTS students_crm (
  id             INT AUTO_INCREMENT PRIMARY KEY,
  name           VARCHAR(128) NOT NULL,
  phone          VARCHAR(32)  NOT NULL,
  course_id      INT          NOT NULL,
  course_name    VARCHAR(128) NOT NULL,
  status         VARCHAR(16)  NOT NULL DEFAULT 'lead',
  progress_score INT          NOT NULL DEFAULT 0,
  notes          TEXT         NOT NULL,
  created_at     TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- ── Training Sessions + AI analysis (admin tooling) ─────────
CREATE TABLE IF NOT EXISTS sessions (
  id                        INT AUTO_INCREMENT PRIMARY KEY,
  student_id                INT          NOT NULL,
  student_name              VARCHAR(128) NOT NULL,
  instructor_id             INT          NOT NULL,
  instructor_name           VARCHAR(128) NOT NULL,
  course_id                 INT          NOT NULL,
  course_name               VARCHAR(128) NOT NULL,
  start_time                DATETIME     NOT NULL,
  end_time                  DATETIME     NOT NULL,
  status                    VARCHAR(16)  NOT NULL DEFAULT 'scheduled',
  session_number            INT          NOT NULL DEFAULT 1,
  total_sessions            INT          NOT NULL DEFAULT 10,
  raw_notes                 TEXT         NULL,
  ai_strengths              JSON         NULL,
  ai_weaknesses             JSON         NULL,
  ai_recommended_next_focus TEXT         NULL,
  ai_upsell_recommendation  TEXT         NULL
) ENGINE=InnoDB;
