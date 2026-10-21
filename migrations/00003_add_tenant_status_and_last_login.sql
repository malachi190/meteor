-- +goose Up
ALTER TABLE tenants
    ADD COLUMN status text NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'inactive')),
    ADD COLUMN last_login_at timestamptz;

-- +goose Down
ALTER TABLE tenants
    DROP COLUMN last_login_at,
    DROP COLUMN status;
