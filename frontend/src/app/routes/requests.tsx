import { createFileRoute } from "@tanstack/react-router"
import { PagePlaceholder } from "../../components/common/page-placeholder"

export const Route = createFileRoute("/requests")({
  component: RequestsPage,
})

function RequestsPage() {
  return <PagePlaceholder title="Requests" description="Request history and failure analysis." />
}
