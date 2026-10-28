import { useTranslation } from "react-i18next"
import { ProviderBrandIcon } from "../providers/provider-brand-icon"
import { isKnownProviderKind, providerNameKeys } from "../providers/provider-labels"
import type { ProviderConfiguration } from "../../services/omniproxy"
import { Tooltip } from "../../components/ui/tooltip"

/**
 * 底部 Dock 的迷你切换卡片：只保留品牌图标、名称与账号数，
 * 点击即在顶部面板切换查看该提供商，不产生路由跳转。
 */
export function ProviderDockCard({
  provider,
  accountCount,
  onSelect,
}: {
  provider: ProviderConfiguration
  accountCount: number
  onSelect: () => void
}) {
  const { t } = useTranslation()
  const name =
    provider.source === "builtin" && isKnownProviderKind(provider.kind)
      ? t(providerNameKeys[provider.kind])
      : provider.name

  return (
    <Tooltip content={name}>
      <button
        type="button"
        onClick={onSelect}
        aria-label={t("accounts.openDetails", { name })}
        className="bento-dock-card focus-visible:ring-ring/50 hover:bg-muted/70 flex h-full w-full cursor-pointer items-center gap-2.5 rounded-xl px-3 py-2.5 text-left transition-colors outline-none focus-visible:ring-2"
      >
        <ProviderBrandIcon kind={provider.kind} endpoint={provider.baseUrl} />
        <span className="min-w-0 flex-1">
          <span className="block truncate text-xs font-semibold">{name}</span>
          <span className="text-muted-foreground block truncate text-[11px]">
            {t("accounts.count", { count: accountCount })}
          </span>
        </span>
      </button>
    </Tooltip>
  )
}
