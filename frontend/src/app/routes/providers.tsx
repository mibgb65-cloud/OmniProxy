import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { ProvidersView } from "../../features/providers/providers-view"

export const Route = createFileRoute("/providers")({
  component: ProvidersPage,
})

function ProvidersPage() {
  const navigate = useNavigate()
  return (
    <ProvidersView
      onOpenProvider={(providerId) =>
        void navigate({ to: "/accounts/$providerId", params: { providerId } })
      }
    />
  )
}
