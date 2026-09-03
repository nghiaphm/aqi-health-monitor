CREATE TABLE alert_thresholds (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    threshold_aqi INT NOT NULL CHECK (threshold_aqi >= 0 AND threshold_aqi <= 500),
    is_custom     BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_alert_thresholds_user_id ON alert_thresholds (user_id);
