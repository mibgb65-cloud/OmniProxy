import { useTranslation } from "react-i18next"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import { Tooltip } from "../../components/ui/tooltip"
import { formatTokens, type Breakdown } from "../../services/activity"

export function UsageChart({ days }: { days: Breakdown[] }) {
  const { t, i18n } = useTranslation()
  const maximum = days.reduce(
    (max, day) => (BigInt(day.summary.totalTokens) > max ? BigInt(day.summary.totalTokens) : max),
    1n,
  )
  return (
    <Card className="min-w-0 gap-4">
      <CardHeader>
        <h2 className="text-base font-semibold">{t("activity.days")}</h2>
        <div className="text-muted-foreground flex gap-4 text-xs">
          <span className="flex items-center gap-2">
            <span className="size-2 rounded bg-blue-500" />
            {t("activity.input")}
          </span>
          <span className="flex items-center gap-2">
            <span className="size-2 rounded bg-emerald-500" />
            {t("activity.output")}
          </span>
        </div>
      </CardHeader>
      <CardContent>
        {days.length ? (
          <div
            className="flex h-44 min-w-0 items-end gap-1.5"
            role="img"
            aria-label={t("activity.days")}
          >
            {days.map((day) => (
              <Tooltip
                key={day.key}
                content={`${day.key} · ${t("activity.input")}: ${formatTokens(day.summary.inputTokens, i18n.resolvedLanguage)} · ${t("activity.output")}: ${formatTokens(day.summary.outputTokens, i18n.resolvedLanguage)}`}
              >
                <div
                  tabIndex={0}
                  className="group flex h-full min-w-0 flex-1 flex-col items-center justify-end gap-2"
                >
                  <div
                    className="flex w-full max-w-12 flex-col justify-end overflow-hidden rounded-t-sm"
                    style={{
                      height: `${Math.max(1, Number((BigInt(day.summary.totalTokens) * 10000n) / maximum) / 100)}%`,
                    }}
                  >
                    <div
                      className="min-h-0 bg-emerald-500"
                      style={{ flexGrow: Number(day.summary.outputTokens) }}
                    />
                    <div
                      className="min-h-0 bg-blue-500"
                      style={{ flexGrow: Number(day.summary.inputTokens) }}
                    />
                  </div>
                  <span className="text-muted-foreground text-[10px]">{day.key.slice(5)}</span>
                </div>
              </Tooltip>
            ))}
          </div>
        ) : (
          <p className="text-muted-foreground py-8 text-center text-sm">{t("activity.empty")}</p>
        )}
      </CardContent>
    </Card>
  )
}
export function UsageBreakdown({
  title,
  items,
  members = false,
}: {
  title: "activity.accounts" | "activity.models"
  items: Breakdown[]
  members?: boolean
}) {
  const { t, i18n } = useTranslation()
  return (
    <Card className="min-w-0 gap-3">
      <CardHeader>
        <h2 className="text-base font-semibold">{t(title)}</h2>
      </CardHeader>
      <CardContent>
        {items.length ? (
          <div className="app-scroll max-h-96 space-y-3 overflow-auto">
            {items.map((item) => (
              <div
                key={item.key}
                className="flex min-w-0 items-center justify-between gap-3 rounded-lg border p-3"
              >
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium">
                    {item.email ||
                      item.name ||
                      t(members ? "activity.noAccount" : "activity.noModel")}
                  </p>
                  {item.workspaceId ? (
                    <Tooltip content={item.workspaceId}>
                      <span className="text-muted-foreground font-mono text-xs">
                        {item.workspaceId.slice(0, 8)}…{item.workspaceId.slice(-6)}
                      </span>
                    </Tooltip>
                  ) : null}
                  <p className="text-muted-foreground mt-1 text-xs">
                    {t("activity.input")}:{" "}
                    {formatTokens(item.summary.inputTokens, i18n.resolvedLanguage)} ·{" "}
                    {t("activity.output")}:{" "}
                    {formatTokens(item.summary.outputTokens, i18n.resolvedLanguage)}
                  </p>
                </div>
                <span className="shrink-0 font-mono text-sm tabular-nums">
                  {formatTokens(item.summary.totalTokens, i18n.resolvedLanguage)}
                </span>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-muted-foreground py-4 text-sm">{t("activity.empty")}</p>
        )}
      </CardContent>
    </Card>
  )
}
