import { useEffect, useState } from "react"
import { RefreshCw } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Tooltip } from "../../components/ui/tooltip"
import type { AccountItem } from "../../services/omniproxy"
import { subscriptionError } from "../../services/subscription"
import { formatAccountTime } from "../accounts/account-display"
import { useGoogleSubscriptionQuota } from "./use-google-subscription-quota"
import { GoogleQuotaValues } from "./google-quota-values"
import { GoogleSubscriptionModels } from "./google-subscription-models"

export function GoogleSubscriptionQuota({ account }: { account: AccountItem }) {
  const { t } = useTranslation()
  const query = useGoogleSubscriptionQuota(account)
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 30_000)
    return () => clearInterval(timer)
  }, [])
  const authenticated = account.authState === "authenticated"
  return (
    <section
      aria-label={t("google.quota.title")}
      aria-busy={query.isFetching}
      className="bg-muted/20 min-w-0 space-y-3 rounded-xl border p-3"
    >
      <div className="flex items-center justify-between gap-2">
        <p className="text-xs font-medium">{t("google.quota.title")}</p>
        {authenticated ? (
          <Tooltip content={t("google.quota.refresh")}>
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              disabled={query.isFetching}
              className="cursor-pointer"
              aria-label={t("google.quota.refresh")}
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
        ) : null}
      </div>
      <GoogleSubscriptionModels account={account} />
      {authenticated ? (
        <GoogleQuotaValues quota={query.data} loading={query.isPending} now={now} />
      ) : (
        <p className="text-muted-foreground text-xs">{t("quota.signInHint")}</p>
      )}
      {query.isError && authenticated ? (
        <p role="alert" className="text-destructive text-xs leading-5">
          {t(`subscription.error.${subscriptionError(query.error)}`)}
        </p>
      ) : null}
      {query.data && authenticated ? (
        <p className="text-muted-foreground text-[10px]">
          {t("quota.updated", { time: formatAccountTime(query.data.fetchedAt) })}
        </p>
      ) : null}
      <p className="text-muted-foreground text-[10px] leading-5">{t("google.quota.hint")}</p>
    </section>
  )
}
