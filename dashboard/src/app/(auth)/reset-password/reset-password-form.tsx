"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import axios from "axios";
import { apiClient } from "@/lib/api-client";
import PasswordInput from "@/components/password-input";

type AuthErrorResponse = {
  error?: {
    message?: string;
  };
};

type FormNotice = {
  kind: "success" | "error";
  message: string;
};

const passwordLimits = {
  minLength: 12,
  maxLength: 128,
};

/** Updates the account password with the one-time token carried by its reset link. */
export default function ResetPasswordForm({ token }: { token?: string }) {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isComplete, setIsComplete] = useState(false);
  const [notice, setNotice] = useState<FormNotice | null>(
    token ? null : { kind: "error", message: "This reset link is missing its token. Request a fresh email and try again." },
  );

  /** Validates matching passwords and consumes the reset token through the API. */
  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) return;

    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const password = String(form.get("password") ?? "");
    const confirmation = String(form.get("confirm_password") ?? "");
    if (password !== confirmation) {
      setNotice({ kind: "error", message: "The passwords do not match. Check both fields and try again." });
      return;
    }

    setIsSubmitting(true);
    setNotice(null);
    try {
      await apiClient.post("/v1/auth/password-reset", { token, password });
      setIsComplete(true);
      setNotice({ kind: "success", message: "Your password has been updated. Sign in with your new password." });
      formElement.reset();
    } catch (error) {
      const message = axios.isAxiosError<AuthErrorResponse>(error)
        ? error.response?.data?.error?.message
        : undefined;
      setNotice({
        kind: "error",
        message: message ?? "We couldn’t update your password. Check your connection and try again.",
      });
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <section aria-labelledby="reset-title" className="w-full max-w-md rounded-[24px] bg-[#f7f6f2] px-6 py-8 sm:px-10 sm:py-10">
      <p className="text-xs font-semibold tracking-[0.12em] text-[#716f69]">ACCOUNT RECOVERY</p>
      <h1 id="reset-title" className="mt-3 text-[30px] font-semibold leading-tight tracking-[-0.03em] text-[#070607]">
        Choose a new password
      </h1>
      <p className="mt-2 text-sm leading-6 text-[#555451]">
        Use a password between 12 and 128 characters. After it is changed, you’ll need to sign in again on your devices.
      </p>

      {token && !isComplete && (
        <form className="mt-8 space-y-5" onSubmit={handleSubmit}>
          <label className="block text-sm font-medium text-[#292826]">
            New password
            <PasswordInput
              autoComplete="new-password"
              className="h-12 w-full rounded-xl border border-[#c9c8c3] bg-[#f7f6f2] px-4 text-base text-[#070607] outline-none transition focus:border-[#070607] focus:ring-2 focus:ring-[#fc5000]/25"
              maxLength={passwordLimits.maxLength}
              minLength={passwordLimits.minLength}
              name="password"
              required
              type="password"
            />
          </label>

          <label className="block text-sm font-medium text-[#292826]">
            Confirm new password
            <PasswordInput
              autoComplete="new-password"
              className="h-12 w-full rounded-xl border border-[#c9c8c3] bg-[#f7f6f2] px-4 text-base text-[#070607] outline-none transition focus:border-[#070607] focus:ring-2 focus:ring-[#fc5000]/25"
              maxLength={passwordLimits.maxLength}
              minLength={passwordLimits.minLength}
              name="confirm_password"
              required
              type="password"
            />
          </label>

          {notice && <FormNoticeView notice={notice} />}

          <button
            className="flex h-12 w-full items-center justify-center rounded-full bg-[#fc5000] px-6 text-sm font-semibold text-[#070607] transition-colors hover:bg-[#e94a00] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#070607] disabled:cursor-wait disabled:opacity-60"
            disabled={isSubmitting}
            type="submit"
          >
            {isSubmitting ? "Updating password…" : "Update password"}
          </button>
        </form>
      )}

      {(!token || isComplete) && notice && (
        <div className="mt-7">
          <FormNoticeView notice={notice} />
        </div>
      )}

      <p className="mt-7 border-t border-[#d7d5ce] pt-5 text-sm text-[#555451]">
        {isComplete ? "Ready to continue? " : "Need another reset link? "}
        <Link
          className="font-medium text-[#070607] underline decoration-[#fc5000] underline-offset-4"
          href={isComplete ? "/login" : "/forgot-password"}
        >
          {isComplete ? "Return to sign in" : "Request a new link"}
        </Link>
      </p>
    </section>
  );
}

/** Presents password reset feedback with a live announcement for assistive technology. */
function FormNoticeView({ notice }: { notice: FormNotice }) {
  return (
    <div
      aria-live="polite"
      className={notice.kind === "error"
        ? "rounded-xl border border-[#b94337]/40 bg-[#b94337]/5 px-4 py-3 text-sm leading-5 text-[#812b24]"
        : "rounded-xl border border-[#858276]/35 bg-[#e2e2df]/55 px-4 py-3 text-sm leading-5 text-[#292826]"}
      role={notice.kind === "error" ? "alert" : "status"}
    >
      {notice.message}
    </div>
  );
}
