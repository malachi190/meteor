export type WorkspaceIconName =
  | "home"
  | "projects"
  | "api-keys"
  | "docs"
  | "video"
  | "audio"
  | "streaming"
  | "profile"
  | "settings"
  | "menu"
  | "close";

type WorkspaceIconProps = {
  name: WorkspaceIconName;
  className?: string;
};

/** Draws a small, neutral outline icon for a workspace destination. */
export function WorkspaceIcon({ name, className = "h-4 w-4" }: WorkspaceIconProps) {
  const shared = {
    "aria-hidden": true as const,
    className,
    fill: "none",
    stroke: "currentColor",
    strokeLinecap: "round" as const,
    strokeLinejoin: "round" as const,
    strokeWidth: 1.7,
    viewBox: "0 0 24 24",
  };

  switch (name) {
    case "home":
      return <svg {...shared}><path d="m3.5 10 8.5-7 8.5 7" /><path d="M5.5 9v11h13V9M9.5 20v-6h5v6" /></svg>;
    case "projects":
      return <svg {...shared}><path d="M3.5 6.5h6l2 2h9v10a1 1 0 0 1-1 1h-15a1 1 0 0 1-1-1z" /><path d="M3.5 9h17" /></svg>;
    case "api-keys":
      return <svg {...shared}><circle cx="8" cy="15" r="4.5" /><path d="m11.2 11.8 8.3-8.3M16 7l2 2M13.5 9.5l2 2" /></svg>;
    case "docs":
      return <svg {...shared}><path d="M6 3.5h8l4 4V20H6z" /><path d="M14 3.5V8h4M9 12h6M9 15.5h6" /></svg>;
    case "video":
      return <svg {...shared}><rect x="3.5" y="5.5" width="12" height="13" rx="1.5" /><path d="m15.5 10 5-3v10l-5-3z" /></svg>;
    case "audio":
      return <svg {...shared}><path d="M4 10v4M8 7v10M12 4v16M16 7v10M20 10v4" /></svg>;
    case "streaming":
      return <svg {...shared}><circle cx="12" cy="18" r="1" /><path d="M8.5 14.5a5 5 0 0 1 7 0M5.5 11.5a9 9 0 0 1 13 0M2.5 8.5a13 13 0 0 1 19 0M12 19v2" /></svg>;
    case "profile":
      return <svg {...shared}><circle cx="12" cy="8" r="3.5" /><path d="M4.5 20a7.5 7.5 0 0 1 15 0" /></svg>;
    case "settings":
      return <svg {...shared}><circle cx="12" cy="12" r="3" /><path d="m19.4 15 .1.1 1.4 1.1-1.5 2.6-1.7-.7a7.8 7.8 0 0 1-1.7 1l-.3 1.8h-3l-.3-1.8a7.8 7.8 0 0 1-1.7-1l-1.7.7-1.5-2.6 1.4-1.1a7 7 0 0 1 0-2l-1.4-1.1 1.5-2.6 1.7.7a7.8 7.8 0 0 1 1.7-1l.3-1.8h3l.3 1.8a7.8 7.8 0 0 1 1.7 1l1.7-.7 1.5 2.6-1.4 1.1a7 7 0 0 1 0 1.9Z" /></svg>;
    case "menu":
      return <svg {...shared}><path d="M4 6.5h16M4 12h16M4 17.5h16" /></svg>;
    case "close":
      return <svg {...shared}><path d="m6 6 12 12M18 6 6 18" /></svg>;
  }
}
