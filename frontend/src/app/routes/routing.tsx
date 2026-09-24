import { createFileRoute } from "@tanstack/react-router"
import { PagePlaceholder } from "../../components/common/page-placeholder"

export const Route = createFileRoute("/routing")({
  component: RoutingPage,
})

function RoutingPage() {
  return (
    <PagePlaceholder titleKey="pages.routing.title" descriptionKey="pages.routing.description" />
  )
}
