import { Link } from "@tanstack/react-router"
import { ArrowUpRight } from "lucide-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import { Input } from "../../components/ui/input"
import { SelectField } from "../../components/ui/select-field"
import { Tooltip } from "../../components/ui/tooltip"
import { formatTokens } from "../../services/activity"
import type { DashboardInfo } from "../../services/dashboard"
import { stateLabel, requestTime } from "../activity/activity-format"
import { formatDashboardLatency } from "../dashboard/dashboard-model"
import { filterGatewayRecent } from "./gateway-overview-model"

export function GatewayRecentRequests({
  data,
  loading,
}: {
  data?: DashboardInfo
  loading: boolean
}) {
  const { t, i18n } = useTranslation()
  const language = i18n.resolvedLanguage ?? "zh"
  const [search, setSearch] = useState("")
  const [state, setState] = useState("")
  const items = data?.recent ?? []
  const filtered = filterGatewayRecent(items, search, state)
  return (
    <Card className="min-w-0 gap-4 py-4">
      <CardHeader className="flex flex-row flex-wrap items-start justify-between gap-3 px-4">
        <div className="space-y-2">
          <h2 className="text-base font-semibold">{t("gateway.recent.title")}</h2>
          <p className="text-muted-foreground text-xs">{t("gateway.recent.hint")}</p>
        </div>
        <div className="flex max-w-full flex-wrap gap-2">
          <Input
            aria-label={t("gateway.recent.search")}
            placeholder={t("gateway.recent.placeholder")}
            maxLength={128}
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            className="h-8 w-48 min-w-0 text-xs"
          />
          <SelectField
            aria-label={t("activity.state")}
            value={state}
            onValueChange={setState}
            className="h-8 w-auto min-w-28 text-xs"
            options={["", "success", "failed", "cancelled", "pending"].map((value) => ({
              value,
              label: t(value ? stateLabel(value) : "activity.all"),
            }))}
          />
        </div>
      </CardHeader>
      <CardContent className="space-y-3 px-4">
        <div className="min-w-0 overflow-x-auto">
          <table className="w-full min-w-[660px] text-left text-xs">
            <thead className="text-muted-foreground bg-muted/25">
              <tr>
                {(
                  [
                    "activity.time",
                    "activity.model",
                    "gateway.recent.tokens",
                    "gateway.recent.firstToken",
                    "gateway.recent.totalLatency",
                    "activity.state",
                  ] as const
                ).map((key) => (
                  <th key={key} scope="col" className="px-3 py-2.5 font-normal">
                    {t(key)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y">
              {filtered.map((item) => (
                <tr key={item.id} className="hover:bg-muted/20">
                  <td className="px-3 py-3 tabular-nums">
                    <Tooltip content={requestTime(item.startedAt)}>
                      <time dateTime={item.startedAt}>{requestTime(item.startedAt).slice(-8)}</time>
                    </Tooltip>
                  </td>
                  <td className="max-w-48 px-3 py-3">
                    <Link
                      to="/requests"
                      search={{ requestId: item.id }}
                      aria-label={t("gateway.recent.open", { id: item.id })}
                      className="block truncate font-medium hover:underline"
                      title={item.model || item.inputModel}
                    >
                      {item.model || item.inputModel || "—"}
                    </Link>
                    <p className="text-muted-foreground mt-1 truncate" title={item.providerName}>
                      {item.providerName || "—"}
                    </p>
                  </td>
                  <td className="px-3 py-3 whitespace-nowrap tabular-nums">
                    {item.inputKnown ? formatTokens(item.inputTokens, language) : "—"} /{" "}
                    {item.outputKnown ? formatTokens(item.outputTokens, language) : "—"}
                  </td>
                  <td className="px-3 py-3 whitespace-nowrap tabular-nums">
                    {formatDashboardLatency(item.firstTokenMs ?? null, language)}
                  </td>
                  <td className="px-3 py-3 whitespace-nowrap tabular-nums">
                    {item.state === "pending"
                      ? "—"
                      : formatDashboardLatency(item.latencyMs, language)}
                  </td>
                  <td className="px-3 py-3">
                    <span
                      className={`inline-flex rounded-md px-2 py-1 ${item.state === "success" ? "bg-emerald-500/10 text-emerald-700 dark:text-emerald-300" : item.state === "failed" ? "bg-rose-500/10 text-rose-700 dark:text-rose-300" : "bg-muted text-muted-foreground"}`}
                    >
                      {t(stateLabel(item.state))}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {!filtered.length ? (
          <p className="text-muted-foreground py-6 text-center text-sm">
            {t(
              loading
                ? "gateway.loading"
                : data
                  ? "gateway.recent.empty"
                  : "gateway.overview.unavailable",
            )}
          </p>
        ) : null}
        <div className="text-muted-foreground flex flex-wrap items-center justify-between gap-2 border-t pt-3 text-xs">
          <span>
            {data
              ? t("gateway.recent.count", { count: filtered.length, total: items.length })
              : "—"}
          </span>
          <Link
            to="/requests"
            search={{ requestId: undefined }}
            className="text-primary inline-flex items-center gap-1 hover:underline"
          >
            {t("gateway.recent.all")}
            <ArrowUpRight aria-hidden="true" className="size-3" />
          </Link>
        </div>
      </CardContent>
    </Card>
  )
}
