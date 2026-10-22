import Link from "next/link";

import { WorkspaceIcon, type WorkspaceIconName } from "@/components/workspace-icon";

const workspaceDestinations = [
  {
    title: "Projects",
    icon: "projects",
    description: "Create and organize the projects that use Meteor.",
    href: "/workspace/projects",
    action: "Open projects",
  },
  {
    title: "API keys",
    icon: "api-keys",
    description: "Manage test and live credentials within a project.",
    href: "/workspace/api-keys",
    action: "Review API keys",
  },
] as const satisfies ReadonlyArray<{
  title: string;
  icon: WorkspaceIconName;
  description: string;
  href: string;
  action: string;
}>;

const guideDestinations = [
  {
    title: "Video",
    icon: "video",
    description: "Connect video experiences and review the integration steps.",
    href: "/workspace/docs/video",
  },
  {
    title: "Audio",
    icon: "audio",
    description: "Explore audio connection guidance and supported options.",
    href: "/workspace/docs/audio",
  },
  {
    title: "Streaming",
    icon: "streaming",
    description: "Read about streaming workflows and their current availability.",
    href: "/workspace/docs/streaming",
  },
] as const satisfies ReadonlyArray<{
  title: string;
  icon: WorkspaceIconName;
  description: string;
  href: string;
}>;

/** Presents the dashboard areas as a practical starting point without inventing usage metrics. */
export default function WorkspaceHomePage() {
  return (
    <div className="mx-auto max-w-5xl" data-ui="workspace-home">
      <header className="max-w-2xl">
        <p className="text-xs font-medium text-[#66645f]">METEOR WORKSPACE</p>
        <h1 className="mt-2 text-2xl font-semibold tracking-[-0.025em] text-[#070607]">Workspace overview</h1>
        <p className="mt-3 text-sm leading-6 text-[#555451]">
          Choose a project task, account area, or integration guide to continue.
        </p>
      </header>

      <section className="mt-9" aria-labelledby="manage-heading">
        <div className="flex items-baseline justify-between gap-4">
          <h2 className="text-base font-semibold text-[#252421]" id="manage-heading">Build and manage</h2>
          <span className="text-xs text-[#66645f]">Workspace tools</span>
        </div>
        <div className="mt-3 grid gap-3 sm:grid-cols-2">
          {workspaceDestinations.map((destination) => (
            <Link
              className="group flex min-h-36 flex-col justify-between border border-[#e5e4df] bg-white/60 p-5 transition-colors hover:border-[#cbc9c2] hover:bg-white focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#fc5000]"
              href={destination.href}
              key={destination.href}
            >
              <div>
                <h3 className="flex items-center gap-2 text-base font-semibold text-[#070607]">
                  <WorkspaceIcon className="h-[17px] w-[17px] text-[#66645f]" name={destination.icon} />
                  {destination.title}
                </h3>
                <p className="mt-2 max-w-sm text-sm leading-6 text-[#555451]">{destination.description}</p>
              </div>
              <span className="mt-4 text-sm font-medium text-[#292826] group-hover:underline group-hover:underline-offset-4">
                {destination.action} <span aria-hidden="true">→</span>
              </span>
            </Link>
          ))}
        </div>
      </section>

      <section className="mt-9" aria-labelledby="guides-heading">
        <div className="flex items-baseline justify-between gap-4">
          <h2 className="text-base font-semibold text-[#252421]" id="guides-heading">Integration guides</h2>
          <Link className="text-sm font-medium text-[#454440] underline-offset-4 hover:underline" href="/workspace/docs">
            Documentation home
          </Link>
        </div>
        <div className="mt-3 divide-y divide-[#e5e4df] border-y border-[#e5e4df]">
          {guideDestinations.map((guide) => (
            <Link
              className="group flex min-h-[76px] items-center justify-between gap-5 py-4 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#fc5000]"
              href={guide.href}
              key={guide.href}
            >
              <span>
                <span className="flex items-center gap-2 text-sm font-semibold text-[#070607]">
                  <WorkspaceIcon className="h-4 w-4 text-[#66645f]" name={guide.icon} />
                  {guide.title}
                </span>
                <span className="mt-1 block text-sm leading-5 text-[#555451]">{guide.description}</span>
              </span>
              <span aria-hidden="true" className="shrink-0 text-[#555451] transition-transform group-hover:translate-x-1">→</span>
            </Link>
          ))}
        </div>
      </section>

      <section className="mt-9" aria-labelledby="account-heading">
        <h2 className="text-base font-semibold text-[#252421]" id="account-heading">Account</h2>
        <div className="mt-3 grid gap-5 border-y border-[#e5e4df] py-4 sm:grid-cols-2 sm:gap-8">
          <Link className="group flex items-center justify-between gap-4 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#fc5000]" href="/workspace/profile">
            <span>
              <span className="block text-sm font-semibold text-[#070607]">Profile</span>
              <span className="mt-1 block text-sm text-[#555451]">View your account identity and workspace role.</span>
            </span>
            <span aria-hidden="true" className="text-[#555451] group-hover:translate-x-1">→</span>
          </Link>
          <Link className="group flex items-center justify-between gap-4 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#fc5000]" href="/workspace/settings">
            <span>
              <span className="block text-sm font-semibold text-[#070607]">Settings</span>
              <span className="mt-1 block text-sm text-[#555451]">Review account and workspace options.</span>
            </span>
            <span aria-hidden="true" className="text-[#555451] group-hover:translate-x-1">→</span>
          </Link>
        </div>
      </section>
    </div>
  );
}
