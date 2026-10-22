import { PlannedScreen } from "../../_components/planned-screen";

/** Shows the planned detail destination for a project selected from its list. */
export default async function ProjectDetailPage({
  params,
}: {
  params: Promise<{ projectId: string }>;
}) {
  const { projectId } = await params;

  return (
    <PlannedScreen
      context={"Project ID: " + projectId}
      description="A project's overview will bring its configuration and API key controls together."
      nextHref={"/workspace/projects/" + encodeURIComponent(projectId) + "/keys"}
      nextLabel="View project API keys"
      phase="project and API key screens"
      title="Project details"
    />
  );
}
