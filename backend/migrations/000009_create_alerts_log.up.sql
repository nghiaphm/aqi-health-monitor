CREATE TYPE alert_channel_enum AS ENUM ('web_push', 'email');
CREATE TYPE alert_status_enum AS ENUM ('sent', 'failed');

CREATE TABLE alerts_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    station_id  UUID NOT NULL REFERENCES stations(id),
    aqi_value   INT NOT NULL,
    channel     alert_channel_enum NOT NULL,
    sent_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    status      alert_status_enum NOT NULL DEFAULT 'sent'
);

-- Phục vụ query "đã gửi cảnh báo trong X giờ gần đây chưa" (chống spam)
CREATE INDEX idx_alerts_log_user_sent_at ON alerts_log (user_id, sent_at DESC);
