-- +goose Up
-- +goose StatementBegin
INSERT INTO users (full_name) VALUES ('Alice Wonderland');
INSERT INTO users (full_name) VALUES ('Bob Builder');
INSERT INTO users (full_name) VALUES ('Charlie Brown');
INSERT INTO users (full_name) VALUES ('Dora Explorer');
INSERT INTO users (full_name) VALUES ('Eve Online');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM users;
-- +goose StatementEnd
