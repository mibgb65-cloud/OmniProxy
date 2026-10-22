import { useEffect } from "react"
import { useQuery } from "@tanstack/react-query"
import { Events } from "@wailsio/runtime"
import { useTranslation } from "react-i18next"
import { readRiverTokenUsage } from "../../services/activity"
import type { AccountItem } from "../../services/omniproxy"
import { Button } from "../../components/ui/button"
import { AccountTokenSummary } from "./account-token-summary"

export function RiverTokenUsage({ account }: { account: AccountItem }) {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: ["river-token-usage", account.id, account.updatedAt],
    queryFn: () => readRiverTokenUsage(account.id),
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
          {t("river.usage.loading")}
        </p>
      ) : query.isError ? (
        <div role="alert" className="flex flex-wrap items-center gap-2 text-xs">
          <span className="text-destructive">{t("river.usage.failed")}</span>
          <Button
            size="sm"
            variant="outline"
            disabled={query.isFetching}
            onClick={() => void refetch()}
          >
            {t("river.usage.retry")}
          </Button>
        </div>
      ) : query.data?.incompleteRequests ? (
        <p className="text-muted-foreground text-xs leading-5">{t("river.usage.incomplete")}</p>
      ) : null}
    </div>
  )
}
