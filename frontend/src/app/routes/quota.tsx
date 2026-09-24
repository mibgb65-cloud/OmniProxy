import { createFileRoute } from "@tanstack/react-router"
import { PagePlaceholder } from "../../components/common/page-placeholder"

export const Route = createFileRoute("/quota")({
  component: QuotaPage,
})

function QuotaPage() {
  return <PagePlaceholder titleKey="pages.quota.title" descriptionKey="pages.quota.description" />
}
