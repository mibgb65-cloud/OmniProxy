import { createFileRoute } from "@tanstack/react-router"
import { AccountsView } from "../../features/accounts/accounts-view"

export const Route = createFileRoute("/accounts")({
  component: AccountsPage,
})

function AccountsPage() {
  return <AccountsView />
}
