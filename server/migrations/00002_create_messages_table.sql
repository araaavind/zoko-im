-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS messages (
    id bigserial PRIMARY KEY,  
    timestamp timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    content text NOT NULL,
    sender_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    receiver_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    read_status boolean NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_messages_sender_receiver ON messages (sender_id, receiver_id);
CREATE INDEX idx_messages_timestamp ON messages (timestamp);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_messages_sender_receiver;
DROP INDEX IF EXISTS idx_messages_timestamp;
DROP TABLE IF EXISTS messages;
-- +goose StatementEnd
