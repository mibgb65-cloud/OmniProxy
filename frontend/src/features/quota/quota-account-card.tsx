import { MiMoBadge } from "../accounts/mimo-options"
import { MiniMaxBadge } from "../accounts/minimax-options"
import { KimiPlatformBadge } from "../accounts/kimi-platform"
import { Link } from "@tanstack/react-router"
import { StepFunBadge } from "../accounts/stepfun-options"
import { ArrowUpRight, RefreshCw } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent } from "../../components/ui/card"
import { Tooltip } from "../../components/ui/tooltip"
import { cn } from "../../lib/utils"
import { accountDisplayName, formatAccountTime } from "../accounts/account-display"
import { AccountStatusBadge } from "../accounts/account-status-badge"
import { SubscriptionPlanBadge } from "../accounts/subscription-plan-badge"
import { QuotaResource } from "./quota-resource"
import { subscriptionTypeKey } from "../providers/provider-labels"
import type { QuotaEntry } from "./quota-model"

export function QuotaAccountCard({ entry, now }: { entry: QuotaEntry; now: number }) {
  const { t } = useTranslation()
  const name = accountDisplayName(entry.account)
  return (
    <Card className="dark:bg-card/75 min-w-0 gap-0 py-0">
      <CardContent className="flex h-full flex-col gap-3 p-4">
        <div className="space-y-1.5">
          <div className="flex min-w-0 items-center justify-between gap-2">
            <Tooltip content={name}>
              <h3 className="min-w-0 flex-1 truncate text-sm font-semibold">{name}</h3>
            </Tooltip>
            {entry.canQuery && entry.refresh ? (
              <Tooltip content={t("quota.refreshAccount", { name })}>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  className="text-muted-foreground cursor-pointer"
                  aria-label={t("quota.refreshAccount", { name })}
                  disabled={entry.fetching}
                  onClick={() => void entry.refresh?.()}
                >
                  <RefreshCw
                    className={cn(
                      "size-4",
                      entry.fetching && "animate-spin motion-reduce:animate-none",
                    )}
                    aria-hidden="true"
                  />
                </Button>
              </Tooltip>
            ) : null}
          </div>
          <div className="flex min-w-0 items-center justify-between gap-2">
            <div className="flex min-w-0 flex-wrap items-center gap-2">
              {entry.kind === "subscription" ? (
                <SubscriptionPlanBadge plan={entry.quota?.plan ?? entry.account.authPlan} />
              ) : (
                <span className="text-muted-foreground text-xs">
                  {t(
                    entry.account.authMethod === "subscription_login"
                      ? subscriptionTypeKey(entry.provider?.kind ?? entry.account.providerId)
                      : "accounts.apiKey",
                  )}
                </span>
              )}
              <StepFunBadge account={entry.account} />
              <MiniMaxBadge account={entry.account} />
              <MiMoBadge account={entry.account} />
              <KimiPlatformBadge account={entry.account} />
              {!entry.account.enabled ? <AccountStatusBadge enabled={false} /> : null}
            </div>
            <span
              className={cn(
                "ml-auto w-fit max-w-full rounded-md border px-2 py-0.5 text-xs leading-5",
                entry.status === "normal"
                  ? "border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900 dark:bg-emerald-950 dark:text-emerald-300"
                  : entry.status === "error" || entry.status === "exhausted"
                    ? "border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-200"
                    : "bg-muted text-muted-foreground",
              )}
            >
              {t(`quota.status.${entry.status}`)}
            </span>
          </div>
        </div>
        <QuotaResource entry={entry} now={now} />
        <div className="text-muted-foreground mt-auto flex flex-wrap items-end justify-between gap-2 border-t pt-3 text-xs">
          <span className="min-w-0 flex-1 leading-5">
            {entry.fetchedAt
              ? t("quota.updated", { time: formatAccountTime(entry.fetchedAt) })
              : t("quota.neverUpdated")}
          </span>
          {entry.provider ? (
            <Link
              to="/accounts/$providerId"
              params={{ providerId: entry.account.providerId }}
              aria-label={t("quota.detailsLabel", { name })}
              className="hover:text-foreground focus-visible:ring-ring/50 inline-flex items-center gap-1 rounded-md outline-none focus-visible:ring-2"
            >
              {t("quota.details")}
              <ArrowUpRight className="size-3.5" aria-hidden="true" />
            </Link>
          ) : null}
        </div>
      </CardContent>
    </Card>
  )
}
