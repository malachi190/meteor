import Link from "next/link";

import { WorkspaceIcon, type WorkspaceIconName } from "@/components/workspace-icon";

const documentationAreas = [
  {
    title: "Video",
    icon: "video",
    description: "Video connection guidance and API reference.",
    href: "/workspace/docs/video",
  },
  {
    title: "Audio",
    icon: "audio",
    description: "Audio connection guidance and API reference.",
    href: "/workspace/docs/audio",
  },
  {
    title: "Streaming",
    icon: "streaming",
    description: "Streaming workflow guidance and API reference.",
    href: "/workspace/docs/streaming",
  },
] as const satisfies ReadonlyArray<{
  title: string;
  icon: WorkspaceIconName;
  description: string;
  href: string;
}>;

/** Provides the documentation index and routes to its Video, Audio, and Streaming sections. */
export default function DocumentationHomePage() {
  return (
    <div className="mx-auto max-w-5xl">
      <header className="max-w-2xl">
        <p className="text-xs font-medium text-[#66645f]">METEOR API</p>
        <h1 className="mt-2 text-2xl font-semibold tracking-[-0.025em] text-[#070607]">Documentation</h1>
        <p className="mt-3 text-sm leading-6 text-[#555451]">
          Choose an API area for its integration guide, endpoint reference, and practical examples.
        </p>
      </header>
      <div className="mt-8 divide-y divide-[#e5e4df] border-y border-[#e5e4df]">
        {documentationAreas.map((area) => (
          <Link
            className="group flex min-h-[78px] items-center justify-between gap-5 py-4 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#fc5000]"
            href={area.href}
            key={area.href}
          >
            <span>
              <span className="flex items-center gap-2 text-sm font-semibold text-[#070607]">
                <WorkspaceIcon className="h-4 w-4 text-[#66645f]" name={area.icon} />
                {area.title}
              </span>
              <span className="mt-1 block text-sm text-[#555451]">{area.description}</span>
            </span>
            <span aria-hidden="true" className="shrink-0 text-[#555451] group-hover:translate-x-1">→</span>
          </Link>
        ))}
      </div>
    </div>
  );
}
