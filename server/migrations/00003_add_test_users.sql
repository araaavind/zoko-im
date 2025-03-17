-- +goose Up
-- +goose StatementBegin
INSERT INTO users (full_name) VALUES ('Alice Wonderland');
INSERT INTO users (full_name) VALUES ('Bob Builder');
INSERT INTO users (full_name) VALUES ('Charlie Brown');
INSERT INTO users (full_name) VALUES ('Dora Explorer');
INSERT INTO users (full_name) VALUES ('Eve Online');
INSERT INTO users (full_name) VALUES ('Frank Castle');
INSERT INTO users (full_name) VALUES ('Gina Green');
INSERT INTO users (full_name) VALUES ('Harry Potter');
INSERT INTO users (full_name) VALUES ('Ivy League');
INSERT INTO users (full_name) VALUES ('Jack Sparrow');
INSERT INTO users (full_name) VALUES ('Luna Lovegood');
INSERT INTO users (full_name) VALUES ('Mickey Mouse');
INSERT INTO users (full_name) VALUES ('Nina Simone');
INSERT INTO users (full_name) VALUES ('Oscar Wilde');
INSERT INTO users (full_name) VALUES ('Peter Parker');
INSERT INTO users (full_name) VALUES ('Quinn Fabray');
INSERT INTO users (full_name) VALUES ('Ron Weasley');
INSERT INTO users (full_name) VALUES ('Samantha Smith');
INSERT INTO users (full_name) VALUES ('Tony Stark');
INSERT INTO users (full_name) VALUES ('Uma Thurman');
INSERT INTO users (full_name) VALUES ('Victor Frankenstein');
INSERT INTO users (full_name) VALUES ('Wanda Maximoff');
INSERT INTO users (full_name) VALUES ('Xena Warrior');
INSERT INTO users (full_name) VALUES ('Yoda Master');
INSERT INTO users (full_name) VALUES ('Zorro Mask');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM users;
-- +goose StatementEnd
