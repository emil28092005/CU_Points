-- +goose Up
-- Configurable rules that determine how many points a student earns per trigger event.
-- Managed by administrators through the admin dashboard.
CREATE TABLE earning_rules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    points_amount   INTEGER NOT NULL CHECK (points_amount > 0),
    -- trigger_type matches the source system or event that initiates earning
    trigger_type    TEXT NOT NULL CHECK (trigger_type IN ('attendance', 'assignment', 'referral', 'admin')),
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ON earning_rules(trigger_type, is_active);

-- Seed default admin-grant rule so administrators can grant points immediately after deploy
INSERT INTO earning_rules (name, points_amount, trigger_type, is_active)
VALUES ('Manual admin grant', 1, 'admin', TRUE);

-- +goose Down
DROP TABLE IF EXISTS earning_rules;
