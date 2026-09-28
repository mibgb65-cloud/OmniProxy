import { createFileRoute } from "@tanstack/react-router"
import { ProviderAccountsView } from "../../features/accounts/provider-accounts-view"

export const Route = createFileRoute("/accounts_/$providerId")({
  component: ProviderAccountsPage,
})

function ProviderAccountsPage() {
  const { providerId } = Route.useParams()
  return <ProviderAccountsView providerId={providerId} />
}
