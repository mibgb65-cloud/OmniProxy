import { createFileRoute } from "@tanstack/react-router"
import { PagePlaceholder } from "../../components/common/page-placeholder"

export const Route = createFileRoute("/gateway")({
  component: GatewayPage,
})

function GatewayPage() {
  return (
    <PagePlaceholder
      title="Gateway"
      description="Proxy server status and control. The gateway backend lands in a later milestone."
    />
  )
}
