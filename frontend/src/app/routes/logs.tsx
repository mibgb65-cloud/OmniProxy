import { createFileRoute } from "@tanstack/react-router"
import { PagePlaceholder } from "../../components/common/page-placeholder"

export const Route = createFileRoute("/logs")({
  component: LogsPage,
})

function LogsPage() {
  return (
    <PagePlaceholder
      title="Logs"
      description="Structured application logs streamed from the Go backend."
    />
  )
}
