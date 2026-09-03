CREATE TABLE aqi_readings (
    time                TIMESTAMPTZ NOT NULL,
    station_id          UUID NOT NULL REFERENCES stations(id),
    aqi                 INT NOT NULL,
    pm25                NUMERIC(6,2),
    pm10                NUMERIC(6,2),
    o3                  NUMERIC(6,2),
    no2                 NUMERIC(6,2),
    so2                 NUMERIC(6,2),
    co                  NUMERIC(6,2),
    dominant_pollutant  VARCHAR(20),
    PRIMARY KEY (station_id, time)
);

-- Chuyển thành hypertable — bắt buộc cột partition (time) phải nằm trong PK
SELECT create_hypertable('aqi_readings', 'time');

CREATE INDEX idx_aqi_readings_station_time ON aqi_readings (station_id, time DESC);

-- Nén dữ liệu cũ để tiết kiệm dung lượng (áp dụng cho dữ liệu > 30 ngày)
ALTER TABLE aqi_readings SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'station_id'
);

SELECT add_compression_policy('aqi_readings', INTERVAL '30 days');
