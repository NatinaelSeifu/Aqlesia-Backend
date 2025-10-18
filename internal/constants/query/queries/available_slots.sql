-- name: CreateAvailableDate :one
INSERT INTO available_dates (
    slot_date,
    max_capacity,
    current_bookings,
    is_active
) VALUES (
    @slot_date, @max_capacity, @current_bookings, @is_active
)
RETURNING *;

-- name: GetAvailableDate :one
SELECT * FROM available_dates 
WHERE slot_date = @slot_date;

-- name: GetAvailableDates :many
SELECT * FROM available_dates 
WHERE slot_date >= @start_date 
  AND slot_date <= @end_date
  AND is_active = true
ORDER BY slot_date ASC;

-- name: GetAvailableDatesWithCapacity :many
SELECT * FROM available_dates 
WHERE slot_date >= @start_date 
  AND slot_date <= @end_date
  AND is_active = true
  AND current_bookings < max_capacity
ORDER BY slot_date ASC;

-- name: UpdateDateBookingCount :one
UPDATE available_dates
SET current_bookings = @current_bookings,
    updated_at = now()
WHERE slot_date = @slot_date
RETURNING *;

-- name: DeactivateDate :one
UPDATE available_dates
SET is_active = false,
    updated_at = now()
WHERE slot_date = @slot_date
RETURNING *;

-- name: ActivateDate :one
UPDATE available_dates
SET is_active = true,
    updated_at = now()
WHERE slot_date = @slot_date
RETURNING *;

-- name: DeleteOldDates :exec
DELETE FROM available_dates
WHERE slot_date < @cutoff_date;

-- name: UpsertAvailableDate :one
INSERT INTO available_dates (
    slot_date,
    max_capacity,
    current_bookings,
    is_active
) VALUES (
    @slot_date, @max_capacity, @current_bookings, @is_active
)
ON CONFLICT (slot_date) 
DO UPDATE SET 
    max_capacity = EXCLUDED.max_capacity,
    is_active = EXCLUDED.is_active,
    updated_at = now()
RETURNING *;

-- name: GetDateAvailability :one
SELECT 
    slot_date,
    max_capacity,
    current_bookings,
    (max_capacity - current_bookings) as available_spots,
    is_active
FROM available_dates 
WHERE slot_date = @slot_date;

-- name: CountAvailableDates :one
SELECT COUNT(*) FROM available_dates 
WHERE slot_date >= @start_date 
  AND slot_date <= @end_date
  AND is_active = true
  AND current_bookings < max_capacity;

-- Admin/Manager queries for managing availability dates
-- name: GetAllAvailableDates :many
SELECT * FROM available_dates 
WHERE slot_date >= @start_date 
  AND slot_date <= @end_date
ORDER BY slot_date ASC;

-- name: UpdateAvailableDate :one
UPDATE available_dates
SET max_capacity = @max_capacity,
    is_active = @is_active,
    updated_at = now()
WHERE slot_date = @slot_date
RETURNING *;

-- name: DeleteAvailableDate :exec
DELETE FROM available_dates
WHERE slot_date = @slot_date;
