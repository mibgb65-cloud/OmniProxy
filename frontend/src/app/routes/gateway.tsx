import { createFileRoute } from "@tanstack/react-router"
import { PagePlaceholder } from "../../components/common/page-placeholder"

export const Route = createFileRoute("/gateway")({
  component: GatewayPage,
})

function GatewayPage() {
  return (
    <PagePlaceholder titleKey="pages.gateway.title" descriptionKey="pages.gateway.description" />
  )
}
