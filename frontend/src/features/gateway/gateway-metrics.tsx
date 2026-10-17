import { Activity, CheckCircle2, Clock3, Hash, Timer, Zap } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Card } from "../../components/ui/card"
import { Tooltip } from "../../components/ui/tooltip"
import { formatTokens } from "../../services/activity"
import type { DashboardInfo } from "../../services/dashboard"
import { formatDashboardLatency } from "../dashboard/dashboard-model"

export function GatewayMetrics({ data }: { data?: DashboardInfo }) {
  const { t, i18n } = useTranslation()
  const language = i18n.resolvedLanguage ?? "zh"
  const number = (value?: number | null) =>
    value == null
      ? "—"
      : new Intl.NumberFormat(language, { maximumFractionDigits: 1 }).format(value)
  const summary = data?.summary
  const tokensKnown =
    summary && (summary.requests === 0 || summary.completeUsage > 0 || summary.totalTokens !== "0")
  const metrics = [
    {
      key: "gateway.metrics.requests",
      value: number(summary?.requests),
      description: summary
        ? t("gateway.metrics.results", {
            failed: number(summary.failed),
            cancelled: number(summary.cancelled),
          })
        : t("gateway.loading"),
      Icon: Hash,
    },
    {
      key: "gateway.metrics.successRate",
      value: summary?.successRate == null ? "—" : `${number(summary.successRate)}%`,
      description: t("gateway.metrics.successHint"),
      Icon: CheckCircle2,
    },
    {
      key: "gateway.metrics.firstToken",
      value: formatDashboardLatency(summary?.averageFirstTokenMs ?? null, language),
      description: summary
        ? t("gateway.metrics.samples", { count: summary.firstTokenSamples ?? 0 })
        : t("gateway.loading"),
      hint: t("gateway.metrics.firstTokenHint"),
      Icon: Zap,
    },
    {
      key: "gateway.metrics.totalLatency",
      value: formatDashboardLatency(summary?.averageLatencyMs ?? null, language),
      description: t("gateway.metrics.latencyHint"),
      Icon: Timer,
    },
    {
      key: "gateway.metrics.pending",
      value: number(data?.pendingNow),
      description: t("gateway.metrics.pendingHint"),
      Icon: Activity,
    },
    {
      key: "gateway.metrics.tokens",
      value: summary && tokensKnown ? formatTokens(summary.totalTokens, language) : "—",
      description:
        summary && tokensKnown
          ? t("gateway.metrics.tokenParts", {
              input: formatTokens(summary.inputTokens, language),
              output: formatTokens(summary.outputTokens, language),
            })
          : t("activity.unknown"),
      Icon: Clock3,
    },
  ] as const
  return (
    <div className="grid min-w-0 gap-3 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-6">
      {metrics.map(({ key, value, description, Icon, ...metric }) => (
        <Card key={key} role="group" aria-label={t(key)} className="min-w-0 gap-3 p-4">
          <div className="text-muted-foreground flex items-center justify-between gap-2 text-xs">
            <Tooltip content={"hint" in metric ? metric.hint : undefined}>
              <span
                tabIndex={"hint" in metric ? 0 : undefined}
                className="focus-visible:ring-ring rounded-sm outline-none focus-visible:ring-2"
              >
                {t(key)}
              </span>
            </Tooltip>
            <Icon aria-hidden="true" className="size-4 shrink-0" />
          </div>
          <p className="text-2xl font-semibold tracking-tight wrap-anywhere tabular-nums">
            {value}
          </p>
          <p className="text-muted-foreground text-xs leading-5">{description}</p>
        </Card>
      ))}
    </div>
  )
}
