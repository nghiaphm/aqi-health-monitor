CREATE TABLE users (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    keycloak_id  VARCHAR(255) NOT NULL UNIQUE,   -- claim "sub" từ Keycloak JWT
    email        VARCHAR(255) NOT NULL,
    full_name    VARCHAR(255),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_keycloak_id ON users (keycloak_id);
