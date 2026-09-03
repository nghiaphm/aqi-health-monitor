CREATE TABLE stations (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    waqi_station_id  VARCHAR(100) NOT NULL UNIQUE,
    name             VARCHAR(255) NOT NULL,
    geom             GEOGRAPHY(Point, 4326) NOT NULL,
    city             VARCHAR(100),
    is_active        BOOLEAN NOT NULL DEFAULT true,
    last_synced_at   TIMESTAMPTZ
);

CREATE INDEX idx_stations_geom ON stations USING GIST (geom);
CREATE INDEX idx_stations_is_active ON stations (is_active);
