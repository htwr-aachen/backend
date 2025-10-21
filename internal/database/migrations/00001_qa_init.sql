-- +goose Up
CREATE TABLE questions (
    id SERIAL PRIMARY KEY,
    approved BOOLEAN NOT NULL DEFAULT FALSE,
    approved_by VARCHAR(255),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    description_rendered TEXT,
    render_version INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    priority INT NOT NULL DEFAULT 0
);

CREATE TABLE answers (
    id SERIAL PRIMARY KEY,
    question_id INT NOT NULL REFERENCES questions (id) ON DELETE CASCADE,
    approved BOOLEAN NOT NULL DEFAULT FALSE,
    approved_by VARCHAR(255),
    answer TEXT NOT NULL,
    answer_rendered TEXT,
    render_version INT DEFAULT 0,
    priority INT NOT NULL DEFAULT 0,
    known_since TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE deletion_requests (
    id SERIAL PRIMARY KEY,
    entity_type VARCHAR(20) NOT NULL CHECK (
        entity_type IN ('question', 'answer')
    ),
    entity_id INT NOT NULL,
    reason VARCHAR(500),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (
        status IN ('pending', 'approved', 'rejected')
    ),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_answers_question_id ON answers (question_id);
CREATE INDEX idx_questions_created_at ON questions (created_at);
CREATE INDEX idx_answers_known_since ON answers (known_since);

-- For fast sorting and keyset pagination
CREATE INDEX idx_questions_priority_id ON questions (priority DESC, id DESC);
-- This covers the WHERE and ORDER BY clauses of the subquery entirely.
CREATE INDEX idx_answers_q_id_approved_sort ON answers (
    question_id, approved, priority ASC, known_since DESC, created_at DESC
);

-- 3. To optimize the count of pending deletion requests
CREATE INDEX idx_deletion_requests_entity_status ON deletion_requests (
    entity_type, entity_id, status
);

CREATE INDEX idx_entity_type_id ON deletion_requests (entity_type, entity_id);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION CHECK_DELETION_REQUEST_ENTITY()
RETURNS TRIGGER AS $func$
BEGIN
  IF NEW.entity_type = 'question' THEN
    IF NOT EXISTS (SELECT 1 FROM questions WHERE id = NEW.entity_id) THEN
      RAISE EXCEPTION 'Referenced question does not exist';
    END IF;
  ELSIF NEW.entity_type = 'answer' THEN
    IF NOT EXISTS (SELECT 1 FROM answers WHERE id = NEW.entity_id) THEN
      RAISE EXCEPTION 'Referenced answer does not exist';
    END IF;
  END IF;
  RETURN NEW;
END;
$func$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_check_deletion_request_entity
BEFORE INSERT OR UPDATE ON deletion_requests
FOR EACH ROW
EXECUTE FUNCTION CHECK_DELETION_REQUEST_ENTITY();
-- +goose Down


DROP TRIGGER IF EXISTS trg_check_deletion_request_entity ON deletion_requests;
DROP FUNCTION IF EXISTS check_deletion_request_entity();
DROP INDEX IF EXISTS idx_entity_type_id;
DROP TABLE IF EXISTS deletion_requests;

DROP TABLE IF EXISTS answers;
DROP TABLE IF EXISTS questions;
