-- name: CreateUser :one
INSERT INTO users (email, hashed_password, name, role)
VALUES ($1, $2, $3, $4)
RETURNING id, email, hashed_password, name, role, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, hashed_password, name, role, created_at, updated_at FROM users
WHERE email = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT id, email, hashed_password, name, role, created_at, updated_at FROM users
WHERE id = $1 LIMIT 1;

