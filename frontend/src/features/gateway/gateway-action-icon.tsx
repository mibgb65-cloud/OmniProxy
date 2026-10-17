import { LoaderCircle, type LucideIcon } from "lucide-react"
import "./gateway-action-feedback.css"

export function GatewayActionIcon({ pending, icon: Icon }: { pending: boolean; icon: LucideIcon }) {
  return (
    <span
      aria-hidden="true"
      data-pending={pending}
      className="gateway-action-icon relative size-4 shrink-0"
    >
      <Icon className="gateway-action-idle absolute inset-0 size-4" />
      <LoaderCircle className="gateway-action-spinner absolute inset-0 size-4" />
    </span>
  )
}
