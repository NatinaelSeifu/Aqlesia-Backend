-- name: CreateAppointment :one
INSERT INTO appointments (
    user_id,
    appointment_date,
    status,
    notes
) VALUES (
    @user_id, @appointment_date, @status, @notes
)
RETURNING *;

-- name: GetAppointment :one
SELECT * FROM appointments 
WHERE id = @id AND deleted_at IS NULL;

-- name: GetAppointmentWithUser :one
SELECT 
    a.id,
    a.user_id,
    a.appointment_date,
    a.status,
    a.notes,
    a.created_at,
    a.updated_at,
    a.deleted_at,
    u.name as user_name,
    u.lastname as user_lastname,
    u.phone_number as user_phone
FROM appointments a
JOIN users u ON a.user_id = u.id
WHERE a.id = @id AND a.deleted_at IS NULL;

-- name: GetUserAppointments :many
SELECT * FROM appointments 
WHERE user_id = @user_id AND deleted_at IS NULL
ORDER BY appointment_date DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: GetAllAppointments :many
SELECT 
    a.id,
    a.user_id,
    a.appointment_date,
    a.status,
    a.notes,
    a.created_at,
    a.updated_at,
    a.deleted_at,
    u.name as user_name,
    u.lastname as user_lastname,
    u.phone_number as user_phone
FROM appointments a
JOIN users u ON a.user_id = u.id
WHERE a.deleted_at IS NULL
ORDER BY a.appointment_date DESC, a.created_at DESC
LIMIT @limit_count OFFSET @offset_count;

-- name: UpdateAppointment :one
UPDATE appointments
SET 
    appointment_date = COALESCE(@appointment_date, appointment_date),
    notes = COALESCE(@notes, notes),
    updated_at = now()
WHERE id = @id AND deleted_at IS NULL
RETURNING *;

-- name: UpdateAppointmentStatus :one
UPDATE appointments
SET 
    status = @status,
    notes = COALESCE(@notes, notes),
    updated_at = now()
WHERE id = @id AND deleted_at IS NULL
RETURNING *;

-- name: DeleteAppointment :one
UPDATE appointments 
SET deleted_at = now() 
WHERE id = @id AND deleted_at IS NULL 
RETURNING *;

-- name: GetUserActiveAppointment :one
SELECT * FROM appointments 
WHERE user_id = @user_id 
  AND status IN ('pending') 
  AND appointment_date >= CURRENT_DATE
  AND deleted_at IS NULL
ORDER BY appointment_date ASC
LIMIT 1;

-- name: CountAppointmentsByDate :one
SELECT COUNT(*) FROM appointments 
WHERE appointment_date = @appointment_date 
  AND status IN ('pending', 'completed') 
  AND deleted_at IS NULL;

-- name: CountUserAppointments :one
SELECT COUNT(*) FROM appointments 
WHERE user_id = @user_id AND deleted_at IS NULL;

-- name: CountAllAppointments :one
SELECT COUNT(*) FROM appointments 
WHERE deleted_at IS NULL;

-- name: GetAppointmentStats :one
SELECT 
    COUNT(*) as total_appointments,
    COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_appointments,
    COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_appointments,
    COUNT(CASE WHEN status = 'cancelled' THEN 1 END) as cancelled_appointments
FROM appointments 
WHERE deleted_at IS NULL;

-- name: GetAppointmentsByDateRange :many
SELECT 
    a.id,
    a.user_id,
    a.appointment_date,
    a.status,
    a.notes,
    a.created_at,
    a.updated_at,
    a.deleted_at,
    u.name as user_name,
    u.lastname as user_lastname,
    u.phone_number as user_phone
FROM appointments a
JOIN users u ON a.user_id = u.id
WHERE a.appointment_date >= @start_date 
  AND a.appointment_date <= @end_date
  AND a.deleted_at IS NULL
ORDER BY a.appointment_date ASC, a.created_at ASC;

-- name: GetUpcomingAppointments :many
SELECT 
    a.id,
    a.user_id,
    a.appointment_date,
    a.status,
    a.notes,
    a.created_at,
    a.updated_at,
    a.deleted_at,
    u.name as user_name,
    u.lastname as user_lastname,
    u.phone_number as user_phone
FROM appointments a
JOIN users u ON a.user_id = u.id
WHERE a.appointment_date >= CURRENT_DATE
  AND a.status = 'pending'
  AND a.deleted_at IS NULL
ORDER BY a.appointment_date ASC, a.created_at ASC
LIMIT @limit_count;
