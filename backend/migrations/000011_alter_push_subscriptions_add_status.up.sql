ALTER TABLE push_subscriptions
    ADD COLUMN is_active     BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN last_failed_at TIMESTAMPTZ;

CREATE INDEX idx_push_subscriptions_is_active ON push_subscriptions (is_active);
