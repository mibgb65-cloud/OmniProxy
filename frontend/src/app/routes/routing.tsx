import { createFileRoute } from "@tanstack/react-router"
import { PagePlaceholder } from "../../components/common/page-placeholder"

export const Route = createFileRoute("/routing")({
  component: RoutingPage,
})

function RoutingPage() {
  return (
    <PagePlaceholder
      title="Routing"
      description="Model routing rules. The routing engine lands in a later milestone."
    />
  )
}
