import { RefreshCw } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/ui/card"
import { formatBytes } from "../../lib/format"
import { StorageUsageChart } from "./storage-usage-chart"
import { useCleanStorage, useStorageInfo, type CleanTarget } from "./use-storage"

export function StorageOverview() {
  const { t } = useTranslation()
  const { data, isPending, isFetching, isError, error, refetch } = useStorageInfo()
  const clean = useCleanStorage()

  const runClean = (target: CleanTarget) => clean.mutate(target)

  return (
    <Card>
      <CardHeader className="grid-cols-[1fr_auto]">
        <div>
          <CardTitle>{t("settings.storage.title")}</CardTitle>
          <CardDescription className="mt-2">{t("settings.storage.description")}</CardDescription>
        </div>
        <Button size="sm" variant="outline" disabled={isFetching} onClick={() => void refetch()}>
          <RefreshCw aria-hidden="true" className={isFetching ? "animate-spin" : ""} />
          {t("settings.storage.refresh")}
        </Button>
      </CardHeader>
      <CardContent className="space-y-4 text-sm">
        {isPending ? <p className="text-muted-foreground">{t("dashboard.loading")}</p> : null}
        {isError ? (
          <p className="text-destructive">
            {t("settings.storage.loadFailed", { error: error.message })}
          </p>
        ) : null}

        {data ? (
          <>
            <StorageUsageChart data={data} />

            <dl className="grid gap-2">
              <Row label={t("settings.storage.dataDir")} value={data.dataDir} mono />
              <Row label={t("settings.storage.logDir")} value={data.logDir} mono />
              <Row label={t("settings.storage.databasePath")} value={data.databasePath} mono />
            </dl>

            {data.pendingDataDir ? (
              <p className="bg-muted text-muted-foreground rounded-md px-3 py-2">
                {t("settings.storage.pending", { path: data.pendingDataDir })}
              </p>
            ) : null}

            <div className="flex flex-wrap gap-2">
              <Button
                size="sm"
                variant="outline"
                disabled={clean.isPending}
                onClick={() => runClean("logs")}
              >
                {clean.isPending && clean.variables === "logs"
                  ? t("settings.storage.cleaning")
                  : t("settings.storage.cleanLogs")}
              </Button>
              <Button
                size="sm"
                variant="outline"
                disabled={clean.isPending}
                onClick={() => runClean("database")}
              >
                {clean.isPending && clean.variables === "database"
                  ? t("settings.storage.cleaning")
                  : t("settings.storage.cleanDatabase")}
              </Button>
            </div>

            {clean.isError ? (
              <p className="text-destructive">
                {t("settings.storage.cleanFailed", { error: clean.error.message })}
              </p>
            ) : null}
            {clean.isSuccess ? (
              <p className="text-muted-foreground">
                {t("settings.storage.cleaned", { size: formatBytes(clean.data.freedBytes) })}
              </p>
            ) : null}
          </>
        ) : null}
      </CardContent>
    </Card>
  )
}

function Row({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex flex-col gap-1 sm:flex-row sm:items-baseline sm:justify-between sm:gap-4">
      <dt className="text-muted-foreground shrink-0">{label}</dt>
      <dd className={`break-all sm:text-right ${mono ? "font-mono text-xs" : ""}`} title={value}>
        {value}
      </dd>
    </div>
  )
}
