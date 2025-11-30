-- name: CreateReservation :one
INSERT INTO reservations (user_id, room_id, start_date, end_date, total_price, status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, room_id, start_date, end_date, total_price, status, created_at, updated_at;

-- name: GetReservation :one
SELECT id, user_id, room_id, start_date, end_date, total_price, status, created_at, updated_at
FROM reservations
WHERE id = $1 LIMIT 1;

-- name: ListReservationsByUserID :many
SELECT id, user_id, room_id, start_date, end_date, total_price, status, created_at, updated_at
FROM reservations
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: UpdateReservationStatus :one
UPDATE reservations
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, user_id, room_id, start_date, end_date, total_price, status, created_at, updated_at;

