import { useTranslation } from "react-i18next"
import { useProviderCatalog, useProviders } from "../providers/use-providers"
import { AccountProviderCard } from "./account-provider-card"
import { useAccountActions } from "./use-account-actions"
import { useAccounts } from "./use-accounts"

export function AccountsView() {
  const { t } = useTranslation()
  const catalog = useProviderCatalog()
  const providers = useProviders()
  const accounts = useAccounts()
  const actions = useAccountActions()

  const definitions = catalog.data ?? []
  const items = providers.data ?? []
  const orderedProviders = [
    ...definitions.flatMap((definition) =>
      items.filter((item) => item.source === "builtin" && item.kind === definition.kind),
    ),
    ...items.filter((item) => item.source === "custom"),
  ]

  return (
    <div className="w-full min-w-0 space-y-7 pb-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">{t("pages.accounts.title")}</h1>
      </header>
      {catalog.isPending || providers.isPending || accounts.isPending ? (
        <p role="status" className="text-muted-foreground text-sm">
          {t("accounts.loading")}
        </p>
      ) : null}
      {catalog.isError || providers.isError || accounts.isError ? (
        <p role="alert" className="text-destructive text-sm">
          {t("accounts.loadFailed")}
        </p>
      ) : null}
      {actions.actionError ? (
        <p role="alert" className="text-destructive text-sm">
          {t("accounts.actionFailed")}
        </p>
      ) : null}
      {catalog.data && providers.data && accounts.data ? (
        <div className="grid items-start gap-3 min-[3000px]:grid-cols-4 md:grid-cols-2 2xl:grid-cols-3">
          {orderedProviders.map((provider) => (
            <AccountProviderCard
              key={provider.id}
              provider={provider}
              accounts={accounts.data.filter((account) => account.providerId === provider.id)}
              busy={actions.busy}
              onCreate={actions.save}
              onToggle={actions.toggle}
              onDelete={actions.deleteItem}
            />
          ))}
        </div>
      ) : null}
    </div>
  )
}
