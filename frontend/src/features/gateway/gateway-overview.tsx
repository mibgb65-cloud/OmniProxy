import { useMutation } from "@tanstack/react-query"
import { RefreshCw } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { useDashboardStatistics } from "../dashboard/use-dashboard-statistics"
import { GatewayMetrics } from "./gateway-metrics"
import { GatewayRequestTrend } from "./gateway-request-trend"
import { GatewayRecentRequests } from "./gateway-recent-requests"
import { GatewayProviderDistribution } from "./gateway-provider-distribution"
import { GatewayConfigCard } from "./gateway-config-card"
import { GatewayFailoverCard } from "./gateway-failover-card"
import { GatewayRelatedLinks } from "./gateway-related-links"
import { withGatewayFeedback } from "./gateway-action-feedback"
import { requestTime } from "../activity/activity-format"
import type { GatewayPreparation } from "./use-gateway-preparation"

const periods = { 1: "dash.today", 7: "dash.week", 30: "dash.month" } as const

export function GatewayOverview({ data, active }: { data: GatewayPreparation; active: boolean }) {
  const { t } = useTranslation()
  const { statistics, period, setPeriod } = useDashboardStatistics(active)
  const refresh = useMutation({
    mutationFn: () =>
      withGatewayFeedback(() => statistics.refetch({ cancelRefetch: false, throwOnError: true })),
  })
  const info = statistics.isError ? undefined : statistics.data
  return (
    <div className="min-w-0 space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-muted-foreground text-xs">
          {info?.historyIncomplete
            ? t("dash.historyPartial")
            : info
              ? t("dash.updated", { time: requestTime(info.updatedAt) })
              : t("gateway.loading")}
        </p>
        <div className="flex flex-wrap items-center gap-2">
          <div
            role="group"
            aria-label={t("gateway.overview.range")}
            className="bg-muted flex max-w-full flex-wrap gap-1 rounded-lg p-1"
          >
            {([1, 7, 30] as const).map((value) => (
              <Button
                key={value}
                size="sm"
                variant={period === value ? "default" : "ghost"}
                aria-pressed={period === value}
                className={period === value ? "shadow-sm" : "text-muted-foreground"}
                onClick={() => setPeriod(value)}
              >
                {t(periods[value])}
              </Button>
            ))}
          </div>
          <Button
            size="sm"
            variant="outline"
            aria-busy={refresh.isPending}
            disabled={refresh.isPending || statistics.isPending}
            onClick={() => refresh.mutate()}
          >
            <RefreshCw
              aria-hidden="true"
              className={refresh.isPending ? "motion-safe:animate-spin" : ""}
            />
            {t("gateway.overview.refresh")}
          </Button>
        </div>
      </div>
      {statistics.isError ? (
        <p role="alert" className="text-destructive text-sm">
          {t("gateway.overview.failed")}
        </p>
      ) : null}
      <GatewayMetrics data={info} />
      <div className="grid min-w-0 items-start gap-4 xl:grid-cols-[minmax(0,1.8fr)_minmax(0,1fr)]">
        <div className="min-w-0 space-y-4">
          <GatewayRequestTrend data={info} period={period} loading={statistics.isPending} />
          <GatewayRecentRequests data={info} loading={statistics.isPending} />
        </div>
        <div className="min-w-0 space-y-4">
          <GatewayProviderDistribution data={info} loading={statistics.isPending} />
          <GatewayConfigCard data={data} />
          <GatewayFailoverCard active={active} running={data.gateway.isRunning} />
        </div>
      </div>
      <GatewayRelatedLinks />
    </div>
  )
}
