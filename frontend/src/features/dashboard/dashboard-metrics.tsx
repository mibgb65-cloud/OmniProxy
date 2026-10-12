import { Clock3, Coins, MessageSquare, ShieldCheck } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Card, CardContent } from "../../components/ui/card"
import { formatTokens } from "../../services/activity"
import type { DashboardSummary } from "../../services/dashboard"
import { formatDashboardLatency } from "./dashboard-model"

export function DashboardMetrics({
  summary,
  incomplete,
}: {
  summary?: DashboardSummary
  incomplete?: boolean
}) {
  const { t, i18n } = useTranslation()
  const language = i18n.resolvedLanguage ?? "zh"
  const known = summary && !(incomplete && summary.requests === 0)
  const tokensUnknown =
    summary && summary.requests > 0 && summary.completeUsage === 0 && summary.totalTokens === "0"
  const metrics = [
    {
      key: "dash.requests",
      icon: MessageSquare,
      value: known ? new Intl.NumberFormat(language).format(summary.requests) : "—",
      note: summary
        ? t("dash.inFlight", { pending: summary.pending, cancelled: summary.cancelled })
        : "—",
    },
    {
      key: "dash.tokens",
      icon: Coins,
      value: known && !tokensUnknown ? formatTokens(summary.totalTokens, language) : "—",
      note: tokensUnknown
        ? t("dash.trendUsageUnknown")
        : summary
          ? t("dash.tokenParts", {
              input: formatTokens(summary.inputTokens, language),
              output: formatTokens(summary.outputTokens, language),
            })
          : "—",
    },
    {
      key: "dash.successRate",
      icon: ShieldCheck,
      value:
        summary?.successRate !== null && summary?.successRate !== undefined
          ? `${new Intl.NumberFormat(language, { maximumFractionDigits: 1 }).format(summary.successRate)}%`
          : "—",
      note: t("dash.rateNote"),
    },
    {
      key: "dash.latency",
      icon: Clock3,
      value: formatDashboardLatency(summary?.averageLatencyMs, language),
      note: t("dash.latencyNote"),
    },
  ] as const
  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
      {metrics.map(({ key, icon: Icon, value, note }) => (
        <Card key={key} className="min-w-0 gap-0 py-0">
          <CardContent className="flex h-full min-h-32 flex-col gap-3 p-4">
            <div className="text-muted-foreground flex items-center justify-between gap-2 text-sm">
              <span>{t(key)}</span>
              <Icon className="size-4 shrink-0" aria-hidden="true" />
            </div>
            <p className="font-mono text-3xl font-semibold wrap-anywhere tabular-nums">{value}</p>
            <p className="text-muted-foreground mt-auto text-xs leading-5">{note}</p>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
