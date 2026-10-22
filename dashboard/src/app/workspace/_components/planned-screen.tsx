import Link from "next/link";

type PlannedScreenProps = {
  title: string;
  description: string;
  phase: string;
  nextHref: string;
  nextLabel: string;
  context?: string;
};

/** Gives a planned dashboard route a useful explanation and a clear next destination. */
export function PlannedScreen({
  title,
  description,
  phase,
  nextHref,
  nextLabel,
  context,
}: PlannedScreenProps) {
  return (
    <div className="mx-auto max-w-5xl" data-ui="planned-screen">
      <header className="max-w-2xl">
        <p className="text-xs font-medium text-[#66645f]">METEOR WORKSPACE</p>
        <h1 className="mt-2 text-2xl font-semibold tracking-[-0.025em] text-[#070607]">{title}</h1>
        <p className="mt-3 text-sm leading-6 text-[#555451]">{description}</p>
        {context && <p className="mt-2 break-all font-mono text-xs text-[#66645f]">{context}</p>}
      </header>

      <section className="mt-8 max-w-3xl border-y border-[#e4e3de] py-5" aria-labelledby="screen-status">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 className="text-base font-semibold text-[#252421]" id="screen-status">Screen status</h2>
          <span className="rounded-full bg-[#eeede8] px-3 py-1 text-xs font-medium text-[#454440]">Planned</span>
        </div>
        <p className="mt-2 text-sm leading-6 text-[#555451]">
          This route is ready for the {phase}. Its data and controls will be connected when that screen is built.
        </p>
        <Link
          className="mt-4 inline-flex min-h-10 items-center rounded-full border border-[#b9b8b1] px-4 text-sm font-medium text-[#292826] transition-colors hover:bg-[#f0efeb] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#fc5000]"
          href={nextHref}
        >
          {nextLabel}
        </Link>
      </section>
    </div>
  );
}
