-- ============================================================================
-- Daily Quiz: quiz latihan yang dibuat (generate) sendiri oleh siswa dari soal acak.
-- (BRD FR-QUIZ-012 s/d FR-QUIZ-014)
--
-- Dipisah dari tabel quizzes karena sifatnya berbeda: milik 1 siswa, tanpa jadwal event,
-- tanpa kode akses, tanpa timer, dan tidak boleh ikut aturan bentrok jadwal / list quiz admin.
-- ============================================================================

CREATE TABLE daily_quizzes (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id  UUID        NOT NULL REFERENCES users (id),  -- siswa yang membuat

    -- Pilihan siswa. Wajib sampai bidang materi; bab, sub bab, topik opsional (berurutan).
    education_level_id UUID NOT NULL REFERENCES master_data_items (id),
    grade_id           UUID NOT NULL REFERENCES master_data_items (id),
    subject_id         UUID NOT NULL REFERENCES master_data_items (id),
    subject_field_id   UUID NOT NULL REFERENCES master_data_items (id),
    chapter_id         UUID REFERENCES master_data_items (id),
    subchapter_id      UUID REFERENCES master_data_items (id),
    topic_id           UUID REFERENCES master_data_items (id),

    difficulty      VARCHAR(10) NOT NULL CHECK (difficulty IN ('easy', 'medium', 'hard')),  -- level
    total_questions INT         NOT NULL CHECK (total_questions IN (5, 10, 15)),

    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT daily_quizzes_subchapter_needs_chapter CHECK (subchapter_id IS NULL OR chapter_id IS NOT NULL),
    CONSTRAINT daily_quizzes_topic_needs_subchapter   CHECK (topic_id IS NULL OR subchapter_id IS NOT NULL),
    CONSTRAINT daily_quizzes_field_subject_fkey FOREIGN KEY (subject_field_id, subject_id) REFERENCES master_data_items (id, parent_id),
    CONSTRAINT daily_quizzes_grade_level_fkey   FOREIGN KEY (grade_id, education_level_id) REFERENCES master_data_items (id, parent_id)
);

-- Riwayat Daily Quiz siswa (terbaru dulu) & hitung pemakaian per hari (jatah gratis, nanti).
CREATE INDEX daily_quizzes_student_created_idx ON daily_quizzes (student_id, created_at DESC);

COMMENT ON TABLE  daily_quizzes            IS 'Daily Quiz yang di-generate siswa: soal acak sesuai pilihan siswa, tanpa kode akses & timer';
COMMENT ON COLUMN daily_quizzes.difficulty IS 'Level pilihan siswa = questions.difficulty (easy/medium/hard); tipe soal boleh campur';


CREATE TABLE daily_quiz_questions (
    daily_quiz_id UUID NOT NULL REFERENCES daily_quizzes (id) ON DELETE CASCADE,
    question_id   UUID NOT NULL REFERENCES questions (id),
    sort_order    INT  NOT NULL CHECK (sort_order > 0),  -- nomor soal, mulai 1

    PRIMARY KEY (daily_quiz_id, question_id),
    CONSTRAINT daily_quiz_questions_order_unique UNIQUE (daily_quiz_id, sort_order)
);


-- ----------------------------------------------------------------------------
-- Pengumuman quiz di dashboard siswa: quiz publish untuk 1 kelas, urut waktu mulai.
-- ----------------------------------------------------------------------------
CREATE INDEX quizzes_announcement_idx
    ON quizzes (grade_id, start_at)
    WHERE deleted_at IS NULL AND status = 'published';
