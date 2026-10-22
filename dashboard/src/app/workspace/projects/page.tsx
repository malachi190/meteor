import Link from "next/link";

import { PlannedScreen } from "../_components/planned-screen";

/** Provides the projects entrypoint and the planned project creation path. */
export default function ProjectsPage() {
  return (
    <>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-4">
        <span className="text-xs text-[#66645f]">Project management</span>
        <Link
          className="inline-flex min-h-10 items-center rounded-full bg-[#fc5000] px-4 text-sm font-semibold text-[#070607] hover:bg-[#e94900] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#070607]"
          href="/workspace/projects/new"
        >
          Create project
        </Link>
      </div>
      <PlannedScreen
        description="Projects will organize API credentials and configuration for each application using Meteor."
        nextHref="/workspace"
        nextLabel="Back to workspace overview"
        phase="project and API key screens"
        title="Projects"
      />
    </>
  );
}
