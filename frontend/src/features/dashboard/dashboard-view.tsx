import { RefreshCw } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { formatAccountTime } from "../accounts/account-display"
import { DashboardClients } from "./dashboard-clients"
import { DashboardIssues } from "./dashboard-issues"
import { DashboardMetrics } from "./dashboard-metrics"
import { DashboardRecent } from "./dashboard-recent"
import { DashboardResources } from "./dashboard-resources"
import { DashboardTrend } from "./dashboard-trend"
import { useDashboard } from "./use-dashboard"

const periodLabels = { 1: "dash.today", 7: "dash.week", 30: "dash.month" } as const

export function DashboardView() {
  const { t } = useTranslation()
  const data = useDashboard()
  const statistics = data.statistics.data
  return (
    <section className="w-full min-w-0 space-y-5 pb-6">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div className="space-y-1.5">
          <h1 className="text-2xl font-semibold tracking-tight">{t("dashboard.title")}</h1>
          <p className="text-muted-foreground text-sm">{t("dash.description")}</p>
        </div>
        <div className="flex max-w-full flex-wrap items-center gap-2">
          <div
            role="group"
            aria-label={t("dash.range")}
            className="bg-muted flex max-w-full flex-wrap rounded-lg p-1"
          >
            {([1, 7, 30] as const).map((period) => (
              <Button
                key={period}
                variant={data.period === period ? "secondary" : "ghost"}
                size="sm"
                aria-pressed={data.period === period}
                className={
                  data.period === period
                    ? "bg-background text-foreground shadow-xs"
                    : "text-muted-foreground"
                }
                onClick={() => data.setPeriod(period)}
              >
                {t(periodLabels[period])}
              </Button>
            ))}
          </div>
          <Button variant="outline" disabled={data.refreshing} onClick={() => void data.refresh()}>
            <RefreshCw
              className={
                data.refreshing ? "size-4 animate-spin motion-reduce:animate-none" : "size-4"
              }
              aria-hidden="true"
            />
            {t("dash.refresh")}
          </Button>
        </div>
      </header>
      {data.statistics.isError ? (
        <p role="alert" className="text-destructive text-sm">
          {t("dash.failed")}
        </p>
      ) : null}
      {statistics?.historyIncomplete ? (
        <p className="text-muted-foreground bg-muted/40 rounded-lg border px-3 py-2 text-xs leading-5">
          {t("dash.historyPartial")}
        </p>
      ) : null}
      <DashboardMetrics summary={statistics?.summary} incomplete={statistics?.historyIncomplete} />
      {statistics ? (
        <div className="text-muted-foreground flex flex-wrap justify-between gap-2 text-xs">
          <span>
            {t("dash.coverage", {
              known: statistics.summary.completeUsage,
              total: statistics.summary.requests,
            })}
          </span>
          <span>{t("dash.updated", { time: formatAccountTime(statistics.updatedAt) })}</span>
        </div>
      ) : null}
      <div className="grid min-w-0 items-start gap-5 xl:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
        <DashboardTrend
          data={statistics}
          period={data.period}
          loading={data.statistics.isPending}
        />
        <DashboardIssues data={data} />
      </div>
      <div className="grid min-w-0 items-start gap-5 xl:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
        <DashboardRecent items={statistics?.recent ?? []} loading={data.statistics.isPending} />
        <DashboardResources
          data={data.resources}
          ready={data.accounts.isSuccess && data.providers.isSuccess}
        />
      </div>
      <DashboardClients
        items={data.agents.data ?? []}
        failed={data.agents.isError}
        loading={data.agents.isPending}
      />
    </section>
  )
}
