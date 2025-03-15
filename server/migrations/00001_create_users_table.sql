-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
	id bigserial PRIMARY KEY,
    full_name text NOT NULL
);
CREATE INDEX IF NOT EXISTS users_id_idx ON users(id);
CREATE INDEX IF NOT EXISTS users_full_name_idx ON users(full_name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS users_full_name_idx;
DROP INDEX IF EXISTS users_id_idx;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
