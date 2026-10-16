import { Link } from "@tanstack/react-router"
import { ArrowLeft } from "lucide-react"
import { useTranslation } from "react-i18next"
import { useUIStore } from "../../stores/ui-store"
import { ProviderBrandIcon } from "../providers/provider-brand-icon"
import { isKnownProviderKind, providerNameKeys } from "../providers/provider-labels"
import { useProviders } from "../providers/use-providers"
import { AccountDetailsCard } from "./account-details-card"
import { AccountDetailsList } from "./account-details-list"
import { AccountViewToggle } from "./account-view-toggle"
import { AddAccountDialog } from "./add-account-dialog"
import { DeleteAccountDialog } from "./delete-account-dialog"
import { EditAccountDialog } from "./edit-account-dialog"
import { useAccountActions } from "./use-account-actions"
import { useAccounts } from "./use-accounts"
import { OpenAIAccountSettings } from "./openai-account-settings"
import { MiMoAccountSettings } from "./mimo-account-settings"
import { StepFunAccountSettings } from "./stepfun-account-settings"
import { GoogleResources } from "../google/google-resources"
import "./account-card-grid.css"

export function ProviderAccountsView({ providerId }: { providerId: string }) {
  const { t } = useTranslation()
  const providers = useProviders()
  const accounts = useAccounts()
  const actions = useAccountActions()
  const view = useUIStore((state) => state.accountView)
  const provider = providers.data?.find((item) => item.id === providerId)
  const name =
    provider?.source === "builtin" && isKnownProviderKind(provider.kind)
      ? t(providerNameKeys[provider.kind])
      : (provider?.name ?? "")
  const items = accounts.data?.filter((account) => account.providerId === providerId) ?? []

  return (
    <section className="w-full min-w-0 space-y-6 pb-6">
      <Link
        to="/accounts"
        className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1.5 text-sm"
      >
        <ArrowLeft className="size-4" aria-hidden="true" />
        {t("accounts.detail.back")}
      </Link>
      {providers.isPending || accounts.isPending ? (
        <p role="status" className="text-muted-foreground text-sm">
          {t("accounts.loading")}
        </p>
      ) : null}
      {providers.isError || accounts.isError ? (
        <p role="alert" className="text-destructive text-sm">
          {t("accounts.loadFailed")}
        </p>
      ) : null}
      {providers.data && accounts.data && !provider ? (
        <p role="alert" className="text-muted-foreground text-sm">
          {t("accounts.detail.notFound")}
        </p>
      ) : null}
      {provider && accounts.data ? (
        <>
          <header className="flex flex-wrap items-center justify-between gap-4">
            <div className="flex min-w-0 items-center gap-3">
              <ProviderBrandIcon kind={provider.kind} endpoint={provider.baseUrl} />
              <div className="min-w-0">
                <h1 className="truncate text-2xl font-semibold tracking-tight">
                  {t("accounts.detail.title", { name })}
                </h1>
                <p className="text-muted-foreground text-sm">
                  {t("accounts.count", { count: items.length })}
                </p>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              {provider.kind === "openai" ? <OpenAIAccountSettings /> : null}
              {provider.kind === "mimo" ? <MiMoAccountSettings /> : null}
              {provider.kind === "stepfun" ? <StepFunAccountSettings /> : null}
              <AccountViewToggle />
              <AddAccountDialog
                provider={provider}
                onCreate={actions.save}
                onBatchCreate={actions.saveBatch}
              />
            </div>
          </header>
          {!provider.enabled ? (
            <p className="text-muted-foreground text-sm">{t("accounts.providerDisabled")}</p>
          ) : null}
          {provider.kind !== "stepfun" &&
            provider.kind !== "kimi" &&
            provider.kind !== "mimo" &&
            provider.kind !== "deepseek" &&
            provider.kind !== "google" &&
            (provider.kind !== "openai" ||
            items.some((account) => account.authMethod !== "subscription_login") ? (
              <p className="text-muted-foreground text-sm">{t("accounts.detail.balanceNotice")}</p>
            ) : null)}
          {provider.kind === "stepfun" ? (
            <p className="text-muted-foreground text-sm leading-6">{t("stepfun.routingHint")}</p>
          ) : null}
          {provider.kind === "mimo" ? (
            <p className="text-muted-foreground text-sm leading-6">{t("mimo.routingHint")}</p>
          ) : null}
          {provider.kind === "kimi" ? (
            <p className="text-muted-foreground text-sm leading-6">{t("kimi.routingHint")}</p>
          ) : null}
          {provider.kind === "google" ? (
            <>
              <p className="text-muted-foreground text-sm">{t("google.chatOnly")}</p>
              <GoogleResources
                accounts={items.filter((account) => account.authMethod === "api_key")}
              />
            </>
          ) : null}
          {actions.actionError ? (
            <p role="alert" className="text-destructive text-sm">
              {t("accounts.actionFailed")}
            </p>
          ) : null}
          {actions.updateWarning ? (
            <p role="alert" className="text-destructive text-sm">
              {t("accounts.edit.cleanupWarning")}
            </p>
          ) : null}
          {items.length === 0 ? (
            <p className="text-muted-foreground rounded-lg border border-dashed px-4 py-6 text-sm">
              {t("accounts.empty")}
            </p>
          ) : view === "list" ? (
            <AccountDetailsList
              accounts={items}
              busy={actions.busy}
              onToggle={actions.toggle}
              onDelete={actions.deleteItem}
              onEdit={actions.editItem}
              balanceSupported={["deepseek", "stepfun", "kimi"].includes(provider.kind)}
            />
          ) : (
            <div className="account-card-grid">
              {items.map((account) => (
                <AccountDetailsCard
                  key={account.id}
                  account={account}
                  busy={actions.busy}
                  onToggle={actions.toggle}
                  onDelete={actions.deleteItem}
                  onEdit={actions.editItem}
                  balanceSupported={["deepseek", "stepfun", "kimi"].includes(provider.kind)}
                  googleSupported={provider.kind === "google"}
                />
              ))}
            </div>
          )}
        </>
      ) : null}
      <DeleteAccountDialog actions={actions} />
      <EditAccountDialog actions={actions} />
    </section>
  )
}
