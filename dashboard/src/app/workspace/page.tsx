"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import axios from "axios";

import { apiClient } from "@/lib/api-client";

type Account = {
  id: string;
  tenant_id: string;
  email: string;
  role: string;
};

type AccountResponse = {
  user: Account;
};

/** Shows the authenticated workspace entry while the full dashboard shell is being built. */
export default function WorkspacePage() {
  const router = useRouter();
  const [account, setAccount] = useState<Account | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSigningOut, setIsSigningOut] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    let isCurrent = true;

    /** Resolves the account from the server-managed session cookie. */
    async function loadAccount() {
      try {
        const response = await apiClient.get<AccountResponse>("/v1/auth/me");
        if (isCurrent) setAccount(response.data.user);
      } catch {
        if (isCurrent) setError("Your session could not be loaded. Sign in again to continue.");
      } finally {
        if (isCurrent) setIsLoading(false);
      }
    }

    void loadAccount();
    return () => {
      isCurrent = false;
    };
  }, []);

  /** Ends the server session and returns the browser to the sign-in screen. */
  async function signOut() {
    setIsSigningOut(true);
    setError("");
    try {
      await apiClient.post("/v1/auth/logout");
      router.replace("/login");
    } catch (requestError) {
      const message = axios.isAxiosError(requestError) && requestError.response?.status === 401
        ? "Your session has expired. Sign in again to continue."
        : "We couldn’t sign you out. Check your connection and try again.";
      setError(message);
      setIsSigningOut(false);
    }
  }

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-5xl flex-col px-5 py-8 sm:px-8 sm:py-12">
      <header className="flex items-center justify-between border-b border-[#c9c8c3] pb-5">
        <Link aria-label="Meteor dashboard" className="inline-flex items-center gap-2" href="/workspace">
          <span
            aria-hidden="true"
            className="h-9 w-9 shrink-0 mix-blend-multiply [background-image:url('/meteor-logo.png')] [background-position:center] [background-repeat:no-repeat] [background-size:270%_270%]"
          />
          <span className="text-[17px] font-semibold tracking-[-0.02em] text-[#070607]">meteor</span>
        </Link>
        {!isLoading && account && (
          <button
            className="rounded-full border border-[#aaa9a3] px-4 py-2 text-sm font-medium text-[#292826] transition-colors hover:bg-[#f7f6f2]/70 disabled:opacity-60"
            disabled={isSigningOut}
            onClick={signOut}
            type="button"
          >
            {isSigningOut ? "Signing out…" : "Sign out"}
          </button>
        )}
      </header>

      <section aria-live="polite" className="flex flex-1 items-center py-12">
        {isLoading ? (
          <p className="text-sm text-[#555451]">Checking your Meteor session…</p>
        ) : account ? (
          <div className="max-w-xl">
            <p className="text-xs font-semibold tracking-[0.12em] text-[#716f69]">WORKSPACE ACCESS</p>
            <h1 className="mt-3 text-3xl font-semibold tracking-[-0.03em] text-[#070607]">You’re signed in.</h1>
            <p className="mt-3 text-base leading-7 text-[#555451]">
              Your Meteor session is active. The project dashboard is the next screen to build.
            </p>
            <dl className="mt-8 grid gap-4 border-t border-[#c9c8c3] pt-5 text-sm sm:grid-cols-[140px_1fr]">
              <dt className="font-medium text-[#716f69]">Account</dt>
              <dd className="break-all text-[#070607]">{account.email}</dd>
              <dt className="font-medium text-[#716f69]">Workspace ID</dt>
              <dd className="break-all font-mono text-xs text-[#292826]">{account.tenant_id}</dd>
              <dt className="font-medium text-[#716f69]">Role</dt>
              <dd className="text-[#070607]">{account.role}</dd>
            </dl>
          </div>
        ) : (
          <div className="max-w-lg">
            <h1 className="text-2xl font-semibold tracking-[-0.02em] text-[#070607]">Sign in to continue</h1>
            <p className="mt-2 text-sm leading-6 text-[#555451]">{error}</p>
            <Link className="mt-5 inline-flex rounded-full bg-[#fc5000] px-5 py-3 text-sm font-semibold text-[#070607]" href="/login">
              Return to sign in
            </Link>
          </div>
        )}
        {error && account && (
          <p className="ml-6 text-sm text-[#812b24]" role="alert">{error}</p>
        )}
      </section>
    </main>
  );
}
