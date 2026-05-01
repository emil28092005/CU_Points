-- +goose Up
-- Append-only ledger of all point movements. Never delete rows from this table.
CREATE TABLE transactions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    -- partner_id is NULL for earn and admin_grant transactions
    partner_id  UUID REFERENCES partners(id),
    -- amount > 0 means earn/grant; amount < 0 means spend/expire
    amount      INTEGER NOT NULL,
    type        TEXT NOT NULL CHECK (type IN ('earn', 'spend', 'admin_grant', 'expire')),
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Primary query pattern: a student's history ordered by time
CREATE INDEX ON transactions(user_id, created_at DESC);
-- Secondary pattern: a partner's redemption history
CREATE INDEX ON transactions(partner_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS transactions;
