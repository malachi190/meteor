import { PlannedScreen } from "../../../_components/planned-screen";

/** Marks the project-scoped API key list and revocation controls. */
export default async function ProjectKeysPage({
  params,
}: {
  params: Promise<{ projectId: string }>;
}) {
  const { projectId } = await params;

  return (
    <PlannedScreen
      context={"Project ID: " + projectId}
      description="Review a project's key metadata and revoke credentials that are no longer needed."
      nextHref={"/workspace/projects/" + encodeURIComponent(projectId) + "/keys/new"}
      nextLabel="Create an API key"
      phase="project and API key screens"
      title="Project API keys"
    />
  );
}
