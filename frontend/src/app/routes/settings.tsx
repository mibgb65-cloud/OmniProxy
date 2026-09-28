import { createFileRoute } from "@tanstack/react-router"
import { useTranslation } from "react-i18next"
import { DataDirForm } from "../../features/settings/data-dir-form"
import { DesktopBehavior } from "../../features/settings/desktop-behavior"
import { StorageOverview } from "../../features/settings/storage-overview"

export const Route = createFileRoute("/settings")({
  component: SettingsPage,
})

function SettingsPage() {
  const { t } = useTranslation()
  return (
    <section className="w-full min-w-0 space-y-6 pb-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">{t("settings.title")}</h1>
        <p className="text-muted-foreground mt-1 text-sm">{t("settings.description")}</p>
      </header>
      <StorageOverview />
      <DataDirForm />
      <DesktopBehavior />
    </section>
  )
}
