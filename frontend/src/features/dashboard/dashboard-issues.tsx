import { Link } from "@tanstack/react-router"
import { AlertCircle, ArrowUpRight, CircleCheck } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import type { DashboardData } from "./use-dashboard"

export function DashboardIssues({ data }: { data: DashboardData }) {
  const { t } = useTranslation()
  const visible = data.issues.slice(0, 5)
  return (
    <Card className="min-w-0 gap-3 py-4">
      <CardHeader className="flex flex-row items-center justify-between gap-2 px-4">
        <h2 className="text-base font-semibold">{t("dash.issues")}</h2>
        <span className="text-muted-foreground text-xs">
          {t("dash.issuesCount", { count: data.issues.length })}
        </span>
      </CardHeader>
      <CardContent className="space-y-3 px-4">
        {data.metadataError ? (
          <p role="alert" className="text-destructive text-xs leading-5">
            {t("dash.metadataFailed")}
          </p>
        ) : data.metadataPending ? (
          <p className="text-muted-foreground text-xs">{t("accounts.loading")}</p>
        ) : visible.length === 0 ? (
          <div className="text-muted-foreground flex items-start gap-2 rounded-lg border border-dashed p-4 text-sm">
            <CircleCheck className="size-4 shrink-0" aria-hidden="true" />
            <span>{t("dash.issuesEmpty")}</span>
          </div>
        ) : null}
        {visible.map((issue) => (
          <div key={issue.key} className="flex min-w-0 items-start gap-2 rounded-lg border p-3">
            <AlertCircle
              className="mt-0.5 size-4 shrink-0 text-amber-600 dark:text-amber-400"
              aria-hidden="true"
            />
            <div className="min-w-0 flex-1 space-y-1">
              <p className="text-sm leading-5">
                {t(`dash.issue.${issue.kind}`, { count: issue.count })}
              </p>
              {issue.name ? (
                <p className="text-muted-foreground truncate text-xs">{issue.name}</p>
              ) : null}
              {issue.kind === "account" && issue.providerId ? (
                <Link
                  to="/accounts/$providerId"
                  params={{ providerId: issue.providerId }}
                  className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1 text-xs"
                >
                  {t("quota.details")}
                  <ArrowUpRight className="size-3" aria-hidden="true" />
                </Link>
              ) : (
                <Link
                  to={issue.kind === "route" ? "/routing" : "/quota"}
                  className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1 text-xs"
                >
                  {t(issue.kind === "route" ? "nav.routing" : "dash.allQuota")}
                  <ArrowUpRight className="size-3" aria-hidden="true" />
                </Link>
              )}
            </div>
          </div>
        ))}
        {data.issues.length > visible.length ? (
          <Link to="/accounts" className="text-muted-foreground hover:text-foreground text-xs">
            {t("dash.issueMore")}
          </Link>
        ) : null}
        {data.resources.checked < data.resources.summary.monitored ? (
          <p className="text-muted-foreground text-xs leading-5">{t("dash.resourcesUnchecked")}</p>
        ) : null}
      </CardContent>
    </Card>
  )
}
