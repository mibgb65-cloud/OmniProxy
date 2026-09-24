import { createFileRoute } from "@tanstack/react-router"
import { PagePlaceholder } from "../../components/common/page-placeholder"

export const Route = createFileRoute("/accounts")({
  component: AccountsPage,
})

function AccountsPage() {
  return (
    <PagePlaceholder titleKey="pages.accounts.title" descriptionKey="pages.accounts.description" />
  )
}
