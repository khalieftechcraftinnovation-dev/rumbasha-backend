-- ============================================================================
-- Kartu ringkasan & grafik report quiz (BRD FR-RPT-001 s/d FR-RPT-005).
--
--   quiz_report_settings       1 baris: ambang "soal sering salah" (%) dan rentang default laporan
--                              (hari). Diubah admin lewat API, berlaku untuk semua tab report.
--   quiz_attempts_period_idx   pengerjaan Uji Kom / Tryout yang sudah submit per rentang tanggal
--                              submit (kartu, tren mingguan, sebaran nilai). Daily Quiz memakai
--                              index daily_quizzes_report_idx yang sudah ada.
-- ============================================================================

CREATE TABLE quiz_report_settings (
    id                      SMALLINT    PRIMARY KEY DEFAULT 1 CHECK (id = 1),  -- selalu 1 baris
    wrong_threshold_percent INT         NOT NULL DEFAULT 15 CHECK (wrong_threshold_percent BETWEEN 1 AND 100),
    default_range_days      INT         NOT NULL DEFAULT 60 CHECK (default_range_days BETWEEN 7 AND 366),
    updated_by              UUID        REFERENCES users (id),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Nilai awal sesuai desain: ambang 15%, rentang ±2 bulan (contoh desain 10 Sep - 9 Nov).
INSERT INTO quiz_report_settings (id) VALUES (1);

COMMENT ON TABLE  quiz_report_settings                         IS 'Pengaturan report quiz (selalu 1 baris, id = 1)';
COMMENT ON COLUMN quiz_report_settings.wrong_threshold_percent IS 'Soal "sering salah" = salah oleh LEBIH DARI persentase ini dari yang mengerjakan';
COMMENT ON COLUMN quiz_report_settings.default_range_days      IS 'Rentang laporan bawaan (hari, berakhir hari ini) kalau date_from/date_to tidak diisi';

CREATE INDEX quiz_attempts_period_idx
    ON quiz_attempts (submitted_at)
    INCLUDE (quiz_id, student_id, score)
    WHERE submitted_at IS NOT NULL;
