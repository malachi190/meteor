# Source Logic Log

This file records each implementation increment, the files changed, and what the logic does. It is a learning log, not a replacement for code comments or design documentation.

## 2026-10-12 — Initial HTTP API scaffold

- `go.mod`: declares the local Go module and minimum Go version. No external dependencies are used yet.
- `cmd/api/main.go`: reads the listening port from `PORT` (default `8080`), registers a standard-library `GET /healthz` route, and starts an HTTP server with a 5-second header-read timeout.
- `healthResponse` and `healthHandler`: return a small JSON response (`{"status":"ok"}`) with HTTP 200. This is a basic process health endpoint only; it does not yet check PostgreSQL, Redis, or LiveKit.
- Design choice: Go's standard `net/http` router keeps routing and handler behavior visible while we learn HTTP fundamentals. A framework could provide more convenience, but would hide some of this wiring at the start.
- Limitation for this increment: the server does not yet perform graceful shutdown; we can add that after the request/handler flow is familiar.

## 2026-10-12 — PostgreSQL tenant foundation

- `.gitignore` and `.env.example`: keep the local database password out of tracked files while showing which variable Compose requires.
- `compose.yaml`: starts PostgreSQL 18 for local development, binds it only to the machine's loopback interface, persists its data in a named Docker volume, and exposes a readiness check.
- `migrations/00001_create_tenant_core.sql`: creates tenant and project tables, a per-tenant project-name uniqueness rule, and RLS policies that compare each row's tenant ID with the transaction's `app.tenant_id` setting. The tenant ID has no database default because signup must know the ID before setting the RLS context for the insert.
- Design choice: Goose-style SQL migrations keep the actual schema visible and versioned. An ORM could generate schema changes, but that would hide SQL while we are learning it.
- Security note: this Compose database user is an administrative bootstrap user, not the API's future runtime role. The API must use a separate role without table ownership or `BYPASSRLS`; otherwise RLS can be bypassed. PostgreSQL owners normally bypass RLS, while `FORCE ROW LEVEL SECURITY` makes table owners subject to policies; superusers and `BYPASSRLS` roles still bypass it.
- Next connection step: each tenant-scoped database transaction must set `app.tenant_id` locally before its queries, then also include explicit tenant filters in application SQL.

## 2026-10-12 — Meteor naming and connection URL

- `go.mod`, `compose.yaml`, `.env.example`, `SPEC.md`, and `cmd/api/main.go`: rename the app and local database to Meteor.
- `.env.example`: documents a full `DATABASE_URL` for the API running on the host, plus the separate variables the PostgreSQL container needs to initialize itself.
- `.env`: updated the existing local database settings and constructed `DATABASE_URL` from the existing password without printing it. The URL-encoded password must match `POSTGRES_PASSWORD`.
- `compose.yaml` and `.env`: publish the container's PostgreSQL port 5432 on host port 5433 so it does not conflict with a local PostgreSQL server.
- `compose.yaml`: mount the named volume at `/var/lib/postgresql`, which is the data mount layout expected by the PostgreSQL 18 image.
- Applied `migrations/00001_create_tenant_core.sql` with Goose; the local Meteor database is at migration version 1.

## 2026-10-12 — Runtime database pool and restricted role

- `migrations/00002_create_runtime_role.sql`: creates `meteor_app` without login or elevated privileges, then grants only database connection, schema usage, and CRUD access to the tenant/project tables. RLS still filters which rows this role can access. The role password is provisioned separately from migration SQL so secrets do not enter the migration file.
- `.env.example`: separates `DATABASE_URL` for the API from `MIGRATOR_DATABASE_URL` for privileged schema changes. The API URL uses the restricted `meteor_app` role.
- `internal/database/pool.go`: creates a concurrency-safe `pgxpool.Pool`, pings PostgreSQL during startup, and closes the pool if that initial check fails.
- `cmd/api/main.go`: requires `DATABASE_URL`, opens and verifies the pool before starting the HTTP server, and closes the pool when the server exits.
- `go.mod` and `go.sum`: add pgx v5.7.2 and its dependencies while keeping the project on Go 1.22.
- Design choice: `pgxpool` is used instead of opening a fresh connection per request because it reuses a bounded set of database connections. The API role is separate from the migration/admin role so its SQL remains subject to RLS.
- Limitation: tenant context is not set yet because the API has no tenant-scoped database queries. When those are added, each operation will set the tenant ID transaction-locally and also include it in the query filter.

## 2026-10-12 — Tenant context and transaction-local RLS setting

- `internal/tenant/context.go`: stores an authenticated tenant ID in Go's request context using a private key, so unrelated packages cannot collide with the context value. The helper does not read tenant identity from HTTP headers or other client input.
- `internal/database/tenant.go`: requires a tenant ID in context, begins a PostgreSQL transaction, calls parameterized `set_config('app.tenant_id', ..., true)`, and runs the caller's database work in that transaction. The `true` argument makes the setting transaction-local, so it is cleared when the transaction commits or rolls back and cannot leak through the connection pool.
- Design choice: tenant context is applied to each tenant-scoped database transaction instead of setting a pooled connection's session state. Authentication is not implemented yet, so trusted auth code still needs to attach the tenant ID before this helper is called.
- Limitation: there are not yet tenant-scoped handlers or repository queries using the helper; those and cross-tenant verification remain Phase 1 work in `WORKFLOW.md`.

## 2026-10-12 — Runtime configuration, logging, and service layers

- `go.mod` and `go.sum`: add Koanf, godotenv, and Zerolog versions that retain the Go 1.22 module target.
- `internal/config/config.go`: optionally loads `.env` without overwriting variables already present in the process, then maps only `PORT`, `DATABASE_URL`, and `ERROR_LOG_PATH` into a typed config struct. Packages receive configuration from startup wiring instead of reading environment values independently.
- `.env.example`: documents the API port and configurable error-log path alongside database settings.
- `.gitignore`: keeps generated application log files out of source control.
- `internal/logging/logging.go`: creates a human-readable Zerolog logger for CLI output and a separate error-level logger that appends to a private-permission log file. The log directory is ignored by Git.
- `internal/httpapi/server.go`: owns HTTP route and response behavior, logs each request to the CLI without logging headers or query parameters, and records server errors in the error file.
- `internal/health/service.go`: holds the health-check use case and calls a repository interface.
- `internal/database/health_repository.go`: adapts the PostgreSQL pool to the health service's database-check interface.
- `cmd/api/main.go`: acts as the composition root: loads config, initializes logging and the pool, wires repository → service → HTTP handler, then starts the server. Startup/runtime errors are sent to the error logger; if the error log itself cannot be opened, startup reports that failure to stderr.
- The health endpoint returns `200` with `{"status":"ok"}` when PostgreSQL responds and `503` with `{"status":"unavailable"}` when the database check fails.
- OpenTelemetry remains deferred until the baseline logging behavior is established.

## 2026-10-12 — Date-based error log rotation and retention

- `internal/logging/logging.go`: writes errors to files named from the configured base path with the local date inserted before the extension, such as `meteor-errors-2026-10-12.log`.
- `internal/logging/logging.go`: switches to a new file at local midnight and also checks the date when an error is written, so a missed timer does not keep writing to yesterday's file. It removes only matching dated log files older than today and the previous two calendar days.
- `internal/logging/logging.go`: migrates the former undated log to the date from its modification time, appending its contents when that dated file already exists.
- Design choice: calendar dates in the filenames make retention predictable and easy to inspect; file timestamps alone can change when files are copied or restored. The retention policy keeps three dates inclusive.
- Existing `ERROR_LOG_PATH=logs/meteor-errors.log` remains the base path; the running logger derives dated filenames from it.

## 2026-10-16 — sqlc query generation foundation

- `sqlc.yaml`: configures sqlc to read the existing Goose migrations and hand-written queries, then generate a pgx/v5 package under `internal/database/queries`.
- `Makefile`: adds a generation command that runs pinned sqlc version v1.31.1 without adding the generator to Meteor's runtime dependencies.
- `db/queries/projects.sql`: defines project create, tenant-scoped list, and tenant-plus-project lookup queries. Each read query includes an explicit tenant filter as an application-level scope alongside PostgreSQL RLS.
- `internal/database/queries/`: contains generated pgx/v5 models, query methods, and a Querier interface. Regenerate these files from the SQL and config; do not edit them by hand.
- `internal/database/project_repository.go`: converts the trusted context tenant ID to pgx's UUID type, runs create/list/get project operations through `WithTenant`, and passes the tenant ID explicitly to the generated queries.
- Design choice: sqlc generates typed Go wrappers while keeping SQL visible for review; an ORM would instead add a separate mapping and query abstraction that we do not need for this SQL-first project.
- Limitation: tenant creation, dashboard user tables, signup transactions, and the service layer are still pending. Authentication will supply the trusted tenant context in Phase 3.

## 2026-10-16 — Repository package and tenant activity fields

- `internal/repository/health_repository.go` and `internal/repository/project_repository.go`: move repository implementations out of the database infrastructure package into a dedicated repository package. The project repository still uses database transaction helpers and generated sqlc queries.
- `cmd/api/main.go`: update composition-root wiring to construct the health repository from its new package.
- `migrations/00003_add_tenant_status_and_last_login.sql`: add a constrained `active`/`inactive` status with an `active` default and a nullable `last_login_at` timestamp; the down migration removes both fields.
- `internal/database/queries/models.go`: regenerated tenant model includes the new status and last-login fields.
- Design choice: `last_login_at` is nullable to represent tenants that have not logged in yet. The schema records the information; authentication and any inactivity policy will update/use it later.

## 2026-10-16 — Tenant creation under RLS

- `db/queries/tenants.sql`: adds a typed insert query that returns the tenant's ID, name, status, last-login timestamp, and creation timestamp.
- `internal/repository/tenant_repository.go`: creates a random UUID v4 in Go, attaches that actual ID to the request context, and calls the generated insert query inside `WithTenant`. The same ID is therefore used for the transaction-local RLS setting and the permanent tenant row.
- Design choice: the application generates the UUID before the insert because RLS `WITH CHECK` must compare the new row ID with `app.tenant_id`. A placeholder ID would not be saved and would not satisfy that invariant.
- Limitation: this repository method is not exposed through an API or signup service yet; it only defines the database operation for that later flow.

## 2026-10-16 — Dashboard authentication schema and typed queries

- `migrations/00004_create_dashboard_auth.sql`: adds tenant-owned dashboard users, GitHub identities, hashed opaque browser sessions, and hashed one-time email-verification tokens. Every tenant-owned auth table uses forced RLS with both read and write policies tied to `app.tenant_id`; composite foreign keys keep each identity/session/token attached to a user in the same tenant.
- `db/queries/dashboard_users.sql`, `user_identities.sql`, and `email_verification_tokens.sql`: define the auth SQL operations sqlc exposes as typed Go methods. Queries include explicit tenant filters in addition to relying on RLS.
- Design choice: store only SHA-256 hashes of high-entropy random session and verification tokens, so a database read does not reveal reusable bearer secrets. Password hashing and verification behavior will be implemented in the auth service, not SQL.
- Validation: local `sqlc generate` succeeded, `GOCACHE=/tmp/meteor-go-cache go build ./...` succeeded, and Goose applied migrations 4 and 5 to the local database.

## 2026-10-19 — Redis-backed dashboard sessions

- Dashboard sessions now live in Redis, keyed by tenant and user. Login, email verification, and GitHub sign-in replace the user's prior session; authenticated requests resolve the cached account and role without a PostgreSQL session query.
- Session entries expire after 48 hours of inactivity and renew after 24 hours. Logout and password reset invalidate the Redis entry.
- Migration `00008_move_dashboard_sessions_to_redis.sql` removes the PostgreSQL `dashboard_sessions` table and its session lookup function.

## 2026-10-16 — Restricted pre-auth lookup functions

- `migrations/00005_create_auth_lookup_functions.sql`: creates `meteor_auth_lookup`, a non-login, read-only role with `BYPASSRLS`, and four `SECURITY DEFINER` functions in a private schema. The functions resolve a password login, GitHub identity, active session, or unconsumed verification token to only the tenant and user identifiers needed to establish trusted context (plus the password hash and verification timestamp for login).
- `db/queries/auth_lookup.sql`: calls those functions through sqlc queries. Explicit casts preserve generated pgx UUID, text, and timestamp types even though sqlc cannot infer a table function's output columns on its own.
- Security boundary: `meteor_app` receives `EXECUTE` only on these functions, no membership in the lookup role, and no direct `SELECT` on auth tables. Functions use fixed `pg_catalog, pg_temp` search paths and fully qualified table names. All subsequent tenant data reads/writes still use `WithTenant` and forced RLS.
- Limitation: pre-auth lookup functions can reveal whether a login email or token exists to the API process, so auth endpoints must return generic errors and apply rate limits. The lookup role's bypass is intentionally limited to read-only functions and requires careful review if those functions change.
- Validation: `sqlc generate`, `GOCACHE=/tmp/meteor-go-cache go build ./...`, and local Goose migration application to version 5 succeeded.

## 2026-10-21 — Authentication and connection test foundation

- `internal/auth/service_test.go`: exercises signup input validation and password hashing, verified-login gating, session creation, malformed session rejection, and password-reset session revocation with test doubles for external boundaries.
- `internal/auth/redis_sessions_test.go`: optionally connects to a real Redis instance through `TEST_REDIS_URL`, then checks ping, session creation, resolution, invalid-token rejection, and revocation.
- `internal/database/pool_test.go`: checks malformed database URL rejection and optionally verifies a live PostgreSQL connection through `TEST_DATABASE_URL`.
- `internal/database/tenant_test.go`: verifies tenant database work fails closed when tenant context is missing, rejects a missing transaction callback, and optionally checks the transaction-local RLS setting against live PostgreSQL.
- Live-service checks skip when their `TEST_*_URL` variable is unset, so the default unit suite does not require Docker. No GitHub OAuth network flow is covered; it is reserved for frontend integration testing.
