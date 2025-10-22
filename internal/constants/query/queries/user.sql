-- name: CreateUser :one
INSERT INTO users (
    name,
    lastname,
    phone_number,
    password,
    role,
    telegram_id,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: UserByPhoneExists :one
SELECT count(*) FROM users where phone_number = $1 AND deleted_at IS NULL;

-- name: DeleteUser :one
UPDATE users set deleted_at =now() where id=$1 AND deleted_at IS NULL RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET 
  name = COALESCE(NULLIF(@name, ''), name),
  lastname = COALESCE(NULLIF(@lastname, ''), lastname),
  phone_number = COALESCE(NULLIF(@phone_number, ''), phone_number),
  job_title = @job_title,
  education = @education,
  marriage_status = @marriage_status,
  partner_name = @partner_name,
  childrens_name = @childrens_name,
  telegram_id = @telegram_id,
  updated_at = now()
WHERE id = @id AND deleted_at IS NULL
RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUsers :many
SELECT * FROM users WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: CountUsers :one
SELECT COUNT(*) FROM users WHERE deleted_at IS NULL;

-- name: GetUserByPhone :one
SELECT * FROM users WHERE phone_number = $1 AND deleted_at IS NULL;

-- name: ChangePassword :one
UPDATE users
SET 
  password = $1,
  updated_at = now()
WHERE id = $2 AND deleted_at IS NULL
RETURNING id, created_at, updated_at, deleted_at, name, lastname, phone_number, password, telegram_id, role, job_title, education, marriage_status, partner_name, childrens_name, status;

-- name: GetUsersByStatus :many
SELECT * FROM users 
WHERE status = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: CountUsersByStatus :one
SELECT COUNT(*) FROM users WHERE status = $1 AND deleted_at IS NULL;

-- name: UpdateUserStatus :one
UPDATE users
SET 
  status = $1,
  updated_at = now()
WHERE id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: GetUserByPhoneWithPassword :one
SELECT * FROM users 
WHERE phone_number = $1 AND deleted_at IS NULL;

-- name: UpdateTelegramInfo :one
UPDATE users
SET 
  telegram_id = $1,
  telegram_verified = $2,
  updated_at = now()
WHERE phone_number = $3 AND deleted_at IS NULL
RETURNING *;

-- name: GetUserByTelegramID :one
SELECT * FROM users 
WHERE telegram_id = $1 AND telegram_verified = TRUE AND deleted_at IS NULL;

-- name: CreatePasswordResetToken :one
INSERT INTO password_reset_tokens (
    user_id,
    token_hash,
    expires_at
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: GetValidPasswordResetToken :one
SELECT * FROM password_reset_tokens 
WHERE token_hash = $1 AND used = FALSE AND expires_at > NOW();

-- name: MarkPasswordResetTokenUsed :exec
UPDATE password_reset_tokens 
SET used = TRUE, updated_at = NOW()
WHERE id = $1;

-- name: CleanupExpiredResetTokens :exec
DELETE FROM password_reset_tokens 
WHERE expires_at < NOW() OR used = TRUE;

-- name: ResetUserPassword :one
UPDATE users
SET 
  password = $1,
  updated_at = now()
WHERE id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: CreatePasswordResetOTP :one
INSERT INTO password_reset_otps (
    user_id,
    otp_hash,
    expires_at
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: GetValidPasswordResetOTP :one
SELECT * FROM password_reset_otps 
WHERE user_id = $1 AND otp_hash = $2 AND used = FALSE AND expires_at > NOW();

-- name: MarkPasswordResetOTPUsed :exec
UPDATE password_reset_otps 
SET used = TRUE, updated_at = NOW()
WHERE id = $1;

-- name: CleanupExpiredResetOTPs :exec
DELETE FROM password_reset_otps 
WHERE expires_at < NOW() OR used = TRUE;
