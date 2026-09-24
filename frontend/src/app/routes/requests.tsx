import { createFileRoute } from "@tanstack/react-router"
import { PagePlaceholder } from "../../components/common/page-placeholder"

export const Route = createFileRoute("/requests")({
  component: RequestsPage,
})

function RequestsPage() {
  return (
    <PagePlaceholder titleKey="pages.requests.title" descriptionKey="pages.requests.description" />
  )
}
