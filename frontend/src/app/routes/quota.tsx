import { createFileRoute } from "@tanstack/react-router"
import { PagePlaceholder } from "../../components/common/page-placeholder"

export const Route = createFileRoute("/quota")({
  component: QuotaPage,
})

function QuotaPage() {
  return <PagePlaceholder title="Quota" description="Quota monitoring per account and provider." />
}
