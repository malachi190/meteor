import { Suspense } from "react";
import AuthForm from "../auth-form";

type LoginPageProps = {
  searchParams: Promise<{ github_error?: string | string[] }>;
};

/** Shows the dashboard's email and password sign-in screen. */
export default function LoginPage({ searchParams }: LoginPageProps) {
  return (
    <Suspense fallback={<LoginFallback />}>
      <LoginContent searchParams={searchParams} />
    </Suspense>
  );
}

/** Reads OAuth feedback inside the suspense boundary so the rest of the route can prerender. */
async function LoginContent({ searchParams }: LoginPageProps) {
  const query = await searchParams;
  const githubError = Array.isArray(query.github_error) ? query.github_error[0] : query.github_error;

  return <AuthForm mode="login" githubError={githubError} />;
}

/** Provides a stable auth-card placeholder while Next resolves request-specific OAuth feedback. */
function LoginFallback() {
  return (
    <section aria-label="Loading sign-in form" className="w-full max-w-md rounded-[24px] bg-[#f7f6f2] px-6 py-10 sm:px-10">
      <p className="text-sm text-[#555451]">Loading sign-in…</p>
    </section>
  );
}
