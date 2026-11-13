import { useTranslation } from "react-i18next"
import { chartDuration } from "../../lib/chart"
import type { Reliability } from "../../services/activity"
import { Donut, UsageCard, UsageEmpty } from "./usage-charts"
import { usageCountLabel, usagePercentLabel } from "./usage-model"

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-lg border px-3 py-2">
      <dt className="text-muted-foreground truncate text-[11px]">{label}</dt>
      <dd className="mt-0.5 font-mono text-sm font-medium tabular-nums">{value}</dd>
    </div>
  )
}

export function UsageQuality({ reliability }: { reliability: Reliability }) {
  const { t, i18n } = useTranslation()
  const language = i18n.resolvedLanguage ?? "zh"
  const completed = reliability.success + reliability.failed
  if (completed <= 0 || reliability.successRate === null) {
    return (
      <UsageCard title={t("activity.quality")} hint={t("activity.qualityHint")}>
        <UsageEmpty>{t("activity.qualityEmpty")}</UsageEmpty>
      </UsageCard>
    )
  }
  return (
    <UsageCard title={t("activity.quality")} hint={t("activity.qualityHint")}>
      <div className="flex flex-wrap items-center gap-5">
        <Donut
          label={t("activity.successRate")}
          size={124}
          segments={[
            {
              key: "success",
              value: BigInt(reliability.success),
              className: "stroke-emerald-500",
            },
            { key: "failed", value: BigInt(reliability.failed), className: "stroke-rose-500" },
          ]}
          center={
            <>
              <span className="font-mono text-xl font-semibold tabular-nums">
                {usagePercentLabel(reliability.successRate, language)}
              </span>
              <span className="text-muted-foreground text-[10px]">{t("activity.successRate")}</span>
            </>
          }
        />
        <div className="min-w-0 flex-1 space-y-2 text-xs">
          <p className="flex items-center justify-between gap-3">
            <span className="text-muted-foreground inline-flex items-center gap-1.5">
              <span className="size-2 rounded-full bg-emerald-500" aria-hidden="true" />
              {t("activity.success")}
            </span>
            <span className="font-mono tabular-nums">
              {usageCountLabel(reliability.success, language)}
            </span>
          </p>
          <p className="flex items-center justify-between gap-3">
            <span className="text-muted-foreground inline-flex items-center gap-1.5">
              <span className="size-2 rounded-full bg-rose-500" aria-hidden="true" />
              {t("activity.failedState")}
            </span>
            <span className="font-mono tabular-nums">
              {usageCountLabel(reliability.failed, language)}
            </span>
          </p>
          <p className="flex items-center justify-between gap-3">
            <span className="text-muted-foreground inline-flex items-center gap-1.5">
              <span className="bg-muted-foreground/40 size-2 rounded-full" aria-hidden="true" />
              {t("activity.cancelled")}
            </span>
            <span className="font-mono tabular-nums">
              {usageCountLabel(reliability.cancelled, language)}
            </span>
          </p>
        </div>
      </div>
      <dl className="mt-4 grid grid-cols-2 gap-2">
        <Stat
          label={t("activity.avgLatency")}
          value={chartDuration(reliability.averageLatencyMs, language)}
        />
        <Stat
          label={t("activity.avgFirstToken")}
          value={chartDuration(reliability.averageFirstTokenMs, language)}
        />
      </dl>
    </UsageCard>
  )
}
