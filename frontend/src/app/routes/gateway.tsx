import { createFileRoute } from "@tanstack/react-router"
import { GatewayView } from "../../features/gateway/gateway-view"
import { validateGatewaySearch } from "../../features/gateway/gateway-tabs"

export const Route = createFileRoute("/gateway")({
  component: GatewayPage,
  validateSearch: validateGatewaySearch,
})

function GatewayPage() {
  const { tab, client } = Route.useSearch()
  const navigate = Route.useNavigate()
  return (
    <GatewayView
      tab={tab ?? "overview"}
      client={client}
      onTabChange={(next) =>
        void navigate({ search: { tab: next, ...(client ? { client } : {}) }, resetScroll: false })
      }
    />
  )
}
