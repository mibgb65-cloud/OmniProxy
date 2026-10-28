import { useEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { useProviderCatalog, useProviders } from "../providers/use-providers"
import { isKnownProviderKind, providerNameKeys } from "../providers/provider-labels"
import type { ProviderConfiguration } from "../../services/omniproxy"
import { AccountProviderCard } from "./account-provider-card"
import { DeleteAccountDialog } from "./delete-account-dialog"
import { EditAccountDialog } from "./edit-account-dialog"
import { ProviderAccountsView } from "./provider-accounts-view"
import { ProviderSwitcherDock } from "./provider-switcher-dock"
import { useAccountActions } from "./use-account-actions"
import { useAccounts } from "./use-accounts"
import { useFlipReflow } from "./use-flip-reflow"
import "./account-card-grid.css"
import "./bento-accounts.css"

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

  /**
   * 展开只改本页状态、不改路由：被点开的提供商在网格顶部横跨整行展开详情，
   * 其余提供商压缩成底部 Dock；切换与收起共用同一条 FLIP 过渡。
   */
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const panelRef = useRef<HTMLDivElement>(null)
  const lastExpandedId = useRef<string | null>(null)
  const captureFlip = useFlipReflow(containerRef)

  const expanded = orderedProviders.find((provider) => provider.id === expandedId) ?? null
  const dockProviders = orderedProviders.filter((provider) => provider.id !== expanded?.id)
  const accountCounts = new Map<string, number>()
  for (const account of accounts.data ?? []) {
    accountCounts.set(account.providerId, (accountCounts.get(account.providerId) ?? 0) + 1)
  }

  const showProvider = (providerId: string | null) => {
    if (providerId === expandedId) return
    // 必须先在旧布局下取样，布局提交后才由 FLIP 补间到新位置。
    captureFlip()
    setExpandedId(providerId)
  }

  // 展开后把焦点交给面板，收起后交回对应卡片，避免焦点落到页面根节点。
  useEffect(() => {
    if (expandedId) {
      lastExpandedId.current = expandedId
      panelRef.current?.focus({ preventScroll: true })
      return
    }
    const previous = lastExpandedId.current
    lastExpandedId.current = null
    if (!previous) return
    const slots = containerRef.current?.querySelectorAll<HTMLElement>("[data-flip-id]") ?? []
    const slot = Array.from(slots).find((element) => element.dataset.flipId === previous)
    slot?.querySelector<HTMLElement>("button")?.focus({ preventScroll: true })
  }, [expandedId])

  const displayName = (provider: ProviderConfiguration) =>
    provider.source === "builtin" && isKnownProviderKind(provider.kind)
      ? t(providerNameKeys[provider.kind])
      : provider.name

  return (
    <div className="accounts-page w-full min-w-0 space-y-7 pb-6">
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
      {actions.updateWarning ? (
        <p role="alert" className="text-destructive text-sm">
          {t("accounts.edit.cleanupWarning")}
        </p>
      ) : null}
      {catalog.data && providers.data && accounts.data ? (
        <div ref={containerRef} className="bento-accounts" data-expanded={!!expanded}>
          {expanded ? (
            <>
              <div
                key={expanded.id}
                className="bento-flip bento-panel-slot"
                data-flip-id={expanded.id}
              >
                <div
                  ref={panelRef}
                  role="region"
                  tabIndex={-1}
                  aria-label={displayName(expanded)}
                  className="bento-panel outline-none"
                >
                  <ProviderAccountsView
                    providerId={expanded.id}
                    embedded
                    onCollapse={() => showProvider(null)}
                  />
                </div>
              </div>
              {dockProviders.length ? (
                <ProviderSwitcherDock
                  providers={dockProviders}
                  accountCounts={accountCounts}
                  onSelect={showProvider}
                />
              ) : null}
            </>
          ) : (
            orderedProviders.map((provider) => (
              <div key={provider.id} className="bento-flip" data-flip-id={provider.id}>
                <AccountProviderCard
                  provider={provider}
                  accounts={accounts.data.filter((account) => account.providerId === provider.id)}
                  busy={actions.busy}
                  onCreate={actions.save}
                  onBatchCreate={actions.saveBatch}
                  onToggle={actions.toggle}
                  onDelete={actions.deleteItem}
                  onEdit={actions.editItem}
                  onExpand={() => showProvider(provider.id)}
                />
              </div>
            ))
          )}
        </div>
      ) : null}
      <DeleteAccountDialog actions={actions} />
      <EditAccountDialog actions={actions} />
    </div>
  )
}
