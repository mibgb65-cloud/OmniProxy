import { createFileRoute } from "@tanstack/react-router"
import { PagePlaceholder } from "../../components/common/page-placeholder"

export const Route = createFileRoute("/usage")({
  component: UsagePage,
})

function UsagePage() {
  return <PagePlaceholder title="Usage" description="Token and request usage statistics." />
}
