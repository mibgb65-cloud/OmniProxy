import { useState } from "react"
import { Link } from "@tanstack/react-router"
import { RefreshCw } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import type { AccountItem } from "../../services/omniproxy"
import { openGoogleConsole } from "../../services/google"
import { GoogleResourceContent } from "./google-resource-card"
import { useGoogleResources } from "./use-google-resources"

export function GoogleAccountResource({
  account,
  accounts,
}: {
  account: AccountItem
  accounts: AccountItem[]
}) {
  const { t } = useTranslation()
  const query = useGoogleResources(accounts)
  const [openFailed, setOpenFailed] = useState(false)
  const resource = query.data?.find((item) => item.accountIds?.includes(account.id))
  async function open(page: "usage" | "billing") {
    setOpenFailed(false)
    try {
      await openGoogleConsole(page)
    } catch {
      setOpenFailed(true)
    }
  }
  return (
    <section
      aria-label={t("google.resources")}
      aria-busy={query.isFetching}
      className="bg-muted/20 min-w-0 space-y-3 rounded-xl border p-3"
    >
      <header className="flex items-center justify-between gap-2">
        <h4 className="text-xs font-medium">{t("google.resources")}</h4>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label={t("google.refresh")}
          disabled={query.isFetching}
          onClick={() => void query.refetch()}
        >
          <RefreshCw
            className={
              query.isFetching ? "size-3 animate-spin motion-reduce:animate-none" : "size-3"
            }
            aria-hidden="true"
          />
        </Button>
      </header>
      {query.isPending ? (
        <p role="status" className="text-muted-foreground text-xs">
          {t("google.loading")}
        </p>
      ) : null}
      {resource ? <GoogleResourceContent resource={resource} accounts={[account]} /> : null}
      {query.isError || openFailed ? (
        <p role="alert" className="text-destructive text-xs">
          {t("google.error.failed")}
        </p>
      ) : null}
      <div className="flex flex-wrap gap-2 border-t pt-3">
        <Button type="button" variant="outline" size="sm" onClick={() => void open("usage")}>
          {t("google.consoleUsage")}
        </Button>
        <Button type="button" variant="outline" size="sm" onClick={() => void open("billing")}>
          {t("google.consoleBilling")}
        </Button>
        <Button variant="outline" size="sm" asChild>
          <Link to="/usage">{t("google.usage")}</Link>
        </Button>
      </div>
    </section>
  )
}
