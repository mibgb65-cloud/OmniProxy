import { createFileRoute } from "@tanstack/react-router"
import { RequestsView } from "../../features/activity/requests-view"
import { validateRequestSearch } from "../../services/activity"

export const Route = createFileRoute("/requests")({
  component: RequestsPage,
  validateSearch: validateRequestSearch,
})

function RequestsPage() {
  const search = Route.useSearch()
  const navigate = Route.useNavigate()
  return (
    <RequestsView
      key={search.requestId ?? "all"}
      requestId={search.requestId}
      onOpenLogs={(id) => void navigate({ to: "/logs", search: { requestId: id } })}
    />
  )
}
