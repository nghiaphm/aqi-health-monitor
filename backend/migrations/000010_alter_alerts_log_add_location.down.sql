DROP INDEX IF EXISTS idx_alerts_log_user_station_sent_at;
ALTER TABLE alerts_log DROP COLUMN IF EXISTS location_id;
