"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import axios from "axios";
import { apiClient } from "@/lib/api-client";

type AuthErrorResponse = {
  error?: {
    message?: string;
  };
};

type FormNotice = {
  kind: "success" | "error";
  message: string;
};

/** Confirms a signup token through the API and enters the newly verified workspace. */
export default function VerifyEmailForm({ token }: { token?: string }) {
  const router = useRouter();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [notice, setNotice] = useState<FormNotice | null>(
    token ? null : { kind: "error", message: "This verification link is missing its token. Request a fresh email and try again." },
  );

  /** Posts the token only after the user confirms the action, so link scanners cannot consume it. */
  async function handleVerify(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) return;

    setIsSubmitting(true);
    setNotice(null);
    try {
      await apiClient.post("/v1/auth/verify-email", { token });
      router.replace("/workspace");
    } catch (error) {
      const message = axios.isAxiosError<AuthErrorResponse>(error)
        ? error.response?.data?.error?.message
        : undefined;
      setNotice({
        kind: "error",
        message: message ?? "We couldn’t verify this email. Check your connection and try again.",
      });
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <section aria-labelledby="verify-title" className="w-full max-w-md rounded-[24px] bg-[#f7f6f2] px-6 py-8 sm:px-10 sm:py-10">
      <p className="text-xs font-semibold tracking-[0.12em] text-[#716f69]">EMAIL VERIFICATION</p>
      <h1 id="verify-title" className="mt-3 text-[30px] font-semibold leading-tight tracking-[-0.03em] text-[#070607]">
        Confirm your email
      </h1>
      <p className="mt-2 text-sm leading-6 text-[#555451]">
        Confirm the address you used to create your Meteor workspace. This link can be used once.
      </p>

      {token && (
        <form className="mt-8" onSubmit={handleVerify}>
          {notice && <FormNoticeView notice={notice} />}
          <button
            className="flex h-12 w-full items-center justify-center rounded-full bg-[#fc5000] px-6 text-sm font-semibold text-[#070607] transition-colors hover:bg-[#e94a00] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#070607] disabled:cursor-wait disabled:opacity-60"
            disabled={isSubmitting}
            type="submit"
          >
            {isSubmitting ? "Verifying…" : "Verify email"}
          </button>
        </form>
      )}

      {!token && notice && <div className="mt-7"><FormNoticeView notice={notice} /></div>}

      <p className="mt-7 border-t border-[#d7d5ce] pt-5 text-sm text-[#555451]">
        Need a fresh link?{" "}
        <Link className="font-medium text-[#070607] underline decoration-[#fc5000] underline-offset-4" href="/signup">
          Return to sign up
        </Link>
      </p>
    </section>
  );
}

/** Presents verification feedback with a live announcement for assistive technology. */
function FormNoticeView({ notice }: { notice: FormNotice }) {
  return (
    <div
      aria-live="polite"
      className={notice.kind === "error"
        ? "mb-5 rounded-xl border border-[#b94337]/40 bg-[#b94337]/5 px-4 py-3 text-sm leading-5 text-[#812b24]"
        : "mb-5 rounded-xl border border-[#858276]/35 bg-[#e2e2df]/55 px-4 py-3 text-sm leading-5 text-[#292826]"}
      role={notice.kind === "error" ? "alert" : "status"}
    >
      {notice.message}
    </div>
  );
}
