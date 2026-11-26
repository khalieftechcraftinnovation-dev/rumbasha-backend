CREATE TABLE rumath_sessions (
    id              UUID         PRIMARY KEY,
    student_id      UUID         NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    challenge_type  VARCHAR(30)  NOT NULL CHECK (challenge_type IN ('hitung_cepat', 'hitung_bersusun')),
    operation       VARCHAR(1)   NOT NULL CHECK (operation IN ('+', '-')),
    level           VARCHAR(10)  NOT NULL CHECK (level IN ('mudah', 'sedang', 'sulit')),
    total_questions SMALLINT     NOT NULL DEFAULT 10 CHECK (total_questions = 10),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX rumath_sessions_student_history_idx
    ON rumath_sessions (student_id, created_at DESC, id DESC);

CREATE TABLE rumath_questions (
    id              UUID         PRIMARY KEY,
    session_id      UUID         NOT NULL REFERENCES rumath_sessions (id) ON DELETE CASCADE,
    question_number SMALLINT     NOT NULL CHECK (question_number BETWEEN 1 AND 10),
    operand_a       INT          NOT NULL CHECK (operand_a BETWEEN 1 AND 1000),
    operand_b       INT          NOT NULL CHECK (operand_b BETWEEN 1 AND 1000),
    answer          INT          NOT NULL CHECK (answer >= 0),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT rumath_questions_session_number_unique UNIQUE (session_id, question_number)
);
