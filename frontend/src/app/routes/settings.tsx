import { createFileRoute } from "@tanstack/react-router"
import { PagePlaceholder } from "../../components/common/page-placeholder"

export const Route = createFileRoute("/settings")({
  component: SettingsPage,
})

function SettingsPage() {
  return (
    <PagePlaceholder titleKey="pages.settings.title" descriptionKey="pages.settings.description" />
  )
}
