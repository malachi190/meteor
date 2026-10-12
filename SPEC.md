# Meteor: Product & Technical Spec

**Product name:** Meteor **Version:** 0.2 (working source of truth) **Date:** 12 October 2026 **Owner:** Malachi

> **How to read this doc.** Each section is marked **Decided** (agreed) or **Proposed** (not yet reviewed in detail). Proposed items should be reviewed before building. Anything that needs checking is listed in Section 15. This working copy incorporates decisions made during spec review; the original v0.1 download is unchanged.

---

## 1. Overview

### 1.1 What we are building

An API platform that lets other developers add **voice calls, video calls, and live streaming** to their own apps. Developers sign up, get API keys, create rooms through our API, and connect their users using our SDK.

Think Daily.co, but tuned for **African users and networks**: low data usage, weak-connection resilience, local payments, and local support.

### 1.2 Goal of this project (Decided)

- **Primary goal: learning.** Build a real multi-tenant realtime platform end to end.
- **Secondary goal: validation.** Talk to a handful of Nigerian startups. If there is real interest, consider scaling. Not before.
- **No heavy infrastructure spend** until that validation happens.

### 1.3 Scope clarification (Decided)

"Phone calls" means **in-app voice and video only**. Real phone network calls (PSTN/SIP via carriers) are **out of scope**.

### 1.4 Non-goals for the first milestone

- PSTN / carrier phone calls
- AI voice agents
- Transcription or translation
- Multi-region and scale testing in the first milestone; these are deferred until product validation justifies the cost
- Payment processing; the first milestone defines a provider-neutral payment interface, with provider integrations later
- Recording, streaming, large audience playback, and production-scale load tests
- Mobile SDKs beyond a basic JS SDK (React Native / Flutter are later)

---

## 2. Target users

| User | Who they are | What they need |
| --- | --- | --- |
| **Tenant** | A company or developer building an app (telemedicine, tutoring, support, events) | API keys, simple docs, reliable calls, clear usage and billing |
| **End user** | The tenant's customer, joining a call on a phone or laptop, often on mobile data | Calls that work on weak networks and don't burn data |
| **Platform admin** (you) | Operates the platform | Visibility into tenants, usage, health, and costs |

---

## 3. Scale targets

### 3.1 Later validation target (Decided)

- **\~1,000 concurrent participants** across the platform.
- Test first at 100 participants, then at 1,000, after product validation justifies the spend.

### 3.2 Original estimate (for reference)

About 50 tenants, \~10 active, roughly 100k-150k API calls. API calls are cheap. **Media bandwidth is the real constraint.**

### 3.3 Bandwidth estimate (rough, to be verified by load test)

Assuming \~0.5-1 Mbps per video stream:

| Scenario | Outbound at 1,000 participants |
| --- | --- |
| 1:1 calls | \~0.5-1 Gbps |
| 4-person rooms | \~1.5-3 Gbps |
| Data moved per hour of test | \~300 GB or more |

**Implication:** a single 1 Gbps server cannot carry 1,000 people in group rooms. The design needs at least two media nodes, and any host must allow heavy sustained traffic.

---

## 4. Architecture

### 4.1 High-level design (Decided at gist level)

```
 Client apps (tenant's users)
        |  (JS SDK)
        v
 +---------------------------+
 |   Platform API (control   |   tenants, keys, rooms, tokens,
 |   plane)                  |   usage, billing, webhooks
 |   PostgreSQL + Redis      |
 +-------------+-------------+
               | mints join tokens, manages rooms
               v
 +---------------------------+
 |   LiveKit media nodes     |   audio/video routing (SFU)
 |   (Redis shared)          |
 |   - Lagos (primary, later) |
 |   - Johannesburg (later)   |
 +-------------+-------------+
               |
        Egress workers (recording / streaming)
```

### 4.2 Key choices

1. **Media server: LiveKit, self-hosted (Decided).** Open source, written in Go, includes rooms, tokens, webhooks, recording/streaming (Egress), external stream ingest (Ingress), and multi-node/multi-region support through Redis.
2. **Control plane and media are separate (Decided).** The API, database and Redis use little bandwidth and run on cheap infrastructure. Only media nodes need large network capacity.
3. **Later region targets (Decided):** Lagos primary, Johannesburg second. Lagos matches the main user base. Johannesburg can prove multi-region and provide a fallback after validation justifies the cost.
4. **Media hosting:** dedicated servers or hourly cloud instances with generous bandwidth. Avoid large cloud providers for media because data transfer fees are usually high (verify current prices).
5. **Start simple (Decided):** Docker Compose for local development. Move to Kubernetes only if needed, since LiveKit requires host networking, which adds complexity.
6. **Swappable media backend (Proposed):** the platform API should not leak LiveKit details to tenants, so the media layer could later move between LiveKit Cloud and self-hosted nodes without breaking customers.
7. **Go API structure (Decided):** use the Go standard library HTTP server and router, with a modular monolith organized around product areas. Avoid a full web framework for the first milestone.
8. **Service layers (Decided):** keep HTTP handlers, application services, and data repositories separate. Handlers translate HTTP requests and responses; services hold use-case decisions; repositories own SQL and database calls. Wire dependencies from `main` instead of looking them up through package globals.
9. **Configuration (Decided):** use godotenv to load a local `.env` file when present, then Koanf to load environment values once at startup into a typed config struct. Existing process environment values take precedence over `.env`. Pass the config to components that need it; other packages should not independently read process environment variables. Production can provide environment variables without a `.env` file.
10. **Logging (Decided):** use Zerolog. Request logs go to the CLI; error logs go to date-named files (for example, `meteor-errors-2026-10-12.log`). Keep the current and previous two calendar days, and automatically delete older matching files. Keep secrets and credentials out of logs. Add OpenTelemetry later, after the basic request and error logging is in place.

### 4.3 Tech stack (Proposed)

| Layer | Choice | Why |
| --- | --- | --- |
| Platform API | Go standard library | Matches LiveKit's server SDK, good for learning, strong concurrency; use `net/http` without a framework |
| Database | PostgreSQL | Relational data, ledger integrity |
| Database access | `pgxpool` + `sqlc` | PostgreSQL connection pooling with SQL-first, generated typed query code; no ORM |
| Schema migrations | Goose SQL migrations | Versioned SQL changes stay readable and are kept separate from query code |
| Configuration | godotenv + Koanf + typed config struct | Load local `.env` values, build typed configuration at startup, and pass it through the application layers |
| Logging | Zerolog | CLI request logs and file-based error logs |
| Tracing | OpenTelemetry later | Add distributed tracing after basic request/error logging is working |
| Cache / pub-sub | Redis | Required by LiveKit distributed mode, also rate limiting |
| Dashboard | Next.js + TypeScript | Familiar stack |
| JS SDK | TypeScript, wrapping `livekit-client` | Adds our low-bandwidth, data meter and analytics features |
| Payments | Provider interface (implementation later) | Keep payment-provider-specific behavior behind an adapter so services can be added or changed |
| Local/dev | Docker Compose | One command to run everything |


---

## 5. Multi-tenancy model (Proposed)

### 5.1 Hierarchy

```
Tenant (company)
  +-- Dashboard users (team members; email/password and GitHub OAuth sign-in)
  +-- Wallet (credit balance)
  +-- Projects (one per app the tenant builds)
        +-- API keys (test and live)
        +-- Rooms
        +-- Webhook endpoints
```

### 5.2 Rules

- **LiveKit has one shared key pair.** Tenant separation is enforced by **our API**, not by LiveKit.
- Public room names given by tenants are **mapped to internal room IDs** (e.g. `rm_<uuid>`), so two tenants can both create a room called "support" without clashing.
- Every tenant-owned database row carries `tenant_id`; project-owned rows also carry `project_id`.
- **Database-first tenant isolation (Decided):** enable PostgreSQL row-level security (RLS) on tenant-owned tables. Policies must restrict reads and writes by tenant, default-deny when tenant context is absent, and apply checks to inserted or updated rows. The application also scopes queries by tenant and project as a second layer.
- **RLS connection context (Proposed implementation detail):** set the authenticated tenant ID transaction-locally for each database transaction. The API's runtime database role must not own tenant tables or have `BYPASSRLS`; migrations use a separate privileged role. Use `FORCE ROW LEVEL SECURITY` where needed so table-owner access is also subject to policies.
- **Request tenant context:** attach tenant identity only after trusted authentication or tenant resolution; never derive RLS scope from an untrusted client-supplied tenant ID. Set `app.tenant_id` inside each database transaction so it resets on commit or rollback and cannot remain on a pooled connection.
- **Test and live environments are fully separate** data, keys and wallets, mirroring how payment providers work. Test mode never touches real money.
- **Wallets are per tenant and environment, shared across the tenant's projects.**
- **Quotas per project** (enforced when a join token is requested): max concurrent participants, max room size, max recording hours.

### 5.3 Authentication

- **Dashboard users:** self-serve tenant signup and email/password authentication, with GitHub OAuth as an additional sign-in option. Automatically link GitHub sign-in to an existing account only when GitHub provides a verified email matching the account's normalized email. Store the provider's stable user ID, not its mutable username. Account recovery and unlinking details remain to be specified.

- **Tenant server to our API:** secret API key in the `Authorization: Bearer` header. Keys look like `sk_test_...` / `sk_live_...`.
- **Public keys** (`pk_test_...`) may be used for limited client-side actions.
- Keys are shown once, stored **hashed**, with a visible prefix for identification.
- **End user to media:** short-lived **join token** (JWT) minted by our API on request. Contains room, participant identity, permissions, and expiry.

---

## 6. API design (Proposed)

REST, JSON, versioned under `/v1`. Idempotency keys supported on POST requests that create things.

### 6.1 Core endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| POST | `/v1/auth/signup` | Create a tenant and first dashboard user |
| POST | `/v1/auth/login` | Sign in with email and password |
| POST | `/v1/auth/logout` | End the dashboard session |
| GET | `/v1/auth/github` | Begin GitHub OAuth sign-in |
| GET | `/v1/auth/github/callback` | Complete GitHub OAuth sign-in |
| POST | `/v1/projects` | Create a project within the tenant |
| GET | `/v1/projects` | List the tenant's projects |
| GET | `/v1/projects/{id}` | Get project details |
| POST | `/v1/projects/{id}/keys` | Create a test or live API key for the project |
| GET | `/v1/projects/{id}/keys` | List key prefixes and statuses for the project |
| DELETE | `/v1/projects/{id}/keys/{key_id}` | Revoke a project API key |
| POST | `/v1/rooms` | Create a room (name, max participants, mode, options) |
| GET | `/v1/rooms/{id}` | Get room details and status |
| GET | `/v1/rooms` | List rooms |
| DELETE | `/v1/rooms/{id}` | End a room and disconnect everyone |
| POST | `/v1/rooms/{id}/tokens` | Mint a join token for a participant |
| GET | `/v1/rooms/{id}/participants` | List current participants |
| POST | `/v1/rooms/{id}/participants/{pid}/remove` | Kick a participant |
| POST | `/v1/rooms/{id}/recordings` | Start recording |
| POST | `/v1/recordings/{id}/stop` | Stop recording |
| GET | `/v1/recordings/{id}` | Get recording status and download link |
| POST | `/v1/rooms/{id}/streams` | Start live stream out (RTMP) |
| GET | `/v1/usage` | Usage summary (minutes, bandwidth) by date range |
| GET | `/v1/rooms/{id}/quality` | Call quality summary for a room |
| POST | `/v1/webhooks` | Register a webhook endpoint |
| GET | `/v1/wallet` | Credit balance and recent ledger entries |
| POST | `/v1/wallet/topups` | Start a payment to add credit (later milestone; provider adapter required) |

### 6.2 Room options (key ones)

- `mode`: `video`, `audio_only`, `stream`
- `max_participants`
- `low_bandwidth`: `auto` (default), `on`, `off`
- `record`: `false` (default) / `true`
- `empty_timeout`: seconds before an empty room closes

### 6.3 Webhooks sent to tenants

`room.started`, `room.ended`, `participant.joined`, `participant.left`, `recording.ready`, `wallet.low_balance`, `wallet.topup_succeeded`

- Signed with an HMAC secret per endpoint.
- Retried with backoff on failure; delivery attempts logged.

### 6.4 Errors

Consistent JSON error shape: `{ "error": { "code", "message", "request_id" } }`. Standard HTTP status codes. Rate limit headers on every response.

### 6.5 Rate limits

Per API key, enforced in Redis (token bucket). Limits are generous at MVP but must exist from the start.

---

## 7. Data model (Proposed)

### 7.1 Main tables

| Table | Key fields |
| --- | --- |
| `tenants` | id, name, status, created_at |
| `dashboard_users` | id, tenant_id, normalized email (unique), nullable password_hash, role |
| `user_identities` | id, dashboard_user_id, provider (`github`), provider_subject (stable provider user ID; unique with provider), created_at |
| `projects` | id, tenant_id, name, quotas (json) |
| `api_keys` | id, project_id, prefix, key_hash, env (`test`/`live`), last_used_at, revoked_at |
| `rooms` | id, project_id, public_name, livekit_room, mode, status, started_at, ended_at |
| `sessions` | id, room_id, participant_identity, joined_at, left_at |
| `usage_events` | id, project_id, room_id, type, quantity, unit, occurred_at |
| `wallets` | id, tenant_id, env, currency, balance (integer, minor units) |
| `wallet_ledger` | id, wallet_id, type (`topup`/`usage`/`adjustment`), amount, reference, created_at |
| `payments` | id, tenant_id, provider, provider_ref, amount, status, env |
| `recordings` | id, room_id, status, storage_key, duration, size |
| `webhook_endpoints` | id, project_id, url, secret_hash, events |
| `webhook_deliveries` | id, endpoint_id, event, status, attempts, last_error |
| `quality_samples` | room_id, session_id, ts, bitrate, packet_loss, rtt, jitter, bytes_sent, bytes_received |

### 7.2 Rules

- **Money is stored as integers** in minor units (kobo). Never floats.
- **The ledger is append-only.** Wallet balance can always be recomputed from it.
- Quality samples are high volume. Start in Postgres with time partitioning and a retention window (e.g. 30 days). Move to a time-series store only if needed.

### 7.3 Other storage

- **Redis:** LiveKit cluster state, rate limits, short-lived caches.
- **Object storage:** recordings. Use an S3-compatible service. Pick a cheap provider with sensible egress pricing (open decision).

---

## 8. Core feature specs

### 8.1 Rooms and tokens (core)

- Tenant server calls the API to create a room, then requests a join token for each participant.
- Token requests are checked against project quotas before issuing. Once live billing is implemented, they are also checked against the tenant's shared live wallet.
- Tokens expire quickly (default 10 minutes to join; session length set by room options).

### 8.2 Recording (core)

- Uses LiveKit Egress to record rooms to object storage.
- A `recording.ready` webhook fires when done.
- Recording minutes are metered and billed separately.

### 8.3 Live streaming (core, basic)

- Stream a room out to an RTMP destination using Egress.
- Bring in external streams (RTMP/WHIP) using Ingress.
- Large-audience playback (many viewers) is **out of scope for the MVP**. It is a separate bandwidth problem.

### 8.4 Low-bandwidth mode (differentiator)

**Goal:** calls stay usable on weak or expensive connections.

- Built on LiveKit's simulcast, adaptive stream and dynacast options.
- **Auto mode:** the SDK watches connection quality and drops video quality first, then switches to **audio-only** if the connection stays poor, and restores video when it recovers.
- **Data-saver presets:** `normal`, `saver`, `ultra_saver` (caps resolution, framerate and bitrate).
- **Receiver-driven:** only the video a participant is actually viewing is delivered at full quality.
- Tenant can force a mode through room options.

**Acceptance:** under a simulated 300-500 kbps limited connection, audio remains clear and the call does not drop.

### 8.5 Live data meter (differentiator)

**Goal:** users see how much mobile data a call is using.

- The SDK reads WebRTC stats (bytes sent and received) and exposes `onDataUsage` events and a ready-made `DataMeter` component.
- Shows MB used this call, current rate, and an optional warning threshold set by the tenant.

**Acceptance:** displayed usage is within \~10% of the actual measured traffic in a test call.

### 8.6 Call quality analytics (differentiator)

**Goal:** tenants can see why a call was bad.

- The SDK samples stats every few seconds (bitrate, packet loss, round-trip time, jitter, resolution) and sends them to our quality ingest endpoint.
- The dashboard shows per-room and per-participant timelines and a simple quality score.
- `GET /v1/rooms/{id}/quality` returns the same summary through the API.

**Acceptance:** after a test call with an artificially throttled participant, the dashboard clearly shows when and how quality dropped.

### 8.7 Fast reconnect (supporting)

- Rely on LiveKit's reconnection behavior, tested against network switches (Wi-Fi to mobile data) and short signal loss. Verify it, tune it, document it.

---

## 9. Billing and payments

### 9.1 Model (Proposed)

- **Prepaid credit wallet** per tenant, per environment.
- Usage is **metered** (participant-minutes, recording-minutes, streaming-minutes, optionally bandwidth) and debited from the wallet.
- If the balance runs out in live mode, new join tokens are refused. Existing calls are allowed to finish.
- Pricing numbers are **TBD**. Rule: price per minute must comfortably exceed the real bandwidth and compute cost.
- **No large free tier.** Free usage costs real bandwidth. Test mode is free and uses no real media quota beyond reasonable limits.

### 9.2 Payment provider boundary (Decided)

- Define an internal payment-provider interface so Paystack, Flutterwave, or another service can be added without coupling wallet and billing logic to one provider.
- Implement no payment provider in the first milestone. Select a provider later (Paystack and Flutterwave are candidates) and use the same interface for test and live.

### 9.3 Top-up flow

1. In the later payment milestone, tenant calls `POST /v1/wallet/topups` (or clicks "Add credit" in the dashboard).
2. API creates a pending `payments` row and initializes a transaction through the selected provider adapter.
3. Tenant completes payment through the selected provider.
4. The provider calls our webhook. We **verify the signature**, then verify the transaction with the provider, then credit the wallet through a ledger entry. The handler must be **idempotent** (the same event twice credits once).
5. We fire `wallet.topup_succeeded` to the tenant.

### 9.4 Metering

- Usage events are created from LiveKit webhooks (participant joined and left, room finished, egress ended).
- A background job turns usage events into ledger debits at set intervals.
- Reconcile periodically: compare our usage totals against LiveKit's own records.

---

## 10. Security

- API keys hashed at rest; shown once; revocable; scoped by environment.
- Join tokens are short-lived and scoped to one room and identity.
- TLS everywhere. Media is encrypted by WebRTC (DTLS-SRTP) by default.
- Webhooks signed; endpoint secrets encrypted at rest because delivery requires the original secret for HMAC signing.
- Strict tenant scoping in every query (tests must prove one tenant cannot read another's data).
- Rate limiting and basic abuse protection on all public endpoints.
- Secrets in environment variables or a secrets manager, never in the repo.
- Recordings access via expiring signed URLs only.

---

## 11. Observability and operations

- **Structured logs** with `request_id`, `tenant_id`, `project_id`.
- **Metrics** (Prometheus-style): API latency and errors, active rooms, active participants, per-node bandwidth, CPU, and packet loss.
- **Dashboards and alerts** for node saturation (bandwidth and CPU), webhook failures, payment webhook failures, and wallet reconciliation mismatches.
- **Health endpoints** on the API and each media node.
- **Graceful drain** when deploying media nodes, so live calls are not dropped (supported by LiveKit).

---

## 12. Infrastructure and cost plan

### 12.1 Environments

| Environment | Setup | Rough cost |
| --- | --- | --- |
| Local dev | Docker Compose, LiveKit dev mode | Free |
| Staging | One small VPS (API, Postgres, Redis, one LiveKit node) | \~$20-60/month |
| Load test | Hourly cloud instances (e.g. Johannesburg) plus a Lagos node, torn down after | Likely tens of dollars per test |
| Production (only after validation) | Lagos and Johannesburg media nodes, separate control plane | Several hundred dollars/month or more |

All prices are rough and must be checked before committing.

### 12.2 Hosting notes

- Standard Lagos VPS plans often cap monthly transfer (for example 1-2 TB, then throttled). They are **not suitable for media**.
- Lagos dedicated servers exist at roughly $400+/month for 1 Gbps with fair-usage limits (ballpark; read the fair-usage terms).
- Johannesburg has several providers (Vultr, Google Cloud, Oracle, Fly.io) with hourly billing.

### 12.3 Cost rules

- Do not buy always-on production servers until at least a few tenants are interested.
- Keep every load test short and torn down after.
- Track cost per participant-minute from the first test so pricing is based on facts.

---

## 13. Roadmap

| Phase | Goal | Deliverables |
| --- | --- | --- |
| **0. Foundations** | Environment and first call | Docker Compose with LiveKit, Postgres, Redis; a hello-world 1:1 call in a browser |
| **1. Platform core** | Self-serve multi-tenant product | Email/password and GitHub OAuth signup/sign-in, basic dashboard, tenants, projects, project API keys (test/live), rooms, join tokens, quotas, webhooks, provider-neutral payment interface |
| **2. Billing foundation** | Metering and money flow | Wallet and ledger model, usage events, implement a payment provider through the interface |
| **3. Differentiators** | What makes it stand out | Low-bandwidth mode, data meter, quality analytics, dashboard views |
| **4. Recording and streaming** | Extra media features | Egress recording, RTMP out, ingest |
| **5. Product validation** | Test demand and local networks | Demo app, docs site, real Lagos network tests, 5-10 conversations with Nigerian startups, decision on scaling |
| **6. Scale validation** | Prove capacity if justified | 100 then 1,000 participant tests; second region (Johannesburg); record bandwidth, CPU, packet loss, and cost per participant-minute |

---

## 14. Validation checklist (acceptance criteria)

The first platform milestone is complete when the core platform criteria pass. The remaining groups are acceptance criteria for later milestones, not prerequisites for starting development.

**Platform**

- [ ] A new tenant can sign up, create a project, and get test API keys.
- [ ] The dashboard supports email/password and GitHub sign-in, and tenants can manage their own projects and keys.
- [ ] Creating a room and joining with a minted token works from the JS SDK.
- [ ] Payment providers can be integrated behind the provider interface without changing wallet and usage logic.
- [ ] Tenant A cannot see or affect Tenant B's rooms, keys, usage, or payments (verified by tests).
- [ ] Webhooks are delivered, signed, retried on failure, and logged.
- [ ] Rate limits and quotas are enforced.

**Payments**

- [ ] A test-mode Paystack top-up credits the wallet exactly once, even if the webhook arrives twice.
- [ ] Usage debits the wallet correctly; ledger totals always match the balance.
- [ ] With zero balance in live mode, new join tokens are refused.
- [ ] Switching from test to live requires only a configuration change.

**Differentiators**

- [ ] Low-bandwidth mode keeps audio clear at 300-500 kbps and recovers video afterwards.
- [ ] Data meter is within \~10% of measured traffic.
- [ ] Quality dashboard shows when and how a throttled participant degraded.
- [ ] Reconnect works across a Wi-Fi to mobile data switch.

**Scale and cost**

- [ ] 100 concurrent participants run stably on a small setup.
- [ ] 1,000 concurrent participants run stably across at least two media nodes, with recorded bandwidth, CPU, and packet loss.
- [ ] A second region (Johannesburg) serves calls correctly alongside Lagos.
- [ ] Cost per participant-minute is measured and documented.

**Validation**

- [ ] At least 5 conversations with potential tenants completed, with notes on interest, objections, and pricing feedback.

---

## 15. Open questions and risks

**Open decisions**

1. Object storage provider for recordings.
2. Pricing: per-minute rates, minimum top-up, and what test mode allows.
3. Product name and domain.
4. Whether to also support Flutterwave later.
5. Dashboard account linking and recovery behavior for email/password and GitHub OAuth.
6. Account recovery and provider unlinking behavior.

**Risks**

- **Network quality in Nigeria:** how well local mobile networks connect to each hosting location is unknown. Must be tested with real Lagos connections before committing to hosting.
- **Bandwidth cost:** the largest cost and the easiest way to lose money. Mitigated by prepaid credit and no big free tier.
- **Dependency on LiveKit:** the company's marketing focuses on AI voice agents. The server is open source, so risk is manageable, but roadmap changes could matter.
- **Reliability expectations:** tenants will expect uptime. A solo developer should be honest about this during validation.
- **Competition:** LiveKit Cloud, Daily, Agora and others. The Africa-specific features are the differentiation and must be proven valuable.

---

## 16. Glossary

- **SFU:** a server that forwards audio and video between participants without mixing them.
- **Simulcast:** sending the same video at several quality levels so the server can pick the right one per viewer.
- **Egress / Ingress:** LiveKit's services for recording or streaming out, and for bringing external streams in.
- **Join token:** a short-lived pass that lets one participant enter one room.
- **Ledger:** an append-only record of every credit and debit on a wallet.
- **Tenant:** a customer of the platform (a company building an app).
- **Row-level security (RLS):** database rules that filter which rows a database role can read or change, even when a query does not include the tenant filter.
- **Transaction-local tenant context:** the tenant ID attached to one database transaction so RLS can apply the right policy; it expires with that transaction and does not carry over to another pooled connection use.
- **OAuth identity linking:** associating a user's account with a sign-in provider, such as GitHub, so either sign-in method reaches the same account.
