-- name: CreateCommunion :one
INSERT INTO communion (
    user_id,
    communion_date,
    status,
    requested_at
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: GetCommunion :one
SELECT * FROM communion 
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetCommunionWithUser :one
SELECT 
    c.id,
    c.user_id,
    c.communion_date,
    c.status,
    c.requested_at,
    c.approved_at,
    c.approved_by_user_id,
    c.created_at,
    c.updated_at,
    c.deleted_at,
    u.name as user_name,
    u.lastname as user_lastname,
    u.phone_number as user_phone,
    au.name as approved_by_name,
    au.lastname as approved_by_lastname
FROM communion c
JOIN users u ON c.user_id = u.id
LEFT JOIN users au ON c.approved_by_user_id = au.id
WHERE c.id = $1 AND c.deleted_at IS NULL;

-- name: GetUserCommunion :one
SELECT * FROM communion 
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: GetUserCommunionWithDetails :one
SELECT 
    c.id,
    c.user_id,
    c.communion_date,
    c.status,
    c.requested_at,
    c.approved_at,
    c.approved_by_user_id,
    c.created_at,
    c.updated_at,
    c.deleted_at,
    au.name as approved_by_name,
    au.lastname as approved_by_lastname
FROM communion c
LEFT JOIN users au ON c.approved_by_user_id = au.id
WHERE c.user_id = $1 AND c.deleted_at IS NULL
ORDER BY c.created_at DESC
LIMIT 1;

-- name: GetAllCommunions :many
SELECT 
    c.id,
    c.user_id,
    c.communion_date,
    c.status,
    c.requested_at,
    c.approved_at,
    c.approved_by_user_id,
    c.created_at,
    c.updated_at,
    c.deleted_at,
    u.name as user_name,
    u.lastname as user_lastname,
    u.phone_number as user_phone,
    au.name as approved_by_name,
    au.lastname as approved_by_lastname
FROM communion c
JOIN users u ON c.user_id = u.id
LEFT JOIN users au ON c.approved_by_user_id = au.id
WHERE c.deleted_at IS NULL
ORDER BY c.created_at DESC
LIMIT $2 OFFSET $1;

-- name: CountAllCommunions :one
SELECT COUNT(*) FROM communion 
WHERE deleted_at IS NULL;

-- name: GetPendingCommunions :many
SELECT 
    c.id,
    c.user_id,
    c.communion_date,
    c.status,
    c.requested_at,
    c.approved_at,
    c.approved_by_user_id,
    c.created_at,
    c.updated_at,
    c.deleted_at,
    u.name as user_name,
    u.lastname as user_lastname,
    u.phone_number as user_phone
FROM communion c
JOIN users u ON c.user_id = u.id
WHERE c.status = 'pending' AND c.deleted_at IS NULL
ORDER BY c.created_at ASC
LIMIT $2 OFFSET $1;

-- name: CountPendingCommunions :one
SELECT COUNT(*) FROM communion 
WHERE status = 'pending' AND deleted_at IS NULL;

-- name: UpdateCommunionStatus :one
UPDATE communion
SET 
    status = $1,
    approved_at = CASE WHEN $1 IN ('approved', 'rejected') THEN now() ELSE approved_at END,
    approved_by_user_id = CASE WHEN $1 IN ('approved', 'rejected') THEN $2 ELSE approved_by_user_id END,
    updated_at = now()
WHERE id = $3 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateCommunion :one
UPDATE communion
SET 
    communion_date = COALESCE($1, communion_date),
    updated_at = now()
WHERE id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteCommunion :one
UPDATE communion 
SET deleted_at = now() 
WHERE id = $1 AND deleted_at IS NULL 
RETURNING *;
