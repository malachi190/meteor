import { PlannedScreen } from "../../_components/planned-screen";

/** Marks the Streaming integration documentation destination. */
export default function StreamingDocumentationPage() {
  return (
    <PlannedScreen
      description="The Streaming guide will cover only confirmed Meteor streaming capabilities, their connection steps, and API examples."
      nextHref="/workspace/docs"
      nextLabel="Back to documentation"
      phase="API documentation"
      title="Streaming API"
    />
  );
}
