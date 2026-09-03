CREATE TYPE condition_type_enum AS ENUM ('none', 'asthma', 'allergic_rhinitis');
CREATE TYPE age_group_enum AS ENUM ('child', 'adult', 'elderly');
CREATE TYPE sensitivity_level_enum AS ENUM ('normal', 'high');

CREATE TABLE health_profiles (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    condition_type     condition_type_enum NOT NULL DEFAULT 'none',
    age_group          age_group_enum NOT NULL DEFAULT 'adult',
    sensitivity_level  sensitivity_level_enum NOT NULL DEFAULT 'normal',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
