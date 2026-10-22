import { useEffect } from "react"
import { useQuery } from "@tanstack/react-query"
import { Events } from "@wailsio/runtime"
import { useTranslation } from "react-i18next"
import { readAccountTokenUsage } from "../../services/activity"
import type { AccountItem } from "../../services/omniproxy"
import { Button } from "../../components/ui/button"
import { AccountTokenSummary } from "./account-token-summary"

// 本机留存用量：没有官方额度／余额接口的账号（River、自定义提供商等）用它展示已用 Token。
export function AccountUsagePanel({ account }: { account: AccountItem }) {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: ["account-token-usage", account.id, account.updatedAt],
    queryFn: () => readAccountTokenUsage(account.id),
    retry: false,
    staleTime: 30_000,
  })
  const refetch = query.refetch
  useEffect(() => {
    const refresh = () => {
      if (document.visibilityState !== "hidden") void refetch({ cancelRefetch: false })
    }
    const off = Events.On("activity:updated", refresh)
    document.addEventListener("visibilitychange", refresh)
    return () => {
      off()
      document.removeEventListener("visibilitychange", refresh)
    }
  }, [refetch])

  return (
    <div className="min-w-0 space-y-2">
      <AccountTokenSummary usage={query.data} error={query.isError} />
      {query.isPending ? (
        <p role="status" className="text-muted-foreground text-xs">
          {t("activity.accountUsageLoading")}
        </p>
      ) : query.isError ? (
        <div role="alert" className="flex flex-wrap items-center gap-2 text-xs">
          <span className="text-destructive">{t("activity.accountUsageFailed")}</span>
          <Button
            size="sm"
            variant="outline"
            disabled={query.isFetching}
            onClick={() => void refetch()}
          >
            {t("activity.accountUsageRetry")}
          </Button>
        </div>
      ) : query.data?.incompleteRequests ? (
        <p className="text-muted-foreground text-xs leading-5">
          {t("activity.accountUsageIncomplete")}
        </p>
      ) : null}
    </div>
  )
}
