import { PlannedScreen } from "../../_components/planned-screen";

/** Marks the Audio integration documentation destination. */
export default function AudioDocumentationPage() {
  return (
    <PlannedScreen
      description="The Audio guide will describe audio connection options, supported behavior, authentication, and API examples."
      nextHref="/workspace/docs"
      nextLabel="Back to documentation"
      phase="API documentation"
      title="Audio API"
    />
  );
}
