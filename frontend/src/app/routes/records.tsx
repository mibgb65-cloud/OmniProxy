import { createFileRoute } from "@tanstack/react-router"
import { RecordsView } from "../../features/records/records-view"

export const Route = createFileRoute("/records")({
  component: RecordsPage,
})

function RecordsPage() {
  const navigate = Route.useNavigate()
  return (
    <RecordsView
      onOpenRequest={(id) => void navigate({ to: "/requests", search: { requestId: id } })}
    />
  )
}
