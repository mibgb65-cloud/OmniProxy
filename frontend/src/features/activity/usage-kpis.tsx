import type { ComponentType, ReactNode } from "react"
import {
  ChartPie,
  Database,
  Gauge,
  Minus,
  ShieldCheck,
  Sigma,
  TrendingDown,
  TrendingUp,
  Waypoints,
} from "lucide-react"
import { useTranslation } from "react-i18next"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import { formatTokens } from "../../services/activity"
import type { CacheUsage, Reliability, Summary } from "../../services/activity"
import { chartDuration } from "../../lib/chart"
import {
  usageCountLabel,
  usageDelta,
  usagePercentLabel,
  usageRequestAverage,
  usageShare,
  type UsageDelta,
} from "./usage-model"

function DeltaBadge({ delta }: { delta: UsageDelta | null }) {
  const { t, i18n } = useTranslation()
  if (!delta) return null
  if (delta.kind === "new") {
    return (
      <span className="inline-flex items-center gap-1 rounded-full bg-emerald-500/12 px-2 py-0.5 text-[11px] font-medium text-emerald-600 dark:text-emerald-400">
        {t("activity.deltaNew")}
      </span>
    )
  }
  const negative = delta.kind === "down"
  const Icon = delta.kind === "up" ? TrendingUp : negative ? TrendingDown : Minus
  return (
    <span
      className={
        delta.kind === "flat"
          ? "text-muted-foreground inline-flex items-center gap-1 text-[11px]"
          : negative
            ? "inline-flex items-center gap-1 rounded-full bg-rose-500/10 px-2 py-0.5 text-[11px] font-medium text-rose-600 dark:text-rose-400"
            : "inline-flex items-center gap-1 rounded-full bg-emerald-500/12 px-2 py-0.5 text-[11px] font-medium text-emerald-600 dark:text-emerald-400"
      }
    >
      <Icon className="size-3" aria-hidden="true" />
      {usagePercentLabel(delta.percent, i18n.resolvedLanguage ?? "zh")}
    </span>
  )
}

function Kpi({
  label,
  value,
  hint,
  icon: Icon,
  action,
}: {
  label: string
  value: string
  hint?: string
  icon: ComponentType<{ className?: string }>
  action?: ReactNode
}) {
  return (
    <Card role="group" aria-label={label} className="min-w-0 gap-1.5 py-4">
      <CardHeader className="flex flex-row items-center justify-between gap-2 px-4">
        <span className="text-muted-foreground truncate text-xs">{label}</span>
        <Icon className="text-muted-foreground size-4 shrink-0" aria-hidden="true" />
      </CardHeader>
      <CardContent className="px-4">
        <div className="flex flex-wrap items-baseline gap-x-2 gap-y-1">
          <p className="font-mono text-2xl font-semibold tracking-tight wrap-anywhere tabular-nums">
            {value}
          </p>
          {action}
        </div>
        {hint ? <p className="text-muted-foreground mt-1 text-[11px] leading-4">{hint}</p> : null}
      </CardContent>
    </Card>
  )
}

export function UsageKpis({
  allTime,
  previous,
  recent,
  cache,
  reliability,
}: {
  allTime: Summary
  previous: Summary
  recent: Summary
  cache: CacheUsage
  reliability: Reliability
}) {
  const { t, i18n } = useTranslation()
  const language = i18n.resolvedLanguage ?? "zh"
  const average = usageRequestAverage(allTime.totalTokens, allTime.requests)
  const cached = BigInt(cache.cachedInputTokens)
  const input = BigInt(allTime.inputTokens)
  const cacheShare = usageShare(cached, input)
  const cacheKnown = input > 0n && cache.knownRequests > 0
  const cacheValue = !cacheKnown
    ? "—"
    : cache.knownRequests < allTime.requests
      ? `≥ ${usagePercentLabel(cacheShare, language)}`
      : usagePercentLabel(cacheShare, language)
  const success =
    reliability.successRate === null ? "—" : usagePercentLabel(reliability.successRate, language)
  return (
    <div className="grid min-w-0 gap-3 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-6">
      <Kpi
        label={t("activity.total")}
        value={formatTokens(allTime.totalTokens, language)}
        hint={t("activity.inputOutputSplit", {
          input: formatTokens(allTime.inputTokens, language),
          output: formatTokens(allTime.outputTokens, language),
        })}
        icon={Sigma}
      />
      <Kpi
        label={t("activity.requests")}
        value={usageCountLabel(allTime.requests, language)}
        hint={t("activity.coverage", {
          known: allTime.completeUsage,
          total: allTime.requests,
        })}
        icon={Waypoints}
      />
      <Kpi
        label={t("activity.kpiRecent")}
        value={formatTokens(recent.totalTokens, language)}
        hint={t("activity.requestCount", {
          value: usageCountLabel(recent.requests, language),
        })}
        icon={ChartPie}
        action={<DeltaBadge delta={usageDelta(recent.totalTokens, previous.totalTokens)} />}
      />
      <Kpi
        label={t("activity.kpiAverage")}
        value={average === null ? "—" : formatTokens(average, language)}
        hint={t("activity.kpiAverageHint")}
        icon={Gauge}
      />
      <Kpi
        label={t("activity.kpiCache")}
        value={cacheValue}
        hint={
          cacheKnown
            ? t("activity.cacheKnown", {
                count: usageCountLabel(cache.knownRequests, language),
              })
            : t("activity.cacheUnknown")
        }
        icon={Database}
      />
      <Kpi
        label={t("activity.kpiSuccess")}
        value={success}
        hint={
          reliability.requests > 0
            ? t("activity.successHint", {
                latency: chartDuration(reliability.averageLatencyMs, language),
                count: usageCountLabel(reliability.success + reliability.failed, language),
              })
            : t("activity.qualityEmpty")
        }
        icon={ShieldCheck}
      />
    </div>
  )
}
