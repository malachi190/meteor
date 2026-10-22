import { PlannedScreen } from "../_components/planned-screen";

/** Marks the profile destination for account and workspace identity. */
export default function ProfilePage() {
  return (
    <PlannedScreen
      description="Your profile will show the identity and workspace details returned by your authenticated account."
      nextHref="/workspace/settings"
      nextLabel="Open settings"
      phase="profile and settings screens"
      title="Profile"
    />
  );
}
