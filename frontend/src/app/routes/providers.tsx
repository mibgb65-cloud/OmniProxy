import { createFileRoute } from "@tanstack/react-router"
import { ProvidersView } from "../../features/providers/providers-view"

export const Route = createFileRoute("/providers")({
  component: ProvidersPage,
})

function ProvidersPage() {
  return <ProvidersView />
}
