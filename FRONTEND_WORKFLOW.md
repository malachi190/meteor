# Meteor Frontend Workflow

This checklist guides the dashboard build in small, reviewable phases. A phase is complete only when its stated goal is met, not merely when its checklist is checked.

## Current status

**Phases 1–2 — In progress: visual foundation and authentication entry screens**

- [x] Define the visual reference in `DESIGN.md` using Caldera.
- [x] Install the `design-taste-frontend` skill for dashboard work.
- [x] Manually test the project and API-key endpoints (as reported by the user).
- [x] Review and approve the initial frontend workflow.
- [x] Confirm Next.js App Router, TypeScript, Tailwind CSS, Axios, and pnpm; do not use shadcn/ui.
- [x] Choose browser Axios requests with session credentials and an exact-origin backend CORS allowlist.
- [x] Approve `dashboard/` at the repository root for the frontend.
- [x] Agree on the screen areas and use implemented/planned labels in the API documentation.
- [x] Verify the frontend can reach the local API with the session cookie (confirmed in the browser).

**Repository layout:** the frontend lives under `dashboard/` at the repository root; existing backend directories stay in place.

## Planned dashboard screen map

- **Authentication:** signup, sign-in, email verification, password reset, and GitHub sign-in.
- **Dashboard home:** Codex will propose its content after the project, key, profile/settings, and API documentation work reveals which summary and next actions would be useful. Do not invent metrics.
- **Projects and API keys:** tenant-scoped project list/detail/create and test/live key creation, one-time secret reveal, listing, and revocation.
- **API documentation:** an in-dashboard documentation area organized into **Video**, **Audio**, and **Streaming**. Each area explains the relevant connection flow and provides practical integration instructions, code examples, API reference, authentication guidance, and errors.
- **Profile:** the signed-in user's identity and workspace context, with editable fields only when the backend supports them.
- **Settings:** account and workspace settings, with each control backed by an available API operation. Do not present a nonfunctional preference or security control.

The documentation is part of the product interface, not marketing content. The screens should help developers integrate Meteor into their own applications and understand which capabilities are ready to use.

## Phase 0 — Product scope and frontend foundation

**Status: Complete**

- [x] Confirm the Caldera visual reference and app-screen direction.
- [x] Confirm projects, API keys, profile, settings, and Video/Audio/Streaming documentation as dashboard areas.
- [x] Identify current auth, project, and key routes; room/token and media capabilities remain future backend work.
- [x] Select Next.js App Router, TypeScript, Tailwind CSS, Axios, and pnpm; exclude shadcn/ui.
- [x] Configure a separate browser API origin with credentialed Axios requests and explicit backend CORS.
- [x] Scaffold the frontend under `dashboard/` and document its start/build commands.
- [x] Verify API requests and cookie-based sessions in the local browser setup.

**Goal achieved when:** the frontend starts locally, reaches the API with the dashboard session cookie, and the toolchain, screen scope, and API availability labels are established.

## Phase 1 — Dashboard shell and visual foundation

**Status: In progress**

- [x] Read `.agents/skills/design-taste-frontend/references/app-screens.md` before designing dashboard screens; follow its app-screen workflow, not its landing-page workflow.
- [ ] Translate `DESIGN.md` into dashboard tokens for colors, typography, spacing, borders, radii, and interactive states.
- [x] Load DM Sans for interface text and Bebas Neue for the wordmark/display role.
- [ ] Keep Caldera's warm Pumice/Limestone surfaces, Obsidian text, and restrained Ember accent; adapt typography and surface density for frequent use.
- [ ] Build a product shell that identifies Meteor and the signed-in workspace, with navigation for the agreed product areas and account controls.
- [ ] Establish shared patterns for forms, tables/lists, documentation navigation, code samples, dialogs, notices, and status labels.
- [ ] Define responsive behavior for desktop, tablet, and phone before screen-specific layouts spread.

**Dashboard-specific design limits:** do not carry over a marketing hero, 189px display headline, decorative halftone hero, partner strip, or promotional conversion layout. Use the visual identity in service of work tasks and technical reading. Keep the interface flat and information-led; reserve shadows for floating UI such as menus and dialogs.

**Goal achieved when:** the shell is recognizable as Meteor, fits the Caldera direction, supports keyboard and narrow-screen use, and provides reusable patterns for both operational screens and API documentation.

## Phase 2 — Authentication and workspace entry

**Status: In progress**

- [x] Build signup and sign-in forms for the existing email/password API.
- [x] Show signup's verification-required state and provide a resend-verification action.
- [x] Bring email-verification link completion into the dashboard; signup emails now link to the dashboard confirmation page.
- [x] Build the password-reset request form.
- [x] Bring password-reset link completion into the dashboard; reset emails now link to the dashboard password form.
- [x] Implement sign-out and session-expiry handling using the server-managed HttpOnly cookie.
- [x] Add the GitHub OAuth action and send successful callbacks to the signed-in workspace entry.
- [x] Test GitHub OAuth redirect/callback in the browser after provider credentials are configured (user confirmed).
- [x] Show validation, loading, success, and error states for the implemented flows.

**Security rules:** do not store the dashboard session in local storage or JavaScript-readable storage. The backend session establishes the tenant; never use a tenant ID supplied by the browser to authorize data access.

**Goal achieved when:** a user can enter and leave the dashboard through the existing auth flows, see useful feedback for failure states, and arrive scoped to their authenticated workspace.

## Phase 3 — Projects and API keys

**Status: Not started**

- [ ] Integrate project listing, creation, detail, and navigation.
- [ ] Integrate test and live key creation for a selected project.
- [ ] Display a raw key only in the one-time creation flow and provide a clear copy action.
- [ ] Integrate key listing using only the metadata returned after creation.
- [ ] Integrate key revocation with explicit confirmation and a clear revoked state.
- [ ] Provide helpful first-project and no-key empty states, plus loading, validation, and API error states.
- [ ] Keep project data scoped through the authenticated API session.

**Security rules:** never write raw API keys to logs, analytics, persistent browser storage, or later key-list responses. Make the one-time nature of the secret clear before or at creation.

**Goal achieved when:** a signed-in tenant can manage its own projects and keys through the dashboard without exposing another tenant's data or secrets.

## Phase 4 — Profile and settings

**Status: Not started**

- [ ] Design a profile screen around the account and workspace data the authenticated API currently returns.
- [ ] Identify which profile fields can be edited today and keep unsupported fields read-only until backend operations exist.
- [ ] Group settings by account, workspace, and security only where each section has a real user need.
- [ ] Connect settings controls to existing backend operations; record backend gaps for controls that need new endpoints.
- [ ] Include sign-out and links to supported password recovery/security actions.
- [ ] Show save, validation, loading, and failure states for editable settings.

**Goal achieved when:** users can inspect their account/workspace context and use the settings that the backend actually supports, with no misleading or inert controls.

## Phase 5 — API documentation and integration guides

**Status: Not started**

- [ ] Build a documentation home and navigation grouped into Video, Audio, and Streaming.
- [ ] For each supported capability, explain the use case, prerequisites, server-side API-key authentication, and test/live environment behavior.
- [ ] Show the integration sequence from creating/configuring a room through issuing a short-lived join token and connecting an end user with the SDK, where those backend operations exist.
- [ ] Include practical server and client code examples, endpoint methods/paths, request and response examples, errors, and relevant limits.
- [ ] Explain audio-only and low-bandwidth behavior when it is implemented and its exact controls are known.
- [ ] Explain outbound RTMP, inbound RTMP/WHIP, and recording only when the corresponding backend capabilities are available and their contracts are confirmed.
- [ ] Label each capability clearly as available, planned, or unavailable; never document a proposed endpoint as working API behavior.
- [ ] Keep code examples consistent with the real API and supported SDK versions; provide copy controls without placing secrets into persistent browser storage.
- [ ] Add cross-links between related guides, endpoint reference, API-key setup, and error guidance.

**Goal achieved when:** a developer can choose Video, Audio, or Streaming, understand the supported integration path, copy accurate examples, and distinguish shipped API behavior from roadmap items.

## Phase 6 — Dashboard home (content proposal delegated to Codex)

**Status: Deferred until the core dashboard areas clarify the most useful home-screen content**

- [ ] Map the real project, key, account, and integration tasks to identify the home screen's main question.
- [ ] Propose a useful first-run and returning-user layout using available data and links to the next action.
- [ ] Identify the most frequent next action and place it directly on the screen.
- [ ] Include only useful project, integration, or account information that can be computed from real data.
- [ ] Review the proposal with the user before implementing the home screen.

**Goal achieved when:** the proposed home screen has a clear purpose, uses available data honestly, and points to a useful next action. This phase does not block building the other agreed screens.

## Phase 7 — Interaction, accessibility, and responsive review

**Status: Not started**

- [ ] Review each screen for its primary question, main user task, real data, and clear next action.
- [ ] Check keyboard navigation, visible focus, semantic labels, contrast, and dialog behavior.
- [ ] Review the desktop layout at 1440×900 so the important content and first rows are visible without unnecessary scrolling.
- [ ] Review tablet and phone layouts, including a 390px viewport; prevent page-level horizontal scrolling.
- [ ] Make empty, zero, loading, stale, and error states truthful and consistent with API responses.
- [ ] Use believable sample content only where needed during development; keep totals derived from displayed rows and remove placeholders from real API views.

**Goal achieved when:** the main project, key, account, and documentation workflows remain clear and usable across supported viewport sizes and common interaction states.

## Phase 8 — Frontend integration handoff

**Status: Not started**

- [ ] Verify the documented local setup from a clean start.
- [ ] Walk through authentication, project creation/detail, test/live key lifecycle, and profile/settings flows supported by the backend.
- [ ] Review every API guide against implemented backend routes, request/response behavior, SDK examples, and availability labels.
- [ ] Check session renewal/expiry, OAuth callback, and API-unavailable behavior.
- [ ] Record known gaps, configuration needs, and deferred backend capabilities.
- [ ] Update `WORKFLOW.md` when its dashboard milestone conditions are actually met.

**Goal achieved when:** the frontend can be run and its supported dashboard flows reproduced against the current backend, while its API guides accurately describe shipped behavior and identify roadmap items.

## Design and implementation rules

- Treat `DESIGN.md` as the Caldera visual reference and apply only elements that support an operational dashboard or technical documentation.
- Use the installed `design-taste-frontend` skill's app-screen process for dashboard views. Before a screen is designed, inventory its data and jobs, state its one main question and finding, choose a product-specific visual element, and review the screen in a browser at desktop and phone sizes.
- Do not copy a generic dashboard template. Use actual Meteor objects, workspace scope, project/key data, and actions supported by the backend.
- Do not invent metrics, statuses, or API capabilities. Zeros stay zero; missing values are identified as unavailable rather than fabricated.
- Keep the interface task-focused: the most frequent action should be immediately available, while secondary details remain easy to find.
- Explain unfamiliar frontend concepts and important trade-offs in plain language before implementation, then build one small increment at a time.
- Ask for review at the end of each phase before moving to the next phase.
- Update this file's status and checklist as phases are completed. Keep backend workflow status in `WORKFLOW.md` synchronized only when its milestone conditions are met.
- Do not commit or publish frontend changes unless requested.
