CREATE TABLE user_locations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label      VARCHAR(50) NOT NULL DEFAULT 'current', -- 'current' | 'home'
    geom       GEOGRAPHY(Point, 4326) NOT NULL,
    city       VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_locations_geom ON user_locations USING GIST (geom);
CREATE INDEX idx_user_locations_user_id ON user_locations (user_id);
