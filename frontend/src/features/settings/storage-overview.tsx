import { Tooltip } from "../../components/ui/tooltip"
import { RefreshCw } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/ui/card"
import { formatBytes } from "../../lib/format"
import { StoragePathList } from "./storage-path-list"
import { StorageUsageChart } from "./storage-usage-chart"
import { useCleanStorage, useStorageInfo, type CleanTarget } from "./use-storage"

export function StorageOverview() {
  const { t } = useTranslation()
  const { data, isPending, isFetching, isError, error, refetch } = useStorageInfo()
  const clean = useCleanStorage()

  const runClean = (target: CleanTarget) => clean.mutate(target)

  return (
    <Card className="dark:bg-card/75 gap-0 overflow-hidden">
      <CardHeader className="grid-cols-[1fr_auto] items-center pb-5">
        <div>
          <CardTitle className="text-base">{t("settings.storage.title")}</CardTitle>
          <CardDescription className="mt-1">{t("settings.storage.description")}</CardDescription>
        </div>
        <Tooltip content={t("settings.storage.refresh")}>
          <Button
            type="button"
            size="icon-sm"
            variant="ghost"
            className="text-muted-foreground hover:bg-muted/50 hover:text-foreground dark:hover:bg-muted/50 size-9 appearance-none border bg-transparent dark:bg-transparent"
            aria-label={t("settings.storage.refresh")}
            disabled={isFetching}
            onClick={() => void refetch()}
          >
            <RefreshCw
              aria-hidden="true"
              className={isFetching ? "motion-safe:animate-spin" : ""}
            />
          </Button>
        </Tooltip>
      </CardHeader>
      <CardContent className="text-sm">
        {isPending ? <p className="text-muted-foreground">{t("dashboard.loading")}</p> : null}
        {isError ? (
          <p className="text-destructive">
            {t("settings.storage.loadFailed", { error: error.message })}
          </p>
        ) : null}

        {data ? (
          <>
            <StorageUsageChart data={data} />
            <div className="mt-5">
              <StoragePathList data={data} />
            </div>
            {data.pendingDataDir ? (
              <p className="bg-muted text-muted-foreground mt-4 rounded-md px-3 py-2">
                {t("settings.storage.pending", { path: data.pendingDataDir })}
              </p>
            ) : null}
            <div className="mt-5 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
              <p className="text-muted-foreground max-w-md text-xs">
                {t("settings.storage.databaseNote")}
              </p>
              <div className="flex flex-wrap gap-2 sm:justify-end">
                <Button
                  size="sm"
                  variant="outline"
                  className="border-destructive/40 text-destructive hover:bg-destructive/10 hover:text-destructive"
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
            </div>
            {clean.isError ? (
              <p className="text-destructive mt-3">
                {t("settings.storage.cleanFailed", { error: clean.error.message })}
              </p>
            ) : null}
            {clean.isSuccess ? (
              <p className="text-muted-foreground mt-3" role="status">
                {t("settings.storage.cleaned", { size: formatBytes(clean.data.freedBytes) })}
              </p>
            ) : null}
          </>
        ) : null}
      </CardContent>
    </Card>
  )
}
