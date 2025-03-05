-- name: CreateUser :one
INSERT INTO users (username, email, password, age) VALUES ($1, $2, $3, $4) RETURNING username, email, age;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1 LIMIT 1;

-- name: GetUsers :many
SELECT * FROM users ORDER BY username ASC;

-- name: AddUserToRoom :one
UPDATE users SET room_id = $2 WHERE username = $1 RETURNING *;

-- name: RemoveUserFromARoom :one
UPDATE users SET room_id = NULL WHERE id = $1 RETURNING *;

-- name: UpdateUser :one
UPDATE users SET username = $2, email = $3, age = $4 WHERE id = $1 RETURNING *;

-- name: DeleteUser :one
DELETE FROM users WHERE id = $1 RETURNING *;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1 LIMIT 1;

-- name: GetUsersByRoomID :many
SELECT * FROM users WHERE room_id = $1 ORDER BY id ASC;

-- name: UpdateUserPassword :one
UPDATE users SET password = $2 WHERE id = $1 RETURNING *;

-- name: CreateRoom :one
INSERT INTO rooms (name) VALUES ($1) RETURNING *;

-- name: GetRooms :many
SELECT * FROM rooms ORDER BY name ASC;

-- name: GetRoomById :one
SELECT * FROM rooms WHERE id = $1 LIMIT 1;

-- name: DeleteRoom :one
DELETE FROM rooms WHERE id = $1 RETURNING *;

