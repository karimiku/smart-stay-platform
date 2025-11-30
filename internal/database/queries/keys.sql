-- name: CreateKey :one
INSERT INTO keys (reservation_id, user_id, key_code, device_id, valid_from, valid_until)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, reservation_id, user_id, key_code, device_id, valid_from, valid_until, created_at, updated_at;

-- name: GetKeyByReservationID :one
SELECT id, reservation_id, user_id, key_code, device_id, valid_from, valid_until, created_at, updated_at
FROM keys
WHERE reservation_id = $1 LIMIT 1;

-- name: ListKeysByUserID :many
SELECT id, reservation_id, user_id, key_code, device_id, valid_from, valid_until, created_at, updated_at
FROM keys
WHERE user_id = $1
ORDER BY valid_from DESC;

-- name: ListActiveKeysByUserID :many
SELECT id, reservation_id, user_id, key_code, device_id, valid_from, valid_until, created_at, updated_at
FROM keys
WHERE user_id = $1
  AND DATE(valid_from) <= CURRENT_DATE
  AND DATE(valid_until) >= CURRENT_DATE
ORDER BY valid_from DESC;

