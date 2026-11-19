import { Link } from "@tanstack/react-router"
import { RiverAccountSettings } from "./river-account-settings"
import { GoogleAccountSettings } from "./google-account-settings"
import { ArrowLeft } from "lucide-react"
import { useTranslation } from "react-i18next"
import { useUIStore } from "../../stores/ui-store"
import { ProviderBrandIcon } from "../providers/provider-brand-icon"
import { isKnownProviderKind, providerNameKeys } from "../providers/provider-labels"
import { useProviders } from "../providers/use-providers"
import { AccountDetailsCard } from "./account-details-card"
import { AccountDetailsList } from "./account-details-list"
import { AccountQuotaGroupSection } from "./account-quota-group-section"
import { AccountViewToggle } from "./account-view-toggle"
import { AddAccountDialog } from "./add-account-dialog"
import { DeleteAccountDialog } from "./delete-account-dialog"
import { EditAccountDialog } from "./edit-account-dialog"
import { useAccountActions } from "./use-account-actions"
import { useAccounts } from "./use-accounts"
import { useGoogleQuotaGroupSections } from "../google/use-google-quota-group-sections"
import type { AccountItem } from "../../services/omniproxy"
import { OpenAIAccountSettings } from "./openai-account-settings"
import { MiMoAccountSettings } from "./mimo-account-settings"
import { StepFunAccountSettings } from "./stepfun-account-settings"
import { KimiAccountSettings } from "./kimi-account-settings"
import "./account-card-grid.css"

export function ProviderAccountsView({
  providerId,
  embedded = false,
  onCollapse,
}: {
  providerId: string
  /** 作为概览页原位展开面板的内容时使用：标题降为 h2，不渲染返回链接。 */
  embedded?: boolean
  onCollapse?: () => void
}) {
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
  const Title = embedded ? "h2" : "h1"
  /**
   * Google 账号页按额度分组集合分区：含 Gemini 4 Argon 之类额外分组的账号自成一区，
   * 分组相同的账号相邻。只有一个分区时退回原来的平铺，不显示多余标题。
   */
  const googleProvider = provider?.kind === "google"
  const sections = useGoogleQuotaGroupSections(items, googleProvider)
  const showSections = googleProvider && sections.length > 1
  const balanceSupported = ["deepseek", "stepfun", "kimi", "howtofix"].includes(
    provider?.kind ?? "",
  )

  const cardGrid = (list: AccountItem[]) => (
    <div className="account-card-grid">
      {list.map((account) => (
        <AccountDetailsCard
          key={account.id}
          account={account}
          busy={actions.busy}
          onToggle={actions.toggle}
          onDelete={actions.deleteItem}
          onEdit={actions.editItem}
          balanceSupported={balanceSupported}
          googleSupported={googleProvider}
          googleAccounts={items.filter((item) => item.authMethod === "api_key")}
        />
      ))}
    </div>
  )

  const accountList = (list: AccountItem[]) => (
    <AccountDetailsList
      accounts={list}
      busy={actions.busy}
      onToggle={actions.toggle}
      onDelete={actions.deleteItem}
      onEdit={actions.editItem}
      balanceSupported={balanceSupported}
      googleSupported={googleProvider}
    />
  )

  return (
    <section className={embedded ? "w-full min-w-0 space-y-5" : "w-full min-w-0 space-y-6 pb-6"}>
      {embedded ? (
        <button
          type="button"
          onClick={onCollapse}
          className="text-muted-foreground hover:text-foreground focus-visible:ring-ring/50 inline-flex cursor-pointer items-center gap-1.5 rounded-sm text-sm outline-none focus-visible:ring-4"
        >
          <ArrowLeft className="size-4" aria-hidden="true" />
          {t("accounts.detail.back")}
        </button>
      ) : (
        <Link
          to="/accounts"
          className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1.5 text-sm"
        >
          <ArrowLeft className="size-4" aria-hidden="true" />
          {t("accounts.detail.back")}
        </Link>
      )}
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
                <Title className="truncate text-2xl font-semibold tracking-tight">
                  {t("accounts.detail.title", { name })}
                </Title>
                <p className="text-muted-foreground text-sm">
                  {t("accounts.count", { count: items.length })}
                </p>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              {provider.kind === "openai" ? <OpenAIAccountSettings /> : null}
              {provider.kind === "river" ? <RiverAccountSettings /> : null}
              {provider.kind === "google" ? <GoogleAccountSettings /> : null}
              {provider.kind === "mimo" ? <MiMoAccountSettings /> : null}
              {provider.kind === "stepfun" ? <StepFunAccountSettings /> : null}
              {provider.kind === "kimi" ? <KimiAccountSettings /> : null}
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
            provider.kind !== "river" &&
            provider.kind !== "kimi" &&
            provider.kind !== "mimo" &&
            provider.kind !== "deepseek" &&
            provider.kind !== "google" &&
            provider.kind !== "howtofix" &&
            provider.kind !== "dumate" &&
            provider.kind !== "workbuddy_ai" &&
            provider.kind !== "workbuddy" &&
            (provider.kind !== "openai" ||
            items.some((account) => account.authMethod !== "subscription_login") ? (
              <p className="text-muted-foreground text-sm">{t("accounts.detail.balanceNotice")}</p>
            ) : null)}
          {provider.kind === "dumate" ? (
            <p className="text-muted-foreground text-sm leading-6">{t("dumate.account.hint")}</p>
          ) : null}
          {provider.kind === "stepfun" ? (
            <p className="text-muted-foreground text-sm leading-6">{t("stepfun.routingHint")}</p>
          ) : null}
          {provider.kind === "mimo" ? (
            <p className="text-muted-foreground text-sm leading-6">{t("mimo.routingHint")}</p>
          ) : null}
          {provider.kind === "kimi" ? (
            <p className="text-muted-foreground text-sm leading-6">{t("kimi.routingHint")}</p>
          ) : null}
          {provider.kind === "river" ? (
            <p className="text-muted-foreground text-sm leading-6">{t("river.usage.notice")}</p>
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
          ) : showSections ? (
            <div className="space-y-6">
              <p className="text-muted-foreground text-xs leading-5">
                {t("accounts.quotaGroup.hint")}
              </p>
              {sections.map((section) => (
                <AccountQuotaGroupSection key={section.key} section={section}>
                  {view === "list" ? accountList(section.accounts) : cardGrid(section.accounts)}
                </AccountQuotaGroupSection>
              ))}
            </div>
          ) : view === "list" ? (
            accountList(items)
          ) : (
            cardGrid(items)
          )}
        </>
      ) : null}
      <DeleteAccountDialog actions={actions} />
      <EditAccountDialog actions={actions} />
    </section>
  )
}
