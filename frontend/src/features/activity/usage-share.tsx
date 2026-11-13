import type { ReactNode } from "react"
import { useTranslation } from "react-i18next"
import { formatTokens, type Breakdown, type Summary } from "../../services/activity"
import { cn } from "../../lib/utils"
import { ShareBar, UsageCard, UsageEmpty } from "./usage-charts"
import {
  MODEL_COLORS,
  OTHER_COLOR,
  usageCountLabel,
  usagePercentLabel,
  usageShare,
  usageSum,
} from "./usage-model"

const SHARE_LIMIT = 6

function ShareRow({
  label,
  percent,
  value,
  meta,
  colorClass,
  badge,
  language,
}: {
  label: string
  percent: number
  value: string
  meta: string
  colorClass: string
  badge?: ReactNode
  language: string
}) {
  return (
    <li className="min-w-0 space-y-1.5">
      <div className="flex items-baseline justify-between gap-3">
        <span className="flex min-w-0 items-center gap-2">
          <span className={cn("size-2 shrink-0 rounded-full", colorClass)} aria-hidden="true" />
          <span className="truncate text-sm font-medium">{label}</span>
          {badge}
        </span>
        <span className="text-muted-foreground shrink-0 font-mono text-xs tabular-nums">
          {usagePercentLabel(percent, language)}
        </span>
      </div>
      <ShareBar percent={percent} className={colorClass} />
      <div className="text-muted-foreground flex justify-between gap-3 text-[11px]">
        <span className="min-w-0 truncate">{meta}</span>
        <span className="shrink-0 font-mono tabular-nums">{value}</span>
      </div>
    </li>
  )
}

function shortID(value: string) {
  return `${value.slice(0, 8)}…${value.slice(-6)}`
}

export function ModelShareCard({ items, total }: { items: Breakdown[]; total: Summary }) {
  const { t, i18n } = useTranslation()
  const language = i18n.resolvedLanguage ?? "zh"
  const visible = items.slice(0, SHARE_LIMIT)
  const rest = items.slice(SHARE_LIMIT)
  const restSummary = usageSum(rest)
  const requestMeta = (count: number) =>
    t("activity.requestCount", { value: usageCountLabel(count, language) })
  return (
    <UsageCard title={t("activity.modelShare")} hint={t("activity.modelShareHint")}>
      {items.length ? (
        <ul className="space-y-3">
          {visible.map((item, index) => (
            <ShareRow
              key={item.key}
              label={item.name || item.key || t("activity.noModel")}
              percent={usageShare(item.summary.totalTokens, total.totalTokens)}
              value={formatTokens(item.summary.totalTokens, language)}
              meta={requestMeta(item.summary.requests)}
              colorClass={MODEL_COLORS[index % MODEL_COLORS.length]!}
              language={language}
            />
          ))}
          {rest.length ? (
            <ShareRow
              label={t("activity.otherModels", { count: rest.length })}
              percent={usageShare(restSummary.totalTokens, total.totalTokens)}
              value={formatTokens(restSummary.totalTokens, language)}
              meta={requestMeta(restSummary.requests)}
              colorClass={OTHER_COLOR}
              language={language}
            />
          ) : null}
        </ul>
      ) : (
        <UsageEmpty>{t("activity.empty")}</UsageEmpty>
      )}
    </UsageCard>
  )
}

export function ProviderShareCard({ items, total }: { items: Breakdown[]; total: Summary }) {
  const { t, i18n } = useTranslation()
  const language = i18n.resolvedLanguage ?? "zh"
  return (
    <UsageCard title={t("activity.providerShare")} hint={t("activity.providerShareHint")}>
      {items.length ? (
        <ul className="app-scroll max-h-96 space-y-3 overflow-auto pr-1">
          {items.map((item, index) => {
            const legacy = item.key.startsWith("legacy:")
            const name =
              item.name || (legacy ? item.key.slice(7) : item.key) || t("activity.provider")
            return (
              <ShareRow
                key={item.key}
                label={name}
                percent={usageShare(item.summary.totalTokens, total.totalTokens)}
                value={formatTokens(item.summary.totalTokens, language)}
                meta={t("activity.requestCount", {
                  value: usageCountLabel(item.summary.requests, language),
                })}
                colorClass={MODEL_COLORS[index % MODEL_COLORS.length]!}
                language={language}
                badge={
                  legacy ? (
                    <span className="text-muted-foreground bg-muted rounded px-1.5 py-0.5 text-[10px]">
                      {t("activity.legacyTag")}
                    </span>
                  ) : null
                }
              />
            )
          })}
        </ul>
      ) : (
        <UsageEmpty>{t("activity.empty")}</UsageEmpty>
      )}
    </UsageCard>
  )
}

export function AccountShareCard({ items, total }: { items: Breakdown[]; total: Summary }) {
  const { t, i18n } = useTranslation()
  const language = i18n.resolvedLanguage ?? "zh"
  const visible = items.slice(0, SHARE_LIMIT)
  const rest = items.slice(SHARE_LIMIT)
  const restSummary = usageSum(rest)
  return (
    <UsageCard title={t("activity.accounts")} hint={t("activity.accountShareHint")}>
      {items.length ? (
        <ul className="app-scroll max-h-96 space-y-3 overflow-auto pr-1">
          {visible.map((item) => {
            const input = BigInt(item.summary.inputTokens)
            const output = BigInt(item.summary.outputTokens)
            const itemTotal = input + output
            const percent = usageShare(itemTotal, total.totalTokens)
            const requests = t("activity.requestCount", {
              value: usageCountLabel(item.summary.requests, language),
            })
            return (
              <li key={item.key} className="min-w-0 space-y-1.5">
                <div className="flex items-baseline justify-between gap-3">
                  <span className="truncate text-sm font-medium">
                    {item.email || item.name || t("activity.noAccount")}
                  </span>
                  <span className="text-muted-foreground shrink-0 font-mono text-xs tabular-nums">
                    {usagePercentLabel(percent, language)}
                  </span>
                </div>
                <ShareBar
                  percent={percent}
                  segments={[
                    {
                      key: "input",
                      percent: usageShare(input, itemTotal),
                      className: "bg-blue-500",
                    },
                    {
                      key: "output",
                      percent: usageShare(output, itemTotal),
                      className: "bg-emerald-500",
                    },
                  ]}
                />
                <div className="text-muted-foreground flex flex-wrap justify-between gap-2 text-[11px]">
                  <span className="min-w-0 truncate font-mono">
                    {item.workspaceId ? `${shortID(item.workspaceId)} · ${requests}` : requests}
                  </span>
                  <span className="font-mono tabular-nums">
                    {formatTokens(item.summary.totalTokens, language)}
                  </span>
                </div>
              </li>
            )
          })}
          {rest.length ? (
            <li className="text-muted-foreground flex items-center justify-between gap-3 border-t pt-3 text-[11px]">
              <span>{t("activity.otherAccounts", { count: rest.length })}</span>
              <span className="font-mono tabular-nums">
                {formatTokens(restSummary.totalTokens, language)}
              </span>
            </li>
          ) : null}
        </ul>
      ) : (
        <UsageEmpty>{t("activity.empty")}</UsageEmpty>
      )}
    </UsageCard>
  )
}
