import { StepFunBalanceValues } from "./stepfun-resource"
import { RefreshCw } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Tooltip } from "../../components/ui/tooltip"
import type { AccountItem } from "../../services/omniproxy"
import { balanceError } from "../../services/balance"
import { useAccountBalance } from "./use-account-balance"
import { formatAccountTime } from "./account-display"
import "./subscription-quota.css"

export function AccountBalance({ account }: { account: AccountItem }) {
  const { t } = useTranslation()
  const query = useAccountBalance(account)
  const data = query.data
  const entries = data?.entries || []
  return (
    <section
      aria-label={t("balance.title")}
      aria-busy={query.isFetching}
      className="bg-muted/20 min-w-0 space-y-3 rounded-xl border p-3"
    >
      <div className="flex items-center justify-between gap-2">
        <span className="text-sm font-medium">{t("balance.title")}</span>
        <div className="flex items-center gap-2">
          {data && !data.stepfun ? (
            <span
              className={`rounded-md px-2 py-0.5 text-[10px] ${data.available ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300" : "bg-amber-50 text-amber-800 dark:bg-amber-950 dark:text-amber-200"}`}
            >
              {t(data.available ? "balance.available" : "balance.insufficient")}
            </span>
          ) : null}
          <Tooltip content={t("balance.refresh")}>
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              aria-label={t("balance.refresh")}
              disabled={query.isFetching}
              className="text-muted-foreground cursor-pointer"
              onClick={() => void query.refetch({ cancelRefetch: false })}
            >
              <RefreshCw
                className={
                  query.isFetching ? "size-3 animate-spin motion-reduce:animate-none" : "size-3"
                }
                aria-hidden="true"
              />
            </Button>
          </Tooltip>
        </div>
      </div>
      {data ? (
        data.stepfun ? (
          <StepFunBalanceValues value={data.stepfun} />
        ) : entries.length > 0 ? (
          <div className="space-y-3">
            {entries.map((entry) => (
              <div key={entry.currency} className="min-w-0 space-y-2">
                <div className="flex min-w-0 flex-wrap items-baseline justify-between gap-2">
                  <span className="text-muted-foreground text-xs">
                    {t("balance.total")} <span className="font-mono">{entry.currency}</span>
                  </span>
                  <span
                    className="quota-information-update max-w-full font-mono text-xl font-semibold break-all tabular-nums"
                    key={`${entry.currency}:${entry.total}`}
                  >
                    {entry.currency === "CNY" ? "¥" : "US$"}
                    {entry.total}
                  </span>
                </div>
                <dl className="text-muted-foreground grid grid-cols-2 gap-x-3 gap-y-1 text-[11px]">
                  <div>
                    <dt>{t("balance.granted")}</dt>
                    <dd className="mt-0.5 font-mono break-all tabular-nums">{entry.granted}</dd>
                  </div>
                  <div>
                    <dt>{t("balance.toppedUp")}</dt>
                    <dd className="mt-0.5 font-mono break-all tabular-nums">{entry.toppedUp}</dd>
                  </div>
                </dl>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-muted-foreground text-xs">{t("balance.empty")}</p>
        )
      ) : query.isPending ? (
        <div aria-hidden="true" className="space-y-2">
          <div className="bg-muted h-6 w-32 rounded" />
          <div className="bg-muted h-4 w-48 max-w-full rounded" />
        </div>
      ) : null}
      {query.isError ? (
        <p role="alert" className="text-destructive text-xs">
          {t(`balance.error.${balanceError(query.error)}`, {
            defaultValue: t("balance.error.query_failed"),
          })}
        </p>
      ) : null}
      {data ? (
        <p className="text-muted-foreground text-[10px]">
          {t("balance.updated", { time: formatAccountTime(data.fetchedAt) })}
        </p>
      ) : null}
    </section>
  )
}
