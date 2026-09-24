import { createFileRoute } from "@tanstack/react-router"
import { PagePlaceholder } from "../../components/common/page-placeholder"

export const Route = createFileRoute("/providers")({
  component: ProvidersPage,
})

function ProvidersPage() {
  return (
    <PagePlaceholder
      title="Providers"
      description="Provider configuration and health. The provider backend lands in a later milestone."
    />
  )
}
