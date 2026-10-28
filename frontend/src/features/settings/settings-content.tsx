import { AppearanceSettings } from "./appearance-settings"
import { DataDirForm } from "./data-dir-form"
import { DesktopBehavior } from "./desktop-behavior"
import { ProxyPortSetting } from "./proxy-port-setting"
import { RequestRetentionSettings } from "./request-retention"
import { StorageOverview } from "./storage-overview"
import { LegacyImportCard } from "./legacy-import-card"
import type { AppPreferencesTab } from "./app-preferences-state"

export function SettingsContent({
  section,
}: {
  section: Exclude<AppPreferencesTab, "about" | "updates">
}) {
  return (
    <section className="min-w-0 space-y-4 pb-1">
      {section === "settings" ? (
        <>
          <DesktopBehavior />
          <ProxyPortSetting />
          <AppearanceSettings />
        </>
      ) : null}
      {section === "activity" ? <RequestRetentionSettings /> : null}
      {section === "storage" ? (
        <>
          <DataDirForm />
          <LegacyImportCard />
          <StorageOverview />
        </>
      ) : null}
    </section>
  )
}
