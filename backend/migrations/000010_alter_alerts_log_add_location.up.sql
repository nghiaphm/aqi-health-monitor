ALTER TABLE alerts_log
    ADD COLUMN location_id UUID REFERENCES user_locations(id) ON DELETE SET NULL;

-- Tối ưu truy vấn chống spam theo cặp (user, station) thay vì chỉ theo user
CREATE INDEX idx_alerts_log_user_station_sent_at ON alerts_log (user_id, station_id, sent_at DESC);
