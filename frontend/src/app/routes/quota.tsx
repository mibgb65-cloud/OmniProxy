import { createFileRoute } from "@tanstack/react-router"
import { QuotaView } from "../../features/quota/quota-view"

export const Route = createFileRoute("/quota")({
  component: QuotaView,
})
