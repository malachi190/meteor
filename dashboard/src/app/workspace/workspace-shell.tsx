"use client";

import { useEffect, useState, type ReactNode } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import axios from "axios";

import { apiClient } from "@/lib/api-client";
import { WorkspaceIcon, type WorkspaceIconName } from "@/components/workspace-icon";

type Account = {
  id: string;
  tenant_id: string;
  email: string;
  role: string;
};

type AccountResponse = {
  user: Account;
};

const navigationGroups = [
  {
    label: "Workspace",
    links: [
      { label: "Home", href: "/workspace", icon: "home" },
      { label: "Projects", href: "/workspace/projects", icon: "projects" },
      { label: "API keys", href: "/workspace/api-keys", icon: "api-keys" },
    ],
  },
  {
    label: "Integration guides",
    links: [
      { label: "Documentation", href: "/workspace/docs", icon: "docs" },
      { label: "Video", href: "/workspace/docs/video", icon: "video" },
      { label: "Audio", href: "/workspace/docs/audio", icon: "audio" },
      { label: "Streaming", href: "/workspace/docs/streaming", icon: "streaming" },
    ],
  },
  {
    label: "Account",
    links: [
      { label: "Profile", href: "/workspace/profile", icon: "profile" },
      { label: "Settings", href: "/workspace/settings", icon: "settings" },
    ],
  },
] as const satisfies ReadonlyArray<{
  label: string;
  links: ReadonlyArray<{ label: string; href: string; icon: WorkspaceIconName }>;
}>;

/** Loads the authenticated account and provides the responsive Meteor workspace frame. */
export function WorkspaceShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const [account, setAccount] = useState<Account | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSigningOut, setIsSigningOut] = useState(false);
  const [error, setError] = useState("");
  const [retryCount, setRetryCount] = useState(0);
  const [isMenuOpen, setIsMenuOpen] = useState(false);

  useEffect(() => {
    let isCurrent = true;

    /** Resolves the account using the server-managed session cookie. */
    async function loadAccount() {
      setIsLoading(true);
      setError("");
      try {
        const response = await apiClient.get<AccountResponse>("/v1/auth/me");
        if (isCurrent) setAccount(response.data.user);
      } catch {
        if (isCurrent) {
          setAccount(null);
          setError("We couldn’t load your Meteor account. Check your connection and try again.");
        }
      } finally {
        if (isCurrent) setIsLoading(false);
      }
    }

    void loadAccount();
    return () => {
      isCurrent = false;
    };
  }, [retryCount]);

  /** Ends the server session and returns the browser to sign-in. */
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

  /** Renders the grouped dashboard destinations with a clear active route. */
  function renderNavigation() {
    return navigationGroups.map((group) => (
      <section className="mt-7 first:mt-0" key={group.label}>
        <h2 className="px-3 text-xs font-medium text-[#66645f]">{group.label}</h2>
        <nav aria-label={group.label} className="mt-2 grid gap-1">
          {group.links.map((item) => {
            const isActive = item.href === "/workspace"
              ? pathname === item.href
              : pathname === item.href || pathname.startsWith(item.href + "/");

            return (
              <Link
                aria-current={isActive ? "page" : undefined}
                className={"flex min-h-10 items-center rounded-md px-3 text-sm transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#fc5000] " + (
                  isActive
                    ? "bg-[#e8e7e2] font-semibold text-[#070607]"
                    : "text-[#454440] hover:bg-[#eeede8] hover:text-[#070607]"
                )}
                href={item.href}
                key={item.href}
                onClick={() => setIsMenuOpen(false)}
              >
                <WorkspaceIcon className="mr-3 h-[17px] w-[17px] shrink-0 text-[#66645f]" name={item.icon} />
                {item.label}
              </Link>
            );
          })}
        </nav>
      </section>
    ));
  }

  if (isLoading) {
    return (
      <main aria-live="polite" className="flex min-h-svh items-center justify-center px-5">
        <p className="text-sm text-[#555451]">Checking your Meteor session…</p>
      </main>
    );
  }

  if (!account) {
    return (
      <main className="mx-auto flex min-h-svh w-full max-w-lg flex-col justify-center px-5 py-12">
        <p className="text-xs font-semibold tracking-[0.1em] text-[#716f69]">METEOR WORKSPACE</p>
        <h1 className="mt-3 text-2xl font-semibold tracking-[-0.02em]">Sign in to continue</h1>
        <p className="mt-2 text-sm leading-6 text-[#555451]" role="alert">
          {error || "Your session has ended. Sign in to return to your workspace."}
        </p>
        {error && (
          <button
            className="mt-5 min-h-11 w-fit rounded-full border border-[#a5a39c] px-5 text-sm font-medium hover:bg-[#f7f6f2]"
            onClick={() => setRetryCount((count) => count + 1)}
            type="button"
          >
            Try again
          </button>
        )}
        <Link
          className="mt-5 inline-flex min-h-11 w-fit items-center rounded-full bg-[#fc5000] px-5 text-sm font-semibold text-[#070607] hover:bg-[#e94900]"
          href="/login"
        >
          Return to sign in
        </Link>
      </main>
    );
  }

  return (
    <div className="min-h-svh bg-[var(--meteor-frame)] lg:p-2" data-ui="workspace-shell">
      <div className="min-h-svh bg-[var(--meteor-chrome)] lg:grid lg:min-h-[calc(100vh-16px)] lg:grid-cols-[232px_minmax(0,1fr)] lg:overflow-hidden lg:rounded-[16px] lg:border lg:border-[#e4e3de]">
        <aside className="hidden border-r border-[#e7e6e1] px-4 py-5 lg:flex lg:flex-col">
          <Link aria-label="Meteor dashboard home" className="flex w-fit items-center gap-2 px-2" href="/workspace">
            <span
              aria-hidden="true"
              className="h-8 w-8 shrink-0 mix-blend-multiply [background-image:url('/meteor-logo.png')] [background-position:center] [background-repeat:no-repeat] [background-size:270%_270%]"
            />
            <span className="text-[17px] font-semibold tracking-[-0.02em] text-[#070607]">meteor</span>
          </Link>
          <div className="mt-7 border-y border-[#e5e4df] px-3 py-3">
            <p className="text-xs text-[#66645f]">Current scope</p>
            <p className="mt-1 text-sm font-medium text-[#252421]">Workspace</p>
          </div>
          <div className="mt-6 flex-1">{renderNavigation()}</div>
          <div className="mt-8 border-t border-[#e5e4df] px-3 pt-4">
            <p className="break-all text-sm font-medium text-[#252421]">{account.email}</p>
            <p className="mt-1 text-xs capitalize text-[#66645f]">{account.role}</p>
          </div>
        </aside>

        <div className="relative min-w-0">
          <header className="relative flex min-h-[58px] items-center justify-between gap-4 border-b border-[#e7e6e1] px-4 sm:px-6 lg:min-h-12 lg:border-0 lg:px-7">
            <div className="flex min-w-0 items-center gap-3 lg:hidden">
              <button
                aria-expanded={isMenuOpen}
                aria-label={isMenuOpen ? "Close workspace navigation" : "Open workspace navigation"}
                className="inline-flex h-10 w-10 items-center justify-center rounded-md border border-[#d8d7d1] text-lg leading-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#fc5000]"
                onClick={() => setIsMenuOpen((open) => !open)}
                type="button"
              >
                <WorkspaceIcon className="h-5 w-5" name={isMenuOpen ? "close" : "menu"} />
              </button>
              <Link aria-label="Meteor dashboard home" className="flex items-center gap-1.5" href="/workspace">
                <span
                  aria-hidden="true"
                  className="h-7 w-7 shrink-0 mix-blend-multiply [background-image:url('/meteor-logo.png')] [background-position:center] [background-repeat:no-repeat] [background-size:270%_270%]"
                />
                <span className="text-base font-semibold tracking-[-0.02em]">meteor</span>
              </Link>
            </div>
            <p className="hidden truncate text-sm text-[#5e5c56] lg:block">Workspace</p>
            <div className="flex min-w-0 items-center gap-3">
              <span className="hidden max-w-64 truncate text-sm text-[#5e5c56] sm:inline">{account.email}</span>
              <button
                className="min-h-10 shrink-0 rounded-full border border-[#b9b8b1] px-4 text-sm font-medium text-[#292826] transition-colors hover:bg-[#eeede8] disabled:opacity-60"
                disabled={isSigningOut}
                onClick={signOut}
                type="button"
              >
                {isSigningOut ? "Signing out…" : "Sign out"}
              </button>
            </div>
          </header>

          {isMenuOpen && (
            <div className="absolute inset-x-0 top-[58px] z-10 border-b border-[#d8d7d1] bg-[#f7f6f2] px-4 py-5 shadow-[0_8px_18px_rgba(7,6,7,0.08)] lg:hidden">
              <div className="mb-5 border-b border-[#e5e4df] px-3 pb-4">
                <p className="break-all text-sm font-medium">{account.email}</p>
                <p className="mt-1 text-xs capitalize text-[#66645f]">{account.role}</p>
              </div>
              {renderNavigation()}
            </div>
          )}

          <main className="min-h-[calc(100vh-58px)] bg-[var(--meteor-sheet)] px-5 py-8 sm:px-7 lg:mx-2 lg:mb-2 lg:min-h-[calc(100vh-80px)] lg:rounded-[14px] lg:border lg:border-[#e6e5e0] lg:px-8 lg:py-8 xl:px-10" data-ui="workspace-content">
            {error && (
              <div className="mb-6 flex flex-wrap items-center justify-between gap-3 border border-[#ead0cb] bg-[#fff8f6] px-4 py-3 text-sm text-[#702b24]" role="alert">
                <span>{error}</span>
                <button
                  className="min-h-9 font-medium underline underline-offset-2"
                  onClick={() => setRetryCount((count) => count + 1)}
                  type="button"
                >
                  Retry
                </button>
              </div>
            )}
            {children}
          </main>
        </div>
      </div>
    </div>
  );
}
