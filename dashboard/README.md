# Meteor Dashboard

The dashboard frontend uses Next.js App Router, TypeScript, Tailwind CSS, and Axios. Its screen and design work follows the root `FRONTEND_WORKFLOW.md` and `DESIGN.md`.

## Local development

1. Copy `.env.example` to `.env.local` and set `NEXT_PUBLIC_API_BASE_URL` if the Go API is not at `http://localhost:8080`.
2. Set `DASHBOARD_ORIGIN=http://localhost:3000` in the backend environment. The Go API allows credentialed browser requests from this exact origin and from the configured `PUBLIC_BASE_URL` API origin.
3. Install dependencies and run the frontend from this directory:

   ```sh
   pnpm install
   pnpm dev
   ```

Open `http://localhost:3000`. The Axios browser client sends the HttpOnly dashboard session cookie with API requests.

Use `pnpm build` for a production build and `pnpm lint` for ESLint.

The page at `/` is a temporary scaffold placeholder. Dashboard screens will be designed in workflow phases.
