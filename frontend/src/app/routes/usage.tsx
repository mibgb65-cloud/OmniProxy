import { createFileRoute } from "@tanstack/react-router"
import { UsageView } from "../../features/activity/usage-view"

export const Route = createFileRoute("/usage")({
  component: UsagePage,
})

function UsagePage() {
  return <UsageView />
}
