import { PlannedScreen } from "../../_components/planned-screen";

/** Marks the Video integration documentation destination. */
export default function VideoDocumentationPage() {
  return (
    <PlannedScreen
      description="The Video guide will document the supported connection flow, authentication, API reference, and working examples."
      nextHref="/workspace/docs"
      nextLabel="Back to documentation"
      phase="API documentation"
      title="Video API"
    />
  );
}
