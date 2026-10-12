import { Link } from "@tanstack/react-router"
import { ArrowUpRight, RefreshCw } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import { formatAccountTime } from "../accounts/account-display"
import type { DashboardResources as ResourceData } from "./use-dashboard-resources"

export function DashboardResources({ data, ready }: { data: ResourceData; ready: boolean }) {
  const { t } = useTranslation()
  const counts = [
    { label: "dash.resourceExhausted", value: data.summary.exhausted },
    { label: "dash.resourceErrors", value: data.summary.errors },
    { label: "dash.resourceUncertain", value: data.summary.uncertain },
  ] as const
  return (
    <Card className="min-w-0 gap-3 py-4">
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
      <CardContent className="space-y-4 px-4">
        <dl className="grid grid-cols-3 gap-2">
          {counts.map((item) => (
            <div key={item.label} className="min-w-0 space-y-1.5">
              <dt className="text-muted-foreground text-xs leading-5 wrap-anywhere">
                {t(item.label)}
              </dt>
              <dd className="font-mono text-2xl font-semibold tabular-nums">
                {ready ? (item.value ?? "—") : "—"}
              </dd>
            </div>
          ))}
        </dl>
        <div className="text-muted-foreground space-y-1 text-xs leading-5">
          <p>
            {ready
              ? data.checked
                ? t("dash.resourceCache", { checked: data.checked, total: data.summary.monitored })
                : t("dash.resourceUnchecked")
              : t("accounts.loading")}
          </p>
          {data.lastChecked ? (
            <p>{t("dash.resourceLast", { time: formatAccountTime(data.lastChecked) })}</p>
          ) : null}
        </div>
        {data.result ? (
          <p role="status" className="text-muted-foreground text-xs leading-5">
            {t("dash.resourceChecked", data.result)}
          </p>
        ) : null}
        <Link
          to="/quota"
          className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1 text-sm"
        >
          {t("dash.allQuota")}
          <ArrowUpRight className="size-3.5" aria-hidden="true" />
        </Link>
      </CardContent>
    </Card>
  )
}
