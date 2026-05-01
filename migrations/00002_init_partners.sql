-- +goose Up
CREATE TABLE partners (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- user_id references the cashier/partner account in users table
    user_id         UUID REFERENCES users(id) ON DELETE SET NULL,
    name            TEXT NOT NULL,
    address         TEXT NOT NULL,
    -- max_spend_pct: maximum percentage of a purchase total that can be paid with points
    max_spend_pct   INTEGER NOT NULL DEFAULT 50 CHECK (max_spend_pct BETWEEN 1 AND 100),
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ON partners(is_active);

-- +goose Down
DROP TABLE IF EXISTS partners;
