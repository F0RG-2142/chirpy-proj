-- +goose Up
CREATE TABLE Users (
    id UUID,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    email TEXT
);
CREATE TABLE Chirps (
    id UUID,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    body TEXT,
    user_id UUID
);

-- +goose Down
DROP TABLE users;