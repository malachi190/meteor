import { PlannedScreen } from "../../../../_components/planned-screen";

/** Marks the one-time API key creation and secret reveal flow for a project. */
export default async function NewProjectKeyPage({
  params,
}: {
  params: Promise<{ projectId: string }>;
}) {
  const { projectId } = await params;

  return (
    <PlannedScreen
      context={"Project ID: " + projectId}
      description="Choose a key environment, create the credential, and reveal its secret once with a copy action."
      nextHref={"/workspace/projects/" + encodeURIComponent(projectId) + "/keys"}
      nextLabel="Back to project API keys"
      phase="project and API key screens"
      title="Create an API key"
    />
  );
}
