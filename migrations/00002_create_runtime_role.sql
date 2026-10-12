-- +goose Up
CREATE ROLE meteor_app
    NOLOGIN
    NOSUPERUSER
    NOCREATEDB
    NOCREATEROLE
    NOINHERIT
    NOBYPASSRLS;

GRANT CONNECT ON DATABASE meteor TO meteor_app;
GRANT USAGE ON SCHEMA public TO meteor_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON tenants, projects TO meteor_app;

-- +goose Down
REVOKE SELECT, INSERT, UPDATE, DELETE ON tenants, projects FROM meteor_app;
REVOKE USAGE ON SCHEMA public FROM meteor_app;
REVOKE CONNECT ON DATABASE meteor FROM meteor_app;
DROP ROLE meteor_app;
