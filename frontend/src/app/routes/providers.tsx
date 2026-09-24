import { createFileRoute } from "@tanstack/react-router"
import { PagePlaceholder } from "../../components/common/page-placeholder"

export const Route = createFileRoute("/providers")({
  component: ProvidersPage,
})

function ProvidersPage() {
  return (
    <PagePlaceholder
      titleKey="pages.providers.title"
      descriptionKey="pages.providers.description"
    />
  )
}
