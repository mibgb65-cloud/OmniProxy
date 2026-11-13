import { useQuery } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { readUsage } from "../../services/activity"
import { ProviderModelUsageView } from "./provider-model-usage"
import { UsageComposition } from "./usage-composition"
import { UsageDailyChart } from "./usage-daily"
import { UsageHours } from "./usage-hours"
import { UsageKpis } from "./usage-kpis"
import { UsageModelTrend } from "./usage-model-trend"
import { UsageQuality } from "./usage-quality"
import { AccountShareCard, ModelShareCard, ProviderShareCard } from "./usage-share"
import { EMPTY_CACHE, EMPTY_RELIABILITY, EMPTY_SUMMARY, usageSum } from "./usage-model"

export function UsageView() {
  const { t } = useTranslation()
  const query = useQuery({ queryKey: ["usage"], queryFn: readUsage, refetchInterval: 5000 })
  const data = query.data
  const allTime = data?.allTime ?? EMPTY_SUMMARY
  const previous = data?.previous ?? EMPTY_SUMMARY
  const days = data?.days ?? []
  const accounts = data?.accounts ?? []
  const models = data?.models ?? []
  const providers = data?.providers ?? []
  const hours = data?.hours ?? []
  const modelDays = data?.modelDays ?? []
  const cache = data?.cache ?? EMPTY_CACHE
  const reliability = data?.reliability ?? EMPTY_RELIABILITY
  const recent = usageSum(days)
  return (
    <section className="w-full min-w-0 space-y-5 pb-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">{t("pages.usage.title")}</h1>
        <p className="text-muted-foreground max-w-4xl text-xs leading-5">
          {t("activity.usageNote")}
        </p>
      </header>
      {query.isError ? (
        <p role="alert" className="text-destructive text-sm">
          {t("activity.failed")}
        </p>
      ) : null}
      {query.isPending ? (
        <p role="status" className="text-muted-foreground text-sm">
          {t("activity.loading")}
        </p>
      ) : null}
      {data ? (
        <>
          <UsageKpis
            allTime={allTime}
            previous={previous}
            recent={recent}
            cache={cache}
            reliability={reliability}
          />
          <UsageDailyChart days={days} />
          <div className="grid min-w-0 gap-4 xl:grid-cols-2">
            <UsageComposition summary={allTime} />
            <ProviderShareCard
              items={providers}
              total={allTime}
              className="xl:col-start-1 xl:row-start-2"
            />
            <ModelShareCard
              items={models}
              total={allTime}
              className="xl:col-start-2 xl:row-span-2 xl:row-start-1"
            />
          </div>
          <UsageModelTrend modelDays={modelDays} days={days} />
          <div className="grid min-w-0 gap-4 xl:grid-cols-2">
            <UsageHours hours={hours} />
            <UsageQuality reliability={reliability} />
          </div>
          <AccountShareCard items={accounts} total={allTime} />
          <ProviderModelUsageView items={data.providerModels ?? []} />
        </>
      ) : null}
    </section>
  )
}
