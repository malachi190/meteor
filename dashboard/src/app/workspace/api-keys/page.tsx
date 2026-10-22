import { PlannedScreen } from "../_components/planned-screen";

/** Explains that key management is scoped to each project and links to that entrypoint. */
export default function ApiKeysPage() {
  return (
    <PlannedScreen
      description="API keys belong to a project, where test and live credentials can be created, reviewed, and revoked."
      nextHref="/workspace/projects"
      nextLabel="Open projects"
      phase="project and API key screens"
      title="API keys"
    />
  );
}
