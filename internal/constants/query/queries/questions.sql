-- name: CreateQuestion :one
INSERT INTO questions (
    user_id, question, status
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetQuestion :one
SELECT 
    q.*,
    u.name as user_name,
    r.name as responder_name
FROM questions q
LEFT JOIN users u ON q.user_id = u.id
LEFT JOIN users r ON q.responded_by = r.id
WHERE q.id = $1;

-- name: GetQuestions :many
SELECT 
    q.*,
    CASE WHEN $6::boolean THEN u.name ELSE NULL END as user_name,
    CASE WHEN $6::boolean THEN r.name ELSE NULL END as responder_name
FROM questions q
LEFT JOIN users u ON q.user_id = u.id
LEFT JOIN users r ON q.responded_by = r.id
WHERE 
    ($1::uuid IS NULL OR q.user_id = $1) AND
    ($2::varchar IS NULL OR q.status = $2) AND
    ($3::timestamptz IS NULL OR q.created_at >= $3) AND
    ($4::timestamptz IS NULL OR q.created_at <= $4)
ORDER BY q.created_at DESC
LIMIT $5;

-- name: GetQuestionsCount :one
SELECT COUNT(*) FROM questions q
WHERE 
    ($1::uuid IS NULL OR q.user_id = $1) AND
    ($2::varchar IS NULL OR q.status = $2) AND
    ($3::timestamptz IS NULL OR q.created_at >= $3) AND
    ($4::timestamptz IS NULL OR q.created_at <= $4);

-- name: GetQuestionsPaginated :many
SELECT 
    q.*,
    CASE WHEN $7::boolean THEN u.name ELSE NULL END as user_name,
    CASE WHEN $7::boolean THEN r.name ELSE NULL END as responder_name
FROM questions q
LEFT JOIN users u ON q.user_id = u.id
LEFT JOIN users r ON q.responded_by = r.id
WHERE 
    ($1::uuid IS NULL OR q.user_id = $1) AND
    ($2::varchar IS NULL OR q.status = $2) AND
    ($3::timestamptz IS NULL OR q.created_at >= $3) AND
    ($4::timestamptz IS NULL OR q.created_at <= $4)
ORDER BY q.created_at DESC
LIMIT $5 OFFSET $6;

-- name: GetQuestionsByUser :many
SELECT q.* FROM questions q
WHERE q.user_id = $1
ORDER BY q.created_at DESC;

-- name: GetAllQuestions :many
SELECT 
    q.*,
    u.name as user_name
FROM questions q
LEFT JOIN users u ON q.user_id = u.id
ORDER BY q.created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetAllQuestionsCount :one
SELECT COUNT(*) FROM questions;

-- name: GetQuestionsByStatus :many
SELECT 
    q.*,
    u.name as user_name,
    r.name as responder_name
FROM questions q
LEFT JOIN users u ON q.user_id = u.id
LEFT JOIN users r ON q.responded_by = r.id
WHERE q.status = $1
ORDER BY q.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetPendingQuestions :many
SELECT 
    q.*,
    u.name as user_name
FROM questions q
LEFT JOIN users u ON q.user_id = u.id
WHERE q.status = 'pending'
ORDER BY q.created_at ASC;

-- name: UpdateQuestion :one
UPDATE questions 
SET 
    question = COALESCE($2, question),
    status = COALESCE($3, status),
    admin_response = COALESCE($4, admin_response),
    responded_by = COALESCE($5, responded_by),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateQuestionStatus :one
UPDATE questions 
SET 
    status = $2,
    admin_response = $3,
    responded_by = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateQuestionText :one
UPDATE questions 
SET 
    question = $2,
    updated_at = NOW()
WHERE id = $1 AND user_id = $3 AND status = 'pending'
RETURNING *;

-- name: DeleteQuestion :exec
DELETE FROM questions 
WHERE id = $1;

-- name: DeleteQuestionByUser :exec
DELETE FROM questions 
WHERE id = $1 AND user_id = $2 AND status = 'pending';

-- name: GetQuestionStats :one
SELECT 
    COUNT(*) as total,
    COUNT(*) FILTER (WHERE status = 'pending') as pending,
    COUNT(*) FILTER (WHERE status = 'answered') as answered,
    COUNT(*) FILTER (WHERE status = 'closed') as closed,
    COUNT(*) FILTER (WHERE status = 'cancelled') as cancelled
FROM questions;

-- name: GetUserQuestionStats :one
SELECT 
    COUNT(*) as total,
    COUNT(*) FILTER (WHERE status = 'pending') as pending,
    COUNT(*) FILTER (WHERE status = 'answered') as answered,
    COUNT(*) FILTER (WHERE status = 'closed') as closed,
    COUNT(*) FILTER (WHERE status = 'cancelled') as cancelled
FROM questions
WHERE user_id = $1;
