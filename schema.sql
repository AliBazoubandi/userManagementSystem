-- Rooms Table
CREATE TABLE rooms (
    id SERIAL PRIMARY KEY,
    name varchar(255) NOT NULL
);

-- Users Table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username varchar(255) NOT NULL,
    email varchar(255) NOT NULL,
    password varchar(255) NOT NULL,
    age int,
    room_id INT REFERENCES rooms(id) ON DELETE SET NULL,
    created_at timestamp DEFAULT NOW()
);
