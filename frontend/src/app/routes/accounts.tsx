import { createFileRoute } from "@tanstack/react-router"
import { PagePlaceholder } from "../../components/common/page-placeholder"

export const Route = createFileRoute("/accounts")({
  component: AccountsPage,
})

function AccountsPage() {
  return (
    <PagePlaceholder
      title="Accounts"
      description="Multi-account management. The account backend (internal/core/account) lands in a later milestone."
    />
  )
}
