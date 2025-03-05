-- +goose Up
-- +goose StatementBegin
INSERT INTO users (username, email, password, age) VALUES ('jesse_james', 'jesse_james@gmail.com', '987654321', 25);
INSERT INTO users (username, email, password, age) VALUES ('will_tennyson', 'will_tennyson@gmail.com', 'qwerty123', 27);
INSERT INTO users (username, email, password, age) VALUES ('mike_isretel', 'mike_isretel@gmail.com', 'password123', 46);
INSERT INTO users (username, email, password, age) VALUES ('jeff_nipperd', 'jeff_nipperd@gmail.com', 'password321', 28);
INSERT INTO users (username, email, password, age) VALUES ('tren_twins', 'tren_twins@gmail.com', 'password123', 26);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM users WHERE username IN ('jesse_james', 'will_tennyson', 'mike_isretel', 'jeff_nipperd', 'tren_twins');
-- +goose StatementEnd
