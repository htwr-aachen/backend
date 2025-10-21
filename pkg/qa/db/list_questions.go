package db

import (
	"context"
	"fmt"
	"time"

	"github.com/htwr-aachen/backend/pkg/qa/schema"
	"github.com/rs/zerolog/log"
)

func (db *DB) ListQuestions(ctx context.Context, approvedFilter schema.ApprovedFilter, beforeId uint32, limit int) ([]schema.Question, error) {
	if err := validateApprovedFilter(approvedFilter); err != nil {
		log.Err(err).Msg("invalid approved filter")
		return nil, err
	}

	query := `
SELECT
	q.id as question_id,
	q.title as question_title,
	q.description as question_description,
	q.approved as question_approved,
	q.created_at as question_created_at,
	q.priority as question_priority
FROM
	questions q
WHERE
	q.id < $1
	AND (
		($2 = 0 AND q.approved IS TRUE) OR
		($2 = 1 AND q.approved IS FALSE) OR
		$2 = 2
	)
ORDER BY q.id DESC
LIMIT $3;
`
	rows, err := db.db.Query(ctx, query, beforeId, approvedFilter, limit)
	if err != nil {
		return nil, fmt.Errorf("querying questions: %w", err)
	}
	defer rows.Close()

	questions := make([]schema.Question, 0)

	for rows.Next() {
		var q schema.Question
		if err := rows.Scan(&q.Id, &q.Title, &q.Description, &q.Approved, &q.CreatedAt, &q.Priority); err != nil {
			return nil, fmt.Errorf("scanning questions: %w", err)
		}

		log.Debug().Object("question", q).Uint32("before_id", beforeId).Int("limit", limit).Msg("scanned question from db")
		q.Answers = []schema.Answer{}
		questions = append(questions, q)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating question rows: %w", err)
	}

	return questions, nil
}
func (db *DB) ListUnapprovedQuestions(ctx context.Context, beforeId uint32, limit int) ([]schema.Question, error) {
	return db.ListQuestions(ctx, schema.FILTER_UNAPPROVED, beforeId, limit)
}

func (db *DB) ListApprovedQuestions(ctx context.Context, beforeId uint32, limit int) ([]schema.Question, error) {
	return db.ListQuestions(ctx, schema.FILTER_APPROVED, beforeId, limit)
}

func (db *DB) ListAnsweredApprovedQuestions(ctx context.Context, lastPriority int, beforeId uint32, limit int) ([]schema.Question, error) {
	query := `
SELECT
    q.id AS question_id,
    q.approved AS question_approved,
    q.title AS question_title,
    q.description_rendered AS question_description,
    q.created_at AS question_created_at,
    q.priority AS question_priority,
    COALESCE(
        (
            SELECT COUNT(*)
            FROM deletion_requests dr_sub
            WHERE dr_sub.entity_type = 'question'
              AND dr_sub.entity_id = q.id
              AND dr_sub.status = 'pending'
        ),
        0
    ) AS question_deletion_requests_count,
    a.answer_id,
    a.answer_approved,
    a.answer_question_id,
    a.answer_text,
    a.answer_known_since,
    a.answer_created_at,
    a.answer_priority,
    a.answer_deletion_requests_count
FROM
    questions q
INNER JOIN LATERAL (
    SELECT
        ans.id AS answer_id,
        ans.approved AS answer_approved,
        ans.question_id AS answer_question_id,
        ans.answer_rendered AS answer_text,
        ans.known_since AS answer_known_since,
        ans.created_at AS answer_created_at,
        ans.priority AS answer_priority,
        COALESCE(
            (
                SELECT COUNT(*)
                FROM deletion_requests dr_ans_sub
                WHERE dr_ans_sub.entity_type = 'answer'
                  AND dr_ans_sub.entity_id = ans.id
                  AND dr_ans_sub.status = 'pending'
            ),
            0
        ) AS answer_deletion_requests_count
    FROM
        answers ans
    WHERE
        ans.question_id = q.id
        AND ans.approved IS TRUE
    ORDER BY
        ans.priority ASC,
        ans.known_since DESC NULLS LAST,
        ans.created_at DESC
    LIMIT 1
) a ON TRUE
WHERE
    q.approved IS TRUE
    AND (
        q.priority < $1
        OR (q.priority = $1 AND q.id < $2)
    )
ORDER BY
    q.priority DESC,
    q.id DESC
LIMIT $3;
    `

	// Update function call to pass three parameters: lastPriority, beforeId, limit
	rows, err := db.db.Query(ctx, query, lastPriority, beforeId, limit)
	if err != nil {
		return nil, fmt.Errorf("querying answered questions with answer: %w", err)
	}

	defer rows.Close()

	questions := make([]schema.Question, 0)

	for rows.Next() {
		var q schema.Question
		var a schema.Answer

		// Update the Scan to include the new deletion_requests_count column
		if err := rows.Scan(
			&q.Id, &q.Approved, &q.Title, &q.Description, &q.CreatedAt, &q.Priority,
			&q.DeletionRequestsCount,
			&a.Id, &a.Approved, &a.QuestionId, &a.Answer, &a.KnownSince, &a.CreatedAt, &a.Priority, &a.DeletionRequestsCount,
		); err != nil {
			return nil, fmt.Errorf("scanning questions: %w", err)
		}

		log.Trace().Object("question", q).Uint32("before_id", beforeId).Int("limit", limit).Msg("fetched answered approved questions from db")

		// The original code was appending the single answer 'a' to a new slice on every loop.
		// It's likely intended to have a single answer, which is what the query provides.
		q.Answers = append(q.Answers, a)
		questions = append(questions, q)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating question rows: %w", err)
	}

	return questions, nil
}

func (db *DB) ListUnansweredApprovedQuestions(ctx context.Context, lastPriority int, beforeId uint32, limit int) ([]schema.Question, error) {
	query := `
SELECT
    q.id AS question_id,
    q.title AS question_title,
    q.description_rendered AS question_description,
    q.created_at AS question_created_at,
    q.priority AS question_priority,
    COALESCE(dr.request_count, 0) AS deletion_requests_count
FROM
    questions q
LEFT JOIN LATERAL (
    SELECT
        COUNT(*) AS request_count
    FROM
        deletion_requests dr_sub
    WHERE
        dr_sub.entity_type = 'question'
        AND dr_sub.entity_id = q.id
        AND dr_sub.status = 'pending'
) dr ON TRUE
WHERE
    q.approved IS TRUE
    AND NOT EXISTS (SELECT 1 FROM answers a WHERE a.question_id = q.id AND a.approved IS TRUE)
    AND (
        q.priority < $1
        OR (q.priority = $1 AND q.id < $2)
    )
ORDER BY
    q.priority DESC,
    q.id DESC
LIMIT $3;
    `
	// Note: The parameters map to $1, $2, and $3 in the query.
	rows, err := db.db.Query(ctx, query, lastPriority, beforeId, limit)
	if err != nil {
		return nil, fmt.Errorf("querying unanswered questions: %w", err)
	}

	defer rows.Close()

	questions := make([]schema.Question, 0)

	for rows.Next() {
		var q schema.Question

		// SCANNED FIELDS: question_id, question_title, question_description, question_created_at, question_priority, deletion_requests_count
		if err := rows.Scan(&q.Id, &q.Title, &q.Description, &q.CreatedAt, &q.Priority, &q.DeletionRequestsCount); err != nil {
			return nil, fmt.Errorf("scanning questions: %w", err)
		}
		log.Trace().Object("question", q).Uint32("before_id", beforeId).Int("limit", limit).Msg("fetched unanswered approved questions from db")

		q.Answers = []schema.Answer{}
		questions = append(questions, q)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating question rows: %w", err)
	}

	return questions, nil
}

func (db *DB) ListQuestionsWithAnswers(ctx context.Context, approved bool, orderBy string, limit int) ([]schema.Question, error) {
	//ascending := orderBy == "asc"

	questionIDsQuery := `
		SELECT DISTINCT q.id, q.created_at
		FROM questions q
		JOIN answers a ON q.id = a.question_id
		WHERE a.approved = $1
		ORDER BY q.created_at ASC
		LIMIT $2
	`
	rows, err := db.db.Query(ctx, questionIDsQuery, approved, limit)
	if err != nil {
		return nil, fmt.Errorf("querying question IDs: %w", err)
	}
	defer rows.Close()

	var questionIDs []uint32
	for rows.Next() {
		var id uint32
		var created_at time.Time
		if err := rows.Scan(&id, &created_at); err != nil {
			return nil, fmt.Errorf("scanning question ID: %w", err)
		}
		questionIDs = append(questionIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating question IDs: %w", err)
	}

	if len(questionIDs) == 0 {
		return []schema.Question{}, nil
	}

	questions, err := db.getQuestions(ctx, questionIDs)
	if err != nil {
		return nil, fmt.Errorf("getting questions: %w", err)
	}

	approvedFilter := schema.FILTER_UNAPPROVED
	if approved {
		approvedFilter = schema.FILTER_APPROVED
	}
	answers, err := db.ListAnswers(ctx, questionIDs, approvedFilter, 99999999, 10000)
	if err != nil {
		return nil, fmt.Errorf("getting answers: %w", err)
	}

	for i := range questions {
		if answerList, ok := answers[questions[i].Id]; ok {
			questions[i].Answers = answerList
		}
	}

	return questions, nil
}
