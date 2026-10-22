"use client";

import { useState, type InputHTMLAttributes } from "react";

/** Renders a password input with an accessible control for showing or hiding its value. */
export default function PasswordInput(props: InputHTMLAttributes<HTMLInputElement>) {
  const [isVisible, setIsVisible] = useState(false);
  const accessibleName = props["aria-label"] ?? props.name?.replaceAll(/[_-]/g, " ") ?? "password";
  const visibilityAction = isVisible ? "Hide" : "Show";

  return (
    <span className="relative mt-2 block">
      <input
        {...props}
        className={props.className ? props.className + " pr-12" : "pr-12"}
        type={isVisible ? "text" : "password"}
      />
      <button
        aria-label={visibilityAction + " " + accessibleName}
        aria-pressed={isVisible}
        className="absolute inset-y-0 right-0 inline-flex w-12 items-center justify-center rounded-r-xl text-[#555451] transition-colors hover:text-[#070607] focus-visible:outline-2 focus-visible:outline-offset-[-3px] focus-visible:outline-[#070607]"
        onClick={() => setIsVisible((visible) => !visible)}
        type="button"
      >
        {isVisible ? <EyeOffIcon /> : <EyeIcon />}
      </button>
    </span>
  );
}

/** Draws the eye mark used when a password can be revealed. */
function EyeIcon() {
  return (
    <svg aria-hidden="true" className="h-5 w-5" fill="none" viewBox="0 0 24 24">
      <path d="M2.5 12s3.3-6 9.5-6 9.5 6 9.5 6-3.3 6-9.5 6-9.5-6-9.5-6Z" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.7" />
      <circle cx="12" cy="12" r="2.7" stroke="currentColor" strokeWidth="1.7" />
    </svg>
  );
}

/** Draws the crossed-eye mark used when a password is currently visible. */
function EyeOffIcon() {
  return (
    <svg aria-hidden="true" className="h-5 w-5" fill="none" viewBox="0 0 24 24">
      <path d="m3 3 18 18M10.6 6.2A10.8 10.8 0 0 1 12 6c6.2 0 9.5 6 9.5 6a15.4 15.4 0 0 1-3 3.7M6.2 6.2C3.8 7.7 2.5 12 2.5 12s3.3 6 9.5 6c1.1 0 2.1-.2 3-.5M9.9 9.9a3 3 0 0 0 4.2 4.2" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.7" />
    </svg>
  );
}
