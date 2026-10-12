import { useQuery } from "@tanstack/react-query"
import { ArrowDownLeft, ArrowUpRight, RefreshCw, Sigma } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import { formatTokens, readUsage } from "../../services/activity"
import { UsageChart, UsageBreakdown } from "./usage-breakdown"
import { ProviderModelUsageView } from "./provider-model-usage"

export function UsageView() {
  const { t, i18n } = useTranslation()
  const query = useQuery({ queryKey: ["usage"], queryFn: readUsage, refetchInterval: 5000 })
  const summary = query.data?.allTime
  const metrics = [
    { key: "activity.total", value: summary?.totalTokens, icon: Sigma },
    { key: "activity.input", value: summary?.inputTokens, icon: ArrowDownLeft },
    { key: "activity.output", value: summary?.outputTokens, icon: ArrowUpRight },
  ] as const
  return (
    <section className="w-full min-w-0 space-y-5 pb-6">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-semibold tracking-tight">{t("pages.usage.title")}</h1>
        <Button variant="outline" disabled={query.isFetching} onClick={() => void query.refetch()}>
          <RefreshCw
            className={query.isFetching ? "motion-safe:animate-spin" : ""}
            aria-hidden="true"
          />
          {t("activity.refresh")}
        </Button>
      </header>
      {query.isError ? (
        <p role="alert" className="text-destructive text-sm">
          {t("activity.failed")}
        </p>
      ) : null}
      <div className="grid gap-4 lg:grid-cols-3">
        {metrics.map(({ key, value, icon: Icon }) => (
          <Card key={key} className="min-w-0 gap-2">
            <CardHeader className="text-muted-foreground flex flex-row items-center justify-between text-sm">
              <span>{t(key)}</span>
              <Icon className="size-4" aria-hidden="true" />
            </CardHeader>
            <CardContent>
              <p className="font-mono text-3xl font-semibold wrap-anywhere tabular-nums">
                {value === undefined ? "—" : formatTokens(value, i18n.resolvedLanguage)}
              </p>
            </CardContent>
          </Card>
        ))}
      </div>
      <div className="text-muted-foreground flex flex-wrap justify-between gap-2 text-xs">
        <span>
          {t("activity.requests")}: {summary?.requests ?? "—"}
        </span>
        <span>
          {t("activity.coverage", {
            known: summary?.completeUsage ?? 0,
            total: summary?.requests ?? 0,
          })}
        </span>
      </div>
      <p className="text-muted-foreground text-xs leading-5">{t("activity.usageNote")}</p>
      {query.isPending ? (
        <p role="status" className="text-muted-foreground text-sm">
          {t("activity.loading")}
        </p>
      ) : null}
      {query.data ? (
        <>
          <UsageChart days={query.data.days ?? []} />
          <ProviderModelUsageView items={query.data.providerModels ?? []} />
          <div className="grid min-w-0 gap-5 xl:grid-cols-2">
            <UsageBreakdown title="activity.accounts" items={query.data.accounts ?? []} members />
            <UsageBreakdown title="activity.models" items={query.data.models ?? []} />
          </div>
        </>
      ) : null}
    </section>
  )
}
