-- Bật các extension cần thiết cho toàn bộ hệ thống
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- cho gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS postgis;    -- cho kiểu dữ liệu GEOGRAPHY + hàm ST_*
CREATE EXTENSION IF NOT EXISTS timescaledb; -- cho hypertable (time-series)
