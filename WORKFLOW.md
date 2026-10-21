# Meteor Build Workflow

This is the milestone checklist for building Meteor from the agreed spec. We update task status as work is completed. A phase is complete only when its **Goal achieved when** conditions are met; finishing the listed code alone does not automatically close a phase.

## Phase 0 — Product and local development foundation

**Status: Complete**

- [x] Agree on the v0.2 product and architecture direction in `SPEC.md`.
- [x] Create the Go standard-library HTTP server and basic health endpoint.
- [x] Start PostgreSQL locally through Docker Compose, with host port 5433 mapped to container port 5432.
- [x] Add the initial tenant/project schema and RLS policies.
- [x] Add the restricted `meteor_app` runtime role and separate migration credentials.
- [x] Connect the API through `pgxpool` and verify the runtime role can connect.

**Goal achieved when:** the API starts only after its runtime database connection succeeds, and its database role is separate from the migration/admin role.

## Phase 1 — API foundations and tenant-safe database access

**Status: In progress**

- [x] Add godotenv for optional local `.env` loading and a Koanf-backed typed config struct; load only required settings at startup and pass config from the composition root.
- [x] Establish handler → service → repository boundaries and wire dependencies from the composition root.
- [x] Add Zerolog request logs to the CLI and error-only logs to a file; keep credentials and secrets out of log fields.
- [x] Rotate error logs into date-named files and retain the latest three calendar days.
- [x] Add a typed tenant ID to Go request context using a private context key.
- [x] Add a transaction helper that reads the tenant ID from context and sets `app.tenant_id` transaction-locally.
- [ ] Attach tenant context only from trusted authentication or tenant-resolution code; never trust a tenant ID supplied directly by a client.
- [ ] Add the first tenant-scoped database operation with explicit tenant/project filters.
- [ ] Confirm queries without tenant context fail closed and cross-tenant reads/writes are denied by RLS.

**Goal achieved when:** configuration and logging are wired through clear service layers, and a tenant-authenticated request runs database work in a transaction with the correct tenant setting; missing or mismatched tenant scope cannot access another tenant's rows.

**Deferred:** OpenTelemetry tracing until baseline request and error logging is working.

## Phase 2 — SQL query layer and tenant onboarding

**Status: In progress**

- [x] Set up sqlc and keep hand-written SQL queries alongside generated Go query code.
- [x] Add a constrained tenant status and nullable last-login timestamp through a new migration.
- [x] Build project create/list/get operations on top of the tenant transaction helper.
- [x] Add tenant creation with an initial tenant ID that can be set as the RLS context.
- [x] Design signup so it creates the tenant and first dashboard user atomically.
- [x] Apply and verify the auth schema migration for dashboard users, GitHub identities, dashboard sessions, and one-time email verification tokens.
- [x] Implement constrained pre-auth lookup functions for login, GitHub identity, session, and email verification resolution.

**Goal achieved when:** tenant signup writes run under RLS and auth lookups before tenant resolution use only the restricted database functions, with typed SQL code and no ORM dependency.

## Phase 3 — Dashboard authentication

**Status: In progress**

- [x] Implement email/password signup with required email verification and Argon2id password hashing.
- [x] Implement secure session creation, renewal, and logout.
- [x] Add GitHub OAuth with state validation and verified primary-email checks.
- [x] Link GitHub login to an existing account only when its verified email matches the normalized account email.
- [x] Attach the authenticated tenant ID to dashboard request context through session middleware.
- [x] Add basic per-process IP rate limits for signup, login, verification, and OAuth endpoints.
- [x] Add one-time password reset links (one-hour expiry), with tenant-scoped atomic password update and revocation of all active sessions.
- [ ] Replace per-process limits with shared Redis limits before running multiple API instances.

**Goal achieved when:** signup, login, logout, and GitHub sign-in resolve a verified dashboard identity and tenant scope without accepting tenant identity from an untrusted request field.

## Phase 4 — Self-serve projects and API keys

**Status: Not started**

- [ ] Add project create/list/detail operations, scoped to the signed-in tenant.
- [ ] Create test/live API keys per project; show raw secrets once and store only hashes plus visible prefixes.
- [ ] Add key listing and revocation.
- [ ] Build the first basic tenant dashboard for signup, projects, and key management.

**Goal achieved when:** a tenant can manage its own projects and keys in the dashboard, and cannot read or change another tenant's records.

## Phase 5 — Realtime control plane

**Status: Not started**

- [ ] Integrate the LiveKit server SDK behind a Meteor-owned media interface.
- [ ] Add project API-key authentication for server-to-server requests.
- [ ] Implement tenant/project-scoped room creation, listing, inspection, and ending.
- [ ] Mint short-lived participant join tokens with room and permission limits.
- [ ] Enforce project quotas when issuing join tokens.

**Goal achieved when:** a tenant can create a room and mint a constrained join token through Meteor without learning shared LiveKit credentials or accessing another tenant's rooms.

## Phase 6 — Usage, wallets, and payment boundary

**Status: Not started**

- [ ] Record usage events with tenant, project, room, unit, and timestamp attribution.
- [ ] Add test/live wallets shared across projects within a tenant and environment.
- [ ] Keep balances in integer minor units and ledger entries append-only.
- [ ] Define a provider-neutral payment interface and fake/test provider; defer real provider integrations.
- [ ] Add usage summaries and wallet history endpoints.

**Goal achieved when:** usage can be attributed and reconciled, and a tenant's projects share the correct environment-specific wallet without floating-point money calculations.

## Phase 7 — Webhooks and optional media features

**Status: Later MVP increment**

- [ ] Add signed webhook endpoints and retryable delivery records.
- [ ] Add recording and RTMP streaming controls only after the core room/token flow is stable.
- [ ] Add participant management and call-quality summaries.

**Goal achieved when:** these optional features have tenant-scoped storage, secure secrets, and clear delivery/state tracking.

## Phase 8 — Verification and MVP readiness

**Status: Not started**

- [ ] Add unit, database integration, and API tests for success, invalid input, missing context, and cross-tenant access.
- [ ] Review auth, API-key, RLS, session, and payment boundaries against the spec.
- [ ] Add production configuration, migrations workflow, log file retention, and health/readiness behavior.
- [ ] Document local setup and the first end-to-end tenant/project/room flow.
- [ ] Validate the basic JavaScript SDK and tenant onboarding flow with a small number of developers.

**Goal achieved when:** the core onboarding-to-room flow is reproducible, tenant isolation and error cases are verified, and the small-scale product validation can begin without premature multi-region infrastructure.

## Phase 9 — Post-validation scale work

**Status: Deferred until product validation**

- [ ] Measure media bandwidth and call quality under realistic network conditions.
- [ ] Evaluate Redis-backed rate limits and distributed LiveKit deployment.
- [ ] Revisit OpenTelemetry tracing after baseline CLI request logs and file-based error logs are established.
- [ ] Add regional media capacity, object storage, and deeper observability only when validated demand justifies the cost.
- [ ] Load test progressively toward 100 and then 1,000 concurrent participants.

**Goal achieved when:** measured demand and operating cost support the next scale step and the chosen infrastructure is validated by load and quality measurements.

## Working rules

- Work through one small, explainable increment at a time.
- Update the current task and phase status when implementation is complete and its goal conditions are met.
- Log implementation logic in `SOURCE_LOGS.md`; documentation-only edits do not need a source-log entry.
- Do not mark a phase complete just because its checklist has code checked off; use its stated goal conditions.
