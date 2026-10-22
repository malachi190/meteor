import type { ReactNode } from "react";

import { WorkspaceShell } from "./workspace-shell";

/** Keeps workspace navigation and session controls consistent across signed-in screens. */
export default function WorkspaceLayout({ children }: { children: ReactNode }) {
  return <WorkspaceShell>{children}</WorkspaceShell>;
}
