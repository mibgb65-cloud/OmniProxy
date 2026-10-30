import { Link } from "@tanstack/react-router"
import { ArrowUpRight, RefreshCw } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import { formatAccountTime } from "../accounts/account-display"
import { groupQuotaEntries, quotaSummary, type QuotaStatus } from "../quota/quota-model"
import type { DashboardResources as ResourceData } from "./use-dashboard-resources"

/** 每行最多渲染的状态点；其余账号只在计数中体现，避免点阵挤成实心圆。 */
const MAX_DOTS = 8
const MAX_PROVIDER_ROWS = 5

const statusDot: Record<QuotaStatus, string> = {
  normal: "bg-emerald-500",
  exhausted: "bg-amber-500",
  error: "bg-rose-500",
  stale: "bg-zinc-400",
  unauthenticated: "bg-zinc-400",
  unsupported: "bg-zinc-600",
  pending: "bg-blue-500",
  unknown: "bg-zinc-500",
}

export function DashboardResources({ data, ready }: { data: ResourceData; ready: boolean }) {
  const { t } = useTranslation()
  const counts = [
    { label: "dash.resourceExhausted", value: data.summary.exhausted },
    { label: "dash.resourceErrors", value: data.summary.errors },
    { label: "dash.resourceUncertain", value: data.summary.uncertain },
  ] as const
  const normal = Math.max(0, data.summary.known - (data.summary.exhausted ?? 0))
  // 健康分布：把三个孤立数字换成正/异常的比例条，宽度用满卡片才能抵住超宽窗口。
  const segments = [
    { key: "dash.resourceNormal", value: normal, color: "bg-emerald-500" },
    { key: "dash.resourceExhausted", value: data.summary.exhausted ?? 0, color: "bg-amber-500" },
    { key: "dash.resourceErrors", value: data.summary.errors, color: "bg-rose-500" },
    { key: "dash.resourceUncertain", value: data.summary.uncertain, color: "bg-zinc-400" },
  ] as const
  const tracked = segments.reduce((sum, item) => sum + item.value, 0)
  const providers = groupQuotaEntries(
    // 不支持查询的类型没有额度语义，保留会让点阵与「正常 X / Y」的分母打架。
    data.entries.filter((entry) => entry.kind !== "unsupported"),
  )
    .map((group) => {
      const summary = quotaSummary(group.entries)
      return {
        id: group.id,
        name: group.provider?.name ?? group.id,
        entries: group.entries,
        healthy: summary.known - (summary.exhausted ?? 0),
        problems: (summary.exhausted ?? 0) + summary.errors,
      }
    })
    .sort(
      (left, right) => right.problems - left.problems || right.entries.length - left.entries.length,
    )
    .slice(0, MAX_PROVIDER_ROWS)
  return (
    // 与左列「最近请求」同高：数字在上、分布与供应商居中、缓存与入口贴底。
    <Card className="flex h-full min-w-0 flex-col gap-3 py-4">
      <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-2 px-4">
        <h2 className="text-base font-semibold">{t("dash.resources")}</h2>
        <Button
          variant="outline"
          size="sm"
          disabled={!ready || !data.canCheck}
          onClick={() => void data.check()}
        >
          <RefreshCw
            className={data.busy ? "size-3.5 animate-spin motion-reduce:animate-none" : "size-3.5"}
            aria-hidden="true"
          />
          {t(data.busy ? "dash.checking" : "dash.checkResources")}
        </Button>
      </CardHeader>
      <CardContent className="flex min-h-0 flex-1 flex-col gap-4 px-4">
        <dl className="grid grid-cols-3 gap-2">
          {counts.map((item) => (
            <div key={item.label} className="min-w-0 space-y-1.5">
              <dt className="text-muted-foreground text-xs leading-5 wrap-anywhere">
                {t(item.label)}
              </dt>
              <dd className="font-mono text-3xl font-semibold tabular-nums 2xl:text-4xl">
                {ready ? (item.value ?? "—") : "—"}
              </dd>
            </div>
          ))}
        </dl>
        {ready ? (
          <div className="space-y-2">
            <div className="flex items-center justify-between gap-2 text-xs">
              <span className="text-muted-foreground">{t("dash.resourceHealth")}</span>
              <span className="tabular-nums">{tracked > 0 ? `${normal} / ${tracked}` : "—"}</span>
            </div>
            <div
              role="img"
              aria-label={t("dash.resourceHealthLabel", { normal, total: tracked })}
              className="bg-muted flex h-2 w-full overflow-hidden rounded-full"
            >
              {segments.map((item) =>
                item.value > 0 ? (
                  <span
                    key={item.key}
                    className={item.color}
                    style={{ width: `${(item.value / Math.max(tracked, 1)) * 100}%` }}
                  />
                ) : null,
              )}
            </div>
            <ul className="text-muted-foreground flex flex-wrap gap-x-4 gap-y-1 text-[11px] leading-4">
              {segments.map((item) => (
                <li key={item.key} className="inline-flex items-center gap-1.5">
                  <span className={`size-1.5 rounded-full ${item.color}`} aria-hidden="true" />
                  {t(item.key)}
                  <span className="text-foreground/70 tabular-nums">{item.value}</span>
                </li>
              ))}
            </ul>
          </div>
        ) : null}
        {providers.length > 0 ? (
          <div className="flex min-h-0 flex-1 flex-col gap-2">
            <p className="text-muted-foreground text-xs">{t("dash.resourceByProvider")}</p>
            <ul className="divide-border flex flex-1 flex-col divide-y">
              {providers.map((provider) => (
                <li key={provider.id} className="flex items-center gap-2 py-2.5">
                  <span className="min-w-0 flex-1 truncate text-xs" title={provider.name}>
                    {provider.name}
                  </span>
                  <span aria-hidden="true" className="flex shrink-0 items-center gap-1">
                    {provider.entries.slice(0, MAX_DOTS).map((entry) => (
                      <span
                        key={entry.account.id}
                        className={`size-1.5 rounded-full ${statusDot[entry.status]}`}
                      />
                    ))}
                  </span>
                  <span className="text-muted-foreground w-6 shrink-0 text-end text-xs tabular-nums">
                    {provider.entries.length}
                  </span>
                  <span className="sr-only">
                    {t("dash.resourceProviderSummary", {
                      provider: provider.name,
                      normal: provider.healthy,
                      total: provider.entries.length,
                    })}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        ) : null}
        <div className="mt-auto space-y-1 border-t pt-3 text-xs leading-5">
          <p className="text-muted-foreground">
            {ready
              ? data.checked
                ? t("dash.resourceCache", { checked: data.checked, total: data.summary.monitored })
                : t("dash.resourceUnchecked")
              : t("accounts.loading")}
          </p>
          {data.lastChecked ? (
            <p className="text-muted-foreground">
              {t("dash.resourceLast", { time: formatAccountTime(data.lastChecked) })}
            </p>
          ) : null}
          {data.result ? (
            <p role="status" className="text-muted-foreground">
              {t("dash.resourceChecked", data.result)}
            </p>
          ) : null}
          <Link
            to="/quota"
            className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1"
          >
            {t("dash.allQuota")}
            <ArrowUpRight className="size-3.5" aria-hidden="true" />
          </Link>
        </div>
      </CardContent>
    </Card>
  )
}
