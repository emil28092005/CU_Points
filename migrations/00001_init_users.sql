-- +goose Up
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email       TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    -- student_id is NULL for partner and admin accounts
    student_id  TEXT UNIQUE,
    password_hash TEXT NOT NULL,
    role        TEXT NOT NULL CHECK (role IN ('student', 'partner', 'admin')),
    -- balance is denormalised for read performance; always updated atomically with transactions table
    balance     INTEGER NOT NULL DEFAULT 0 CHECK (balance >= 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ON users(email);
CREATE INDEX ON users(role);

-- +goose Down
DROP TABLE IF EXISTS users;
