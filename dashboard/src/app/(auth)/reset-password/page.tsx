import { Suspense } from "react";
import type { Metadata } from "next";
import ResetPasswordForm from "./reset-password-form";

export const metadata: Metadata = {
  title: "Choose a new password · Meteor",
  referrer: "no-referrer",
};

type ResetPasswordPageProps = {
  searchParams: Promise<{ token?: string | string[] }>;
};

/** Renders the dashboard destination for one-time password reset email links. */
export default function ResetPasswordPage({ searchParams }: ResetPasswordPageProps) {
  return (
    <Suspense fallback={<AuthFlowFallback />}>
      <ResetPasswordContent searchParams={searchParams} />
    </Suspense>
  );
}

/** Resolves the reset token inside Suspense before rendering the new password form. */
async function ResetPasswordContent({ searchParams }: ResetPasswordPageProps) {
  const query = await searchParams;
  const token = Array.isArray(query.token) ? query.token[0] : query.token;

  return <ResetPasswordForm token={token} />;
}

/** Shows a compact placeholder while Next.js resolves the reset URL. */
function AuthFlowFallback() {
  return (
    <section aria-label="Loading password reset" className="w-full max-w-md rounded-[24px] bg-[#f7f6f2] px-6 py-10 sm:px-10">
      <p className="text-sm text-[#555451]">Loading password reset…</p>
    </section>
  );
}
