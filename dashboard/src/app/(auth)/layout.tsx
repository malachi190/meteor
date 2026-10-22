import Link from "next/link";
import type { ReactNode } from "react";

/** Provides a shared, quiet frame for the dashboard's public authentication screens. */
export default function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <main className="flex min-h-screen flex-col bg-[#e2e2df] px-5 py-8 sm:px-8 sm:py-12">
      <header className="mx-auto flex w-full max-w-5xl items-center justify-between">
        <Link aria-label="Meteor sign in" className="inline-flex items-center gap-2" href="/login">
          <span
            aria-hidden="true"
            className="h-9 w-9 shrink-0 mix-blend-multiply [background-image:url('/meteor-logo.png')] [background-position:center] [background-repeat:no-repeat] [background-size:270%_270%]"
          />
          <span className="text-[17px] font-semibold tracking-[-0.02em] text-[#070607]">meteor</span>
        </Link>
        <span className="text-sm text-[#555451]">Developer dashboard</span>
      </header>
      <div className="flex flex-1 items-center justify-center py-12">{children}</div>
      <footer className="mx-auto w-full max-w-5xl text-xs text-[#555451]">
        Meteor account access
      </footer>
    </main>
  );
}
