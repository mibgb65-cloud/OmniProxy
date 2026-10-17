import { AppearanceSettings } from "./appearance-settings"
import { DataDirForm } from "./data-dir-form"
import { DesktopBehavior } from "./desktop-behavior"
import { RequestRetentionSettings } from "./request-retention"
import { StorageOverview } from "./storage-overview"

export function SettingsContent() {
  return (
    <section className="min-w-0 space-y-6 pb-1">
      <AppearanceSettings />
      <RequestRetentionSettings />
      <StorageOverview />
      <DataDirForm />
      <DesktopBehavior />
    </section>
  )
}
