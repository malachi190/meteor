import { PlannedScreen } from "../_components/planned-screen";

/** Marks the settings destination for supported account and workspace controls. */
export default function SettingsPage() {
  return (
    <PlannedScreen
      description="Settings will include only account and workspace controls backed by Meteor API operations."
      nextHref="/workspace/profile"
      nextLabel="View profile"
      phase="profile and settings screens"
      title="Settings"
    />
  );
}
