import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/ui/card"
import { formatBytes } from "../../lib/format"
import type { StorageInfo } from "../../services/omniproxy"
import { useCleanStorage, useStorageInfo, type CleanTarget } from "./use-storage"

export function StorageOverview() {
  const { t } = useTranslation()
  const { data, isPending, isError, error } = useStorageInfo()
  const clean = useCleanStorage()

  const runClean = (target: CleanTarget) => clean.mutate(target)

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("settings.storage.title")}</CardTitle>
        <CardDescription>{t("settings.storage.description")}</CardDescription>
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
            <dl className="grid gap-2">
              <Row label={t("settings.storage.dataDir")} value={data.dataDir} mono />
              <Row label={t("settings.storage.logDir")} value={data.logDir} mono />
              <Row
                label={t("settings.storage.database")}
                value={`${formatBytes(itemBytes(data, "database"))} — ${data.databasePath}`}
                mono
              />
              <Row
                label={t("settings.storage.logs")}
                value={formatBytes(itemBytes(data, "logs"))}
              />
              <Row
                label={t("settings.storage.total")}
                value={formatBytes(data.totalBytes)}
                strong
              />
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

function itemBytes(data: StorageInfo, name: string): number {
  return data.items?.find((item) => item.name === name)?.bytes ?? 0
}

function Row({
  label,
  value,
  mono,
  strong,
}: {
  label: string
  value: string
  mono?: boolean
  strong?: boolean
}) {
  return (
    <div className="flex items-baseline justify-between gap-4">
      <dt className="text-muted-foreground shrink-0">{label}</dt>
      <dd
        className={`truncate text-right ${mono ? "font-mono text-xs" : ""} ${
          strong ? "font-medium" : ""
        }`}
        title={value}
      >
        {value}
      </dd>
    </div>
  )
}
