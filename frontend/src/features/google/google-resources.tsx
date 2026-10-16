import { Link } from "@tanstack/react-router"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import type { AccountItem } from "../../services/omniproxy"
import { openGoogleConsole } from "../../services/google"
import { GoogleResourceCard } from "./google-resource-card"
import { useGoogleResources } from "./use-google-resources"

export function GoogleResources({ accounts }: { accounts: AccountItem[] }) {
  const { t } = useTranslation()
  const [openFailed, setOpenFailed] = useState(false)
  const query = useGoogleResources(accounts)
  if (!accounts.length) return null
  async function open(page: "usage" | "billing") {
    setOpenFailed(false)
    try {
      await openGoogleConsole(page)
    } catch {
      setOpenFailed(true)
    }
  }
  return (
    <section className="space-y-3" aria-label={t("google.resources")}>
      <header className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-base font-semibold">{t("google.resources")}</h2>
        <div className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={query.isFetching}
            onClick={() => void query.refetch()}
          >
            {t("google.refresh")}
          </Button>
          <Button variant="outline" size="sm" onClick={() => void open("usage")}>
            {t("google.consoleUsage")}
          </Button>
          <Button variant="outline" size="sm" onClick={() => void open("billing")}>
            {t("google.consoleBilling")}
          </Button>
          <Button variant="outline" size="sm" asChild>
            <Link to="/usage">{t("google.usage")}</Link>
          </Button>
        </div>
      </header>
      <p className="text-muted-foreground text-sm">{t("google.description")}</p>
      {query.isPending ? (
        <p role="status" className="text-muted-foreground text-sm">
          {t("google.loading")}
        </p>
      ) : null}
      {query.isError || openFailed ? (
        <p role="alert" className="text-destructive text-sm">
          {t("google.error.failed")}
        </p>
      ) : null}
      <div className="grid items-start gap-3 md:grid-cols-2 xl:grid-cols-3">
        {query.data
          ?.filter((resource) =>
            resource.accountIds?.some((id) => accounts.some((account) => account.id === id)),
          )
          .map((resource) => (
            <GoogleResourceCard key={resource.id} resource={resource} accounts={accounts} />
          ))}
      </div>
    </section>
  )
}
