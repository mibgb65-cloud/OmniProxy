import { Link } from "@tanstack/react-router"
import { ArrowUpRight } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import { Tooltip } from "../../components/ui/tooltip"
import { stateLabel, requestTime } from "../activity/activity-format"
import type { RequestItem } from "../../services/activity"
import { formatDashboardLatency } from "./dashboard-model"

export function DashboardRecent({ items, loading }: { items: RequestItem[]; loading: boolean }) {
  const { t, i18n } = useTranslation()
  return (
    <Card className="min-w-0 gap-3 py-4">
      <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-2 px-4">
        <h2 className="text-base font-semibold">{t("dash.recent")}</h2>
        <Link
          to="/requests"
          search={{ requestId: undefined }}
          className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1 text-xs"
        >
          {t("dash.allRequests")}
          <ArrowUpRight className="size-3" aria-hidden="true" />
        </Link>
      </CardHeader>
      <CardContent className="px-4">
        {items.length ? (
          <ul className="divide-border divide-y">
            {items.map((item) => (
              <li key={item.id}>
                <Link
                  to="/requests"
                  search={{ requestId: item.id }}
                  aria-label={t("dash.requestDetails", { id: item.id })}
                  className="hover:bg-muted/40 focus-visible:ring-ring/50 flex min-w-0 items-center gap-3 rounded-lg py-2.5 outline-none focus-visible:ring-2"
                >
                  <Tooltip content={requestTime(item.startedAt)}>
                    <time
                      dateTime={item.startedAt}
                      className="text-muted-foreground w-12 shrink-0 text-xs tabular-nums"
                    >
                      {new Intl.DateTimeFormat(i18n.resolvedLanguage, {
                        hour: "2-digit",
                        minute: "2-digit",
                        hourCycle: "h23",
                      }).format(new Date(item.startedAt))}
                    </time>
                  </Tooltip>
                  <div className="min-w-0 flex-1 space-y-0.5">
                    <p className="truncate text-sm font-medium">
                      {item.model || item.inputModel || "—"}
                    </p>
                    <p className="text-muted-foreground truncate text-xs">
                      {item.providerName || "—"}
                    </p>
                  </div>
                  <div className="flex max-w-28 shrink-0 flex-col items-end gap-1 text-xs">
                    <span
                      className={`max-w-full rounded-md px-2 py-0.5 ${item.state === "success" ? "bg-emerald-50 text-emerald-800 dark:bg-emerald-950 dark:text-emerald-300" : item.state === "failed" ? "bg-rose-50 text-rose-800 dark:bg-rose-950 dark:text-rose-300" : "bg-muted text-muted-foreground"}`}
                    >
                      {t(stateLabel(item.state))}
                    </span>
                    <span className="text-muted-foreground tabular-nums">
                      {item.state === "pending"
                        ? "—"
                        : formatDashboardLatency(item.latencyMs, i18n.resolvedLanguage ?? "zh")}
                    </span>
                  </div>
                </Link>
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-muted-foreground py-8 text-center text-sm">
            {t(loading ? "dash.loading" : "dash.recentEmpty")}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
