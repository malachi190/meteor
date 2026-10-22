import { PlannedScreen } from "../../_components/planned-screen";

/** Marks the destination for the upcoming project creation form. */
export default function NewProjectPage() {
  return (
    <PlannedScreen
      description="Create a project to keep its test and live API credentials together."
      nextHref="/workspace/projects"
      nextLabel="Back to projects"
      phase="project and API key screens"
      title="Create a project"
    />
  );
}
