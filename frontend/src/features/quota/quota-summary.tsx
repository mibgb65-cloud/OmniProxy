import { AlertTriangle, CircleHelp, Gauge, Layers } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import type { QuotaOverview } from "./use-quota-overview"

export function QuotaSummary({ data }: { data: QuotaOverview }) {
  const { t } = useTranslation()
  const metrics = [
    { key: "quota.monitored", value: data.summary.monitored, icon: Layers },
    { key: "quota.exhausted", value: data.summary.exhausted, icon: Gauge },
    { key: "quota.errors", value: data.summary.errors, icon: AlertTriangle },
    { key: "quota.uncertain", value: data.summary.uncertain, icon: CircleHelp },
  ] as const
  return (
    <div className="space-y-3">
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {metrics.map(({ key, value, icon: Icon }) => (
          <Card key={key} className="min-w-0 gap-3 py-4">
            <CardHeader className="text-muted-foreground flex flex-row items-center justify-between gap-2 px-5 text-sm">
              <span>{t(key)}</span>
              <Icon className="size-4 shrink-0" aria-hidden="true" />
            </CardHeader>
            <CardContent className="px-5">
              <p className="font-mono text-3xl font-semibold tabular-nums">
                {data.ready ? (value ?? "—") : "—"}
              </p>
            </CardContent>
          </Card>
        ))}
      </div>
      <p className="text-muted-foreground text-xs leading-5">{t("quota.summaryNote")}</p>
    </div>
  )
}
