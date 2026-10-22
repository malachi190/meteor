import { Suspense } from "react";
import type { Metadata } from "next";
import VerifyEmailForm from "./verify-email-form";

export const metadata: Metadata = {
  title: "Verify your email · Meteor",
  referrer: "no-referrer",
};

type VerifyEmailPageProps = {
  searchParams: Promise<{ token?: string | string[] }>;
};

/** Renders the dashboard email verification destination for links from signup emails. */
export default function VerifyEmailPage({ searchParams }: VerifyEmailPageProps) {
  return (
    <Suspense fallback={<AuthFlowFallback />}>
      <VerifyEmailContent searchParams={searchParams} />
    </Suspense>
  );
}

/** Resolves the one-time token inside Suspense before handing it to the confirmation form. */
async function VerifyEmailContent({ searchParams }: VerifyEmailPageProps) {
  const query = await searchParams;
  const token = Array.isArray(query.token) ? query.token[0] : query.token;

  return <VerifyEmailForm token={token} />;
}

/** Shows a compact placeholder while Next.js resolves the verification URL. */
function AuthFlowFallback() {
  return (
    <section aria-label="Loading email verification" className="w-full max-w-md rounded-[24px] bg-[#f7f6f2] px-6 py-10 sm:px-10">
      <p className="text-sm text-[#555451]">Loading verification…</p>
    </section>
  );
}
