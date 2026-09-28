import { useState } from "react"
import { useTranslation } from "react-i18next"
import type {
  AccountItem,
  CreateAccountInput,
  ProviderConfiguration,
} from "../../services/omniproxy"
import { useProviderCatalog, useProviders } from "../providers/use-providers"
import { AccountForm } from "./account-form"
import { AccountProviderCard } from "./account-provider-card"
import {
  useAccounts,
  useCreateAccount,
  useDeleteAccount,
  useSetAccountEnabled,
} from "./use-accounts"

export function AccountsView() {
  const { t } = useTranslation()
  const catalog = useProviderCatalog()
  const providers = useProviders()
  const accounts = useAccounts()
  const create = useCreateAccount()
  const setEnabled = useSetAccountEnabled()
  const remove = useDeleteAccount()
  const [selected, setSelected] = useState<ProviderConfiguration | null>(null)
  const [actionError, setActionError] = useState(false)

  const save = async (input: CreateAccountInput) => {
    await create.mutateAsync(input)
    setSelected(null)
  }

  const toggle = (id: string, enabled: boolean) => {
    setActionError(false)
    void setEnabled.mutateAsync({ id, enabled }).catch(() => setActionError(true))
  }

  const deleteItem = (account: AccountItem) => {
    if (!window.confirm(t("accounts.deleteConfirm", { name: account.name }))) return
    setActionError(false)
    void remove.mutateAsync(account.id).catch(() => setActionError(true))
  }

  const definitions = catalog.data ?? []
  const items = providers.data ?? []
  const orderedProviders = [
    ...definitions.flatMap((definition) =>
      items.filter((item) => item.source === "builtin" && item.kind === definition.kind),
    ),
    ...items.filter((item) => item.source === "custom"),
  ]

  return (
    <div className="mx-auto max-w-[1120px] space-y-7 pb-6">
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
      {selected ? (
        <AccountForm
          key={selected.id}
          provider={selected}
          onSubmit={save}
          onCancel={() => setSelected(null)}
        />
      ) : null}
      {actionError ? (
        <p role="alert" className="text-destructive text-sm">
          {t("accounts.actionFailed")}
        </p>
      ) : null}
      {catalog.data && providers.data && accounts.data ? (
        <div className="grid gap-3 md:grid-cols-2">
          {orderedProviders.map((provider) => (
            <AccountProviderCard
              key={provider.id}
              provider={provider}
              accounts={accounts.data.filter((account) => account.providerId === provider.id)}
              busy={setEnabled.isPending || remove.isPending}
              onAdd={() => {
                setActionError(false)
                setSelected(provider)
              }}
              onToggle={toggle}
              onDelete={deleteItem}
            />
          ))}
        </div>
      ) : null}
    </div>
  )
}
