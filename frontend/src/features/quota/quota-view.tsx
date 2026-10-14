import { Link } from "@tanstack/react-router"
import { RefreshCw } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { cn } from "../../lib/utils"
import { ProviderBrandIcon } from "../providers/provider-brand-icon"
import { QuotaAccountCard } from "./quota-account-card"
import { QuotaFilters } from "./quota-filters"
import { QuotaSummary } from "./quota-summary"
import { useQuotaOverview } from "./use-quota-overview"
import { GoogleResources } from "../google/google-resources"

export function QuotaView() {
  const { t } = useTranslation()
  const data = useQuotaOverview()
  return (
    <section className="w-full min-w-0 space-y-5 pb-6">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div className="space-y-1.5">
          <h1 className="text-2xl font-semibold tracking-tight">{t("pages.quota.title")}</h1>
          <p className="text-muted-foreground text-sm">{t("quota.description")}</p>
        </div>
        <Button
          variant="outline"
          disabled={!data.canRefresh}
          onClick={() => void data.refreshVisible()}
        >
          <RefreshCw
            className={cn("size-4", data.refreshing && "animate-spin motion-reduce:animate-none")}
            aria-hidden="true"
          />
          {t(data.refreshing ? "quota.refreshing" : "quota.refreshVisible")}
        </Button>
      </header>
      {data.metadataLoading ? (
        <p role="status" className="text-muted-foreground text-sm">
          {t("quota.loading")}
        </p>
      ) : null}
      {data.metadataError ? (
        <div className="flex flex-wrap items-center gap-3">
          <p role="alert" className="text-destructive text-sm">
            {t("quota.metadataError")}
          </p>
          <Button
            variant="outline"
            size="sm"
            disabled={data.metadataRefreshing}
            onClick={() => void data.refreshMetadata()}
          >
            {t("quota.retryMetadata")}
          </Button>
        </div>
      ) : null}
      <QuotaSummary data={data} />
      <QuotaFilters data={data} />
      {data.ready ? (
        <div className="text-muted-foreground flex flex-wrap items-center justify-between gap-2 text-xs">
          <span>
            {t("quota.showing", { visible: data.visibleCount, total: data.entries.length })}
          </span>
          {data.progress.fetching ? (
            <span role="status" aria-atomic="true">
              {t("quota.progress", data.progress)}
            </span>
          ) : null}
        </div>
      ) : null}
      {data.batchResult ? (
        <p role="status" className="text-muted-foreground text-sm">
          {t(
            data.batchResult.failed ? "quota.batchPartial" : "quota.batchSuccess",
            data.batchResult,
          )}
        </p>
      ) : null}
      {data.ready && data.entries.length === 0 ? (
        <div className="space-y-3 rounded-xl border border-dashed p-6">
          <p className="text-muted-foreground text-sm">{t("quota.noAccounts")}</p>
          <Button variant="outline" asChild>
            <Link to="/accounts">{t("quota.addAccounts")}</Link>
          </Button>
        </div>
      ) : data.ready && data.groups.length === 0 ? (
        <p className="text-muted-foreground rounded-xl border border-dashed p-6 text-sm">
          {t("quota.noMatches")}
        </p>
      ) : null}
      <GoogleResources
        accounts={data.groups.flatMap((group) =>
          group.entries.filter((entry) => entry.kind === "google").map((entry) => entry.account),
        )}
      />
      {data.groups
        .filter((group) => group.entries.some((entry) => entry.kind !== "google"))
        .map((group) => (
          <section
            key={group.id}
            className="space-y-3"
            aria-label={group.provider?.name ?? t("quota.unknownProvider")}
          >
            <header className="flex items-center gap-3">
              <ProviderBrandIcon
                kind={group.provider?.kind ?? ""}
                endpoint={group.provider?.baseUrl}
              />
              <div className="min-w-0">
                <h2 className="text-base font-semibold break-words">
                  {group.provider?.name ?? t("quota.unknownProvider")}
                </h2>
                <p className="text-muted-foreground text-xs">
                  {t("quota.count", { count: group.entries.length })}
                </p>
              </div>
            </header>
            <div className="grid items-stretch gap-3 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
              {group.entries
                .filter((entry) => entry.kind !== "google")
                .map((entry) => (
                  <QuotaAccountCard key={entry.account.id} entry={entry} now={data.now} />
                ))}
            </div>
          </section>
        ))}
      <p className="text-muted-foreground text-xs leading-5">{t("quota.readOnlyNote")}</p>
    </section>
  )
}
