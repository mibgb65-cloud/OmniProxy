import { Link } from "@tanstack/react-router"
import { AlertCircle, ArrowUpRight, CircleCheck } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import type { DashboardIssue } from "./dashboard-model"
import type { DashboardData } from "./use-dashboard"

const issueGroupKeys = {
  account: "dash.issuesGroup.account",
  resource: "dash.issuesGroup.resource",
  route: "dash.issuesGroup.route",
} as const

const issueKinds: DashboardIssue["kind"][] = ["account", "resource", "route"]

function problemBadge(problems: number) {
  return problems > 0
    ? "shrink-0 rounded-md bg-amber-50 px-2 py-0.5 text-xs text-amber-800 dark:bg-amber-950 dark:text-amber-300"
    : "bg-muted text-muted-foreground shrink-0 rounded-md px-2 py-0.5 text-xs"
}

export function DashboardIssues({ data }: { data: DashboardData }) {
  const { t } = useTranslation()
  const visible = data.issues.slice(0, 5)
  const groups = issueKinds
    .map((kind) => ({ kind, count: data.issues.filter((issue) => issue.kind === kind).length }))
    .filter((group) => group.count > 0)
  const more = data.issues.length > visible.length
  const unchecked = data.resources.checked < data.resources.summary.monitored
  const healthy = data.checklist.filter((item) => item.problems === 0 && item.healthy > 0)
  return (
    // 卡片撑满网格行高：标题贴顶、结论与清单在上、待办入口贴底，避免超宽窗口出现行间空洞。
    <Card className="flex h-full min-w-0 flex-col gap-3 py-4">
      <CardHeader className="flex flex-row items-center justify-between gap-2 px-4">
        <h2 className="text-base font-semibold">{t("dash.issues")}</h2>
        <span className="text-muted-foreground text-xs">
          {t("dash.issuesCount", { count: data.issues.length })}
        </span>
      </CardHeader>
      <CardContent className="flex min-h-0 flex-1 flex-col gap-3 px-4">
        {data.metadataError ? (
          <p role="alert" className="text-destructive text-xs leading-5">
            {t("dash.metadataFailed")}
          </p>
        ) : data.metadataPending ? (
          <p className="text-muted-foreground text-xs">{t("accounts.loading")}</p>
        ) : visible.length === 0 ? (
          <div className="space-y-3">
            <p className="text-muted-foreground flex items-start gap-2 text-sm leading-5">
              <CircleCheck
                className="mt-0.5 size-4 shrink-0 text-emerald-600 dark:text-emerald-400"
                aria-hidden="true"
              />
              <span>{t("dash.issuesEmpty")}</span>
            </p>
            <ul className="space-y-2">
              {data.checklist.map((item) => (
                <li
                  key={item.key}
                  className="bg-muted/30 flex items-center gap-2 rounded-lg border px-3 py-3"
                >
                  {item.problems > 0 ? (
                    <AlertCircle
                      className="size-4 shrink-0 text-amber-600 dark:text-amber-400"
                      aria-hidden="true"
                    />
                  ) : (
                    <CircleCheck
                      className="text-muted-foreground size-4 shrink-0"
                      aria-hidden="true"
                    />
                  )}
                  <span className="min-w-0 flex-1 truncate text-sm">{t(item.key)}</span>
                  <span className={`${problemBadge(item.problems)} tabular-nums`}>
                    {item.problems > 0
                      ? t("dash.checkBad", { count: item.problems })
                      : item.healthy > 0
                        ? t("dash.checkOk", { count: item.healthy })
                        : "—"}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        ) : (
          <>
            {groups.length > 1 ? (
              <div className="flex flex-wrap gap-1.5">
                {groups.map((group) => (
                  <span
                    key={group.kind}
                    className="bg-muted/50 text-muted-foreground inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-xs"
                  >
                    {t(issueGroupKeys[group.kind])}
                    <span className="text-foreground tabular-nums">{group.count}</span>
                  </span>
                ))}
              </div>
            ) : null}
            <ul className="min-h-0 space-y-2 overflow-y-auto">
              {visible.map((issue) => (
                <li
                  key={issue.key}
                  className="flex min-w-0 items-start gap-2 rounded-lg border p-3"
                >
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
                </li>
              ))}
            </ul>
            {/* 有问题的条目之外，把仍然正常的检查项压成一行，说明「其余情况」而不是留白。 */}
            {healthy.length > 0 ? (
              <ul className="text-muted-foreground flex flex-wrap gap-x-3 gap-y-1 text-xs leading-5">
                {healthy.map((item) => (
                  <li key={item.key} className="inline-flex items-center gap-1">
                    <CircleCheck
                      className="size-3.5 shrink-0 text-emerald-600 dark:text-emerald-400"
                      aria-hidden="true"
                    />
                    {t(item.key)}
                    <span className="text-foreground/70 tabular-nums">
                      {t("dash.checkOk", { count: item.healthy })}
                    </span>
                  </li>
                ))}
              </ul>
            ) : null}
          </>
        )}
        {more || unchecked ? (
          <div className="mt-auto space-y-1.5 border-t pt-3 text-xs">
            {more ? (
              <Link
                to="/accounts"
                className="text-muted-foreground hover:text-foreground inline-flex items-center gap-1"
              >
                {t("dash.issueMore")}
                <ArrowUpRight className="size-3" aria-hidden="true" />
              </Link>
            ) : null}
            {unchecked ? (
              <p className="text-muted-foreground leading-5">{t("dash.resourcesUnchecked")}</p>
            ) : null}
          </div>
        ) : null}
      </CardContent>
    </Card>
  )
}
