import { AppearanceSettings } from "./appearance-settings"
import { DataDirForm } from "./data-dir-form"
import { DesktopBehavior } from "./desktop-behavior"
import { RequestRetentionSettings } from "./request-retention"
import { StorageOverview } from "./storage-overview"
import type { AppPreferencesTab } from "./app-preferences-state"

export function SettingsContent({ section }: { section: Exclude<AppPreferencesTab, "about"> }) {
  return (
    <section className="min-w-0 space-y-4 pb-1">
      {section === "settings" ? (
        <>
          <DesktopBehavior />
          <AppearanceSettings />
        </>
      ) : null}
      {section === "activity" ? <RequestRetentionSettings /> : null}
      {section === "storage" ? (
        <>
          <DataDirForm />
          <StorageOverview />
        </>
      ) : null}
    </section>
  )
}
