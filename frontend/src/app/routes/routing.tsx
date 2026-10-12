import { createFileRoute } from "@tanstack/react-router"
import { RoutingView } from "../../features/routing/routing-view"
import { validateRoutingSearch } from "../../features/routing/routing-tabs"

export const Route = createFileRoute("/routing")({
  component: RoutingPage,
  validateSearch: validateRoutingSearch,
})

function RoutingPage() {
  const { tab } = Route.useSearch()
  const navigate = Route.useNavigate()
  return (
    <RoutingView
      tab={tab ?? "defaults"}
      onTabChange={(next) => void navigate({ search: { tab: next }, resetScroll: false })}
    />
  )
}
