// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: query.sql

package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const addUserToRoom = `-- name: AddUserToRoom :one
UPDATE users SET room_id = $2 WHERE username = $1 RETURNING id, username, email, password, age, room_id, created_at
`

type AddUserToRoomParams struct {
	Username string      `json:"username"`
	RoomID   pgtype.Int4 `json:"room_id"`
}

func (q *Queries) AddUserToRoom(ctx context.Context, arg AddUserToRoomParams) (User, error) {
	row := q.db.QueryRow(ctx, addUserToRoom, arg.Username, arg.RoomID)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Email,
		&i.Password,
		&i.Age,
		&i.RoomID,
		&i.CreatedAt,
	)
	return i, err
}

const createRoom = `-- name: CreateRoom :one
INSERT INTO rooms (name) VALUES ($1) RETURNING id, name
`

func (q *Queries) CreateRoom(ctx context.Context, name string) (Room, error) {
	row := q.db.QueryRow(ctx, createRoom, name)
	var i Room
	err := row.Scan(&i.ID, &i.Name)
	return i, err
}

const createUser = `-- name: CreateUser :one
INSERT INTO users (username, email, password, age) VALUES ($1, $2, $3, $4) RETURNING username, email, age
`

type CreateUserParams struct {
	Username string      `json:"username"`
	Email    string      `json:"email"`
	Password string      `json:"password"`
	Age      pgtype.Int4 `json:"age"`
}

type CreateUserRow struct {
	Username string      `json:"username"`
	Email    string      `json:"email"`
	Age      pgtype.Int4 `json:"age"`
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (CreateUserRow, error) {
	row := q.db.QueryRow(ctx, createUser,
		arg.Username,
		arg.Email,
		arg.Password,
		arg.Age,
	)
	var i CreateUserRow
	err := row.Scan(&i.Username, &i.Email, &i.Age)
	return i, err
}

const deleteRoom = `-- name: DeleteRoom :one
DELETE FROM rooms WHERE id = $1 RETURNING id, name
`

func (q *Queries) DeleteRoom(ctx context.Context, id int32) (Room, error) {
	row := q.db.QueryRow(ctx, deleteRoom, id)
	var i Room
	err := row.Scan(&i.ID, &i.Name)
	return i, err
}

const deleteUser = `-- name: DeleteUser :one
DELETE FROM users WHERE id = $1 RETURNING id, username, email, password, age, room_id, created_at
`

func (q *Queries) DeleteUser(ctx context.Context, id int32) (User, error) {
	row := q.db.QueryRow(ctx, deleteUser, id)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Email,
		&i.Password,
		&i.Age,
		&i.RoomID,
		&i.CreatedAt,
	)
	return i, err
}

const getRoomById = `-- name: GetRoomById :one
SELECT id, name FROM rooms WHERE id = $1 LIMIT 1
`

func (q *Queries) GetRoomById(ctx context.Context, id int32) (Room, error) {
	row := q.db.QueryRow(ctx, getRoomById, id)
	var i Room
	err := row.Scan(&i.ID, &i.Name)
	return i, err
}

const getRooms = `-- name: GetRooms :many
SELECT id, name FROM rooms ORDER BY name ASC
`

func (q *Queries) GetRooms(ctx context.Context) ([]Room, error) {
	rows, err := q.db.Query(ctx, getRooms)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Room
	for rows.Next() {
		var i Room
		if err := rows.Scan(&i.ID, &i.Name); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getUser = `-- name: GetUser :one
SELECT id, username, email, password, age, room_id, created_at FROM users WHERE id = $1 LIMIT 1
`

func (q *Queries) GetUser(ctx context.Context, id int32) (User, error) {
	row := q.db.QueryRow(ctx, getUser, id)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Email,
		&i.Password,
		&i.Age,
		&i.RoomID,
		&i.CreatedAt,
	)
	return i, err
}

const getUserByUsername = `-- name: GetUserByUsername :one
SELECT id, username, email, password, age, room_id, created_at FROM users WHERE username = $1 LIMIT 1
`

func (q *Queries) GetUserByUsername(ctx context.Context, username string) (User, error) {
	row := q.db.QueryRow(ctx, getUserByUsername, username)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Email,
		&i.Password,
		&i.Age,
		&i.RoomID,
		&i.CreatedAt,
	)
	return i, err
}

const getUsers = `-- name: GetUsers :many
SELECT id, username, email, password, age, room_id, created_at FROM users ORDER BY username ASC
`

func (q *Queries) GetUsers(ctx context.Context) ([]User, error) {
	rows, err := q.db.Query(ctx, getUsers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []User
	for rows.Next() {
		var i User
		if err := rows.Scan(
			&i.ID,
			&i.Username,
			&i.Email,
			&i.Password,
			&i.Age,
			&i.RoomID,
			&i.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getUsersByRoomID = `-- name: GetUsersByRoomID :many
SELECT id, username, email, password, age, room_id, created_at FROM users WHERE room_id = $1 ORDER BY id ASC
`

func (q *Queries) GetUsersByRoomID(ctx context.Context, roomID pgtype.Int4) ([]User, error) {
	rows, err := q.db.Query(ctx, getUsersByRoomID, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []User
	for rows.Next() {
		var i User
		if err := rows.Scan(
			&i.ID,
			&i.Username,
			&i.Email,
			&i.Password,
			&i.Age,
			&i.RoomID,
			&i.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const removeUserFromARoom = `-- name: RemoveUserFromARoom :one
UPDATE users SET room_id = NULL WHERE id = $1 RETURNING id, username, email, password, age, room_id, created_at
`

func (q *Queries) RemoveUserFromARoom(ctx context.Context, id int32) (User, error) {
	row := q.db.QueryRow(ctx, removeUserFromARoom, id)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Email,
		&i.Password,
		&i.Age,
		&i.RoomID,
		&i.CreatedAt,
	)
	return i, err
}

const updateUser = `-- name: UpdateUser :one
UPDATE users SET username = $2, email = $3, age = $4 WHERE id = $1 RETURNING id, username, email, password, age, room_id, created_at
`

type UpdateUserParams struct {
	ID       int32       `json:"id"`
	Username string      `json:"username"`
	Email    string      `json:"email"`
	Age      pgtype.Int4 `json:"age"`
}

func (q *Queries) UpdateUser(ctx context.Context, arg UpdateUserParams) (User, error) {
	row := q.db.QueryRow(ctx, updateUser,
		arg.ID,
		arg.Username,
		arg.Email,
		arg.Age,
	)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Email,
		&i.Password,
		&i.Age,
		&i.RoomID,
		&i.CreatedAt,
	)
	return i, err
}

const updateUserPassword = `-- name: UpdateUserPassword :one
UPDATE users SET password = $2 WHERE id = $1 RETURNING id, username, email, password, age, room_id, created_at
`

type UpdateUserPasswordParams struct {
	ID       int32  `json:"id"`
	Password string `json:"password"`
}

func (q *Queries) UpdateUserPassword(ctx context.Context, arg UpdateUserPasswordParams) (User, error) {
	row := q.db.QueryRow(ctx, updateUserPassword, arg.ID, arg.Password)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Email,
		&i.Password,
		&i.Age,
		&i.RoomID,
		&i.CreatedAt,
	)
	return i, err
}
