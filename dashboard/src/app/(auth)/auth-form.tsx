"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import axios from "axios";

import { apiClient } from "@/lib/api-client";
import PasswordInput from "@/components/password-input";

type AuthMode = "login" | "signup" | "recovery";

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

const githubErrorMessages: Record<string, string> = {
  github_cancelled: "GitHub sign-in was cancelled.",
  github_state_invalid: "The GitHub sign-in expired. Start again.",
  github_unavailable: "GitHub sign-in is unavailable. Check the backend OAuth configuration.",
  github_signin_failed: "GitHub sign-in could not be completed. Try again.",
  verified_email_required: "Your GitHub account needs a verified primary email address.",
  workspace_required: "Enter a workspace name to create a new account with GitHub.",
};

/** Renders an email/password auth form and submits it to the matching Meteor API route. */
export default function AuthForm({ mode, githubError }: { mode: AuthMode; githubError?: string }) {
  const router = useRouter();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [notice, setNotice] = useState<FormNotice | null>(() => {
    if (mode !== "login" || !githubError) return null;
    return {
      kind: "error",
      message: githubErrorMessages[githubError] ?? "GitHub sign-in could not be completed. Try again.",
    };
  });
  const [submittedEmail, setSubmittedEmail] = useState("");
  const [workspace, setWorkspace] = useState("");
  const [githubWorkspace, setGithubWorkspace] = useState("");

  const copy = {
    login: {
      eyebrow: "YOUR WORKSPACE",
      title: "Sign in",
      description: "Continue to your Meteor developer workspace.",
      submit: "Sign in",
    },
    signup: {
      eyebrow: "GET STARTED",
      title: "Create your workspace",
      description: "Set up your workspace and verify your email to continue.",
      submit: "Create workspace",
    },
    recovery: {
      eyebrow: "ACCOUNT RECOVERY",
      title: "Reset your password",
      description: "We’ll send a reset link if an eligible account uses this email.",
      submit: "Send reset link",
    },
  }[mode];

  /** Submits credentials or an account recovery request and turns API responses into user feedback. */
  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setIsSubmitting(true);
    setNotice(null);

    const formData = new FormData(event.currentTarget);
    const email = String(formData.get("email") ?? "").trim();
    setSubmittedEmail(email);

    try {
      if (mode === "login") {
        await apiClient.post("/v1/auth/login", {
          email,
          password: String(formData.get("password") ?? ""),
        });
        router.push("/workspace");
      } else if (mode === "signup") {
        await apiClient.post("/v1/auth/signup", {
          tenant_name: String(formData.get("workspace") ?? "").trim(),
          email,
          password: String(formData.get("password") ?? ""),
        });
        setNotice({
          kind: "success",
          message: "If signup was accepted, check your inbox for a verification link before signing in.",
        });
      } else {
        await apiClient.post("/v1/auth/password-reset/request", { email });
        setNotice({
          kind: "success",
          message: "If an eligible account uses that address, a password reset link will arrive by email.",
        });
      }
    } catch (error) {
      const message = axios.isAxiosError<AuthErrorResponse>(error)
        ? error.response?.data?.error?.message
        : undefined;
      setNotice({
        kind: "error",
        message: message ?? "We couldn’t complete that request. Check your connection and try again.",
      });
    } finally {
      setIsSubmitting(false);
    }
  }

  /** Sends the browser through the API's GitHub OAuth start and callback routes. */
  function startGitHubSignIn() {
    const workspaceName = mode === "signup" ? workspace.trim() : githubWorkspace.trim();
    if (mode === "signup" && !workspaceName) {
      setNotice({ kind: "error", message: "Enter a workspace name before creating an account with GitHub." });
      return;
    }

    const authorizationURL = new URL("/v1/auth/github", apiClient.defaults.baseURL);
    if (workspaceName) authorizationURL.searchParams.set("tenant_name", workspaceName);
    window.location.assign(authorizationURL.toString());
  }

  /** Requests another verification email without revealing whether the address has an eligible account. */
  async function resendVerification() {
    if (!submittedEmail) return;

    setIsSubmitting(true);
    setNotice(null);
    try {
      await apiClient.post("/v1/auth/resend-verification", { email: submittedEmail });
      setNotice({
        kind: "success",
        message: "If the account still needs verification, a fresh link will arrive by email.",
      });
    } catch (error) {
      const message = axios.isAxiosError<AuthErrorResponse>(error)
        ? error.response?.data?.error?.message
        : undefined;
      setNotice({
        kind: "error",
        message: message ?? "We couldn’t resend the link. Check your connection and try again.",
      });
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <section aria-labelledby="auth-title" className="w-full max-w-md rounded-[24px] bg-[#f7f6f2] px-6 py-8 sm:px-10 sm:py-10">
      <p className="text-xs font-semibold tracking-[0.12em] text-[#716f69]">{copy.eyebrow}</p>
      <h1 id="auth-title" className="mt-3 text-[30px] font-semibold leading-tight tracking-[-0.03em] text-[#070607]">
        {copy.title}
      </h1>
      <p className="mt-2 text-sm leading-6 text-[#555451]">{copy.description}</p>

      <form className="mt-8 space-y-5" onSubmit={handleSubmit}>
        {mode === "signup" && (
          <label className="block text-sm font-medium text-[#292826]">
            Workspace name
            <input
              autoComplete="organization"
              className="mt-2 h-12 w-full rounded-xl border border-[#c9c8c3] bg-[#f7f6f2] px-4 text-base text-[#070607] outline-none transition focus:border-[#070607] focus:ring-2 focus:ring-[#fc5000]/25"
              maxLength={120}
              name="workspace"
              onChange={(event) => setWorkspace(event.target.value)}
              placeholder="Your team or company"
              required
              value={workspace}
            />
          </label>
        )}

        <label className="block text-sm font-medium text-[#292826]">
          Email address
          <input
            autoComplete="email"
            className="mt-2 h-12 w-full rounded-xl border border-[#c9c8c3] bg-[#f7f6f2] px-4 text-base text-[#070607] outline-none transition focus:border-[#070607] focus:ring-2 focus:ring-[#fc5000]/25"
            name="email"
            placeholder="you@example.com"
            required
            type="email"
          />
        </label>

        {mode !== "recovery" && (
          <label className="block text-sm font-medium text-[#292826]">
            Password
            <PasswordInput
              autoComplete={mode === "login" ? "current-password" : "new-password"}
              className="h-12 w-full rounded-xl border border-[#c9c8c3] bg-[#f7f6f2] px-4 text-base text-[#070607] outline-none transition focus:border-[#070607] focus:ring-2 focus:ring-[#fc5000]/25"
              maxLength={passwordLimits.maxLength}
              minLength={mode === "signup" ? passwordLimits.minLength : undefined}
              name="password"
              required
              type="password"
            />
            {mode === "signup" && (
              <span className="mt-1.5 block text-xs font-normal text-[#716f69]">
                Use between 12 and 128 characters.
              </span>
            )}
          </label>
        )}

        {mode === "login" && (
          <label className="block text-sm font-medium text-[#292826]">
            Workspace name for a new GitHub account <span className="font-normal text-[#716f69]">(optional)</span>
            <input
              autoComplete="organization"
              className="mt-2 h-12 w-full rounded-xl border border-[#c9c8c3] bg-[#f7f6f2] px-4 text-base text-[#070607] outline-none transition focus:border-[#070607] focus:ring-2 focus:ring-[#fc5000]/25"
              maxLength={120}
              onChange={(event) => setGithubWorkspace(event.target.value)}
              placeholder="Only needed the first time"
              value={githubWorkspace}
            />
          </label>
        )}

        {notice && (
          <div
            aria-live="polite"
            className={`rounded-xl border px-4 py-3 text-sm leading-5 ${notice.kind === "error" ? "border-[#b94337]/40 bg-[#b94337]/5 text-[#812b24]" : "border-[#858276]/35 bg-[#e2e2df]/55 text-[#292826]"}`}
            role={notice.kind === "error" ? "alert" : "status"}
          >
            {notice.message}
          </div>
        )}

        <button
          className="flex h-12 w-full items-center justify-center rounded-full bg-[#fc5000] px-6 text-sm font-semibold text-[#070607] transition-colors hover:bg-[#e94a00] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#070607] disabled:cursor-wait disabled:opacity-60"
          disabled={isSubmitting}
          type="submit"
        >
          {isSubmitting ? "Please wait…" : copy.submit}
        </button>
      </form>

      {mode !== "recovery" && (
        <>
          <div className="my-6 flex items-center gap-3 text-xs text-[#716f69]">
            <span aria-hidden="true" className="h-px flex-1 bg-[#d7d5ce]" />
            <span>OR CONTINUE WITH</span>
            <span aria-hidden="true" className="h-px flex-1 bg-[#d7d5ce]" />
          </div>
          <button
            className="flex h-12 w-full items-center justify-center gap-2.5 rounded-full border border-[#aaa9a3] bg-transparent px-5 text-sm font-medium text-[#070607] transition-colors hover:bg-[#e2e2df]/60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#070607]"
            onClick={startGitHubSignIn}
            type="button"
          >
            <GitHubIcon />
            Continue with GitHub
          </button>
        </>
      )}

      {mode === "signup" && notice?.kind === "success" && submittedEmail && (
        <button
          className="mt-4 text-sm font-medium text-[#070607] underline decoration-[#fc5000] underline-offset-4 disabled:opacity-60"
          disabled={isSubmitting}
          onClick={resendVerification}
          type="button"
        >
          Resend verification email
        </button>
      )}

      <div className="mt-7 border-t border-[#d7d5ce] pt-5 text-sm text-[#555451]">
        {mode === "login" ? (
          <>
            <p>
              <Link className="font-medium text-[#070607] underline decoration-[#fc5000] underline-offset-4" href="/forgot-password">
                Forgot password?
              </Link>
            </p>
            <p className="mt-3">
              New to Meteor?{" "}
              <Link className="font-medium text-[#070607] underline decoration-[#fc5000] underline-offset-4" href="/signup">
                Create a workspace
              </Link>
            </p>
          </>
        ) : mode === "signup" ? (
          <p>
            Already have an account?{" "}
            <Link className="font-medium text-[#070607] underline decoration-[#fc5000] underline-offset-4" href="/login">
              Sign in
            </Link>
          </p>
        ) : (
          <p>
            Remembered your password?{" "}
            <Link className="font-medium text-[#070607] underline decoration-[#fc5000] underline-offset-4" href="/login">
              Return to sign in
            </Link>
          </p>
        )}
      </div>
    </section>
  );
}

/** Draws the GitHub mark used beside the OAuth sign-in action. */
function GitHubIcon() {
  return (
    <svg aria-hidden="true" className="h-[18px] w-[18px]" fill="currentColor" viewBox="0 0 24 24">
      <path d="M12 .75a11.25 11.25 0 0 0-3.56 21.92c.56.1.77-.24.77-.54v-2.1c-3.14.68-3.8-1.33-3.8-1.33-.51-1.31-1.25-1.66-1.25-1.66-1.02-.7.08-.69.08-.69 1.13.08 1.73 1.16 1.73 1.16 1 .1.7 2.67 3.9 1.9.1-.72.4-1.21.72-1.49-2.5-.28-5.13-1.25-5.13-5.57 0-1.23.44-2.24 1.16-3.03-.12-.29-.5-1.44.11-3 0 0 .95-.31 3.1 1.16a10.76 10.76 0 0 1 5.64 0c2.15-1.47 3.1-1.16 3.1-1.16.61 1.56.23 2.71.11 3 .73.79 1.16 1.8 1.16 3.03 0 4.33-2.63 5.29-5.14 5.56.4.35.76 1.03.76 2.08v3.14c0 .3.2.65.77.54A11.25 11.25 0 0 0 12 .75Z" />
    </svg>
  );
}
