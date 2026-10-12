import { createFileRoute } from "@tanstack/react-router"
import { LogsView } from "../../features/logs/logs-view"
import { validateRequestSearch } from "../../services/activity"

export const Route = createFileRoute("/logs")({
  component: LogsPage,
  validateSearch: validateRequestSearch,
})

function LogsPage() {
  const search = Route.useSearch()
  const navigate = Route.useNavigate()
  return (
    <LogsView
      key={search.requestId ?? "all"}
      requestId={search.requestId}
      onOpenRequest={(id) => void navigate({ to: "/requests", search: { requestId: id } })}
    />
  )
}
