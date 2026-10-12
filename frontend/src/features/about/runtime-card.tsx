import { Tooltip } from "../../components/ui/tooltip"
import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/ui/card"
import { formatDuration } from "../../lib/format"
import { useRuntimeInfo } from "./use-about"

/** Runtime details card with a live uptime counter. */
export function RuntimeCard() {
  const { t } = useTranslation()
  const { data, isPending, isError, error } = useRuntimeInfo()
  const uptime = useUptime(data?.startedAt)

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("about.runtime.title")}</CardTitle>
        <CardDescription>{t("about.runtime.description")}</CardDescription>
      </CardHeader>
      <CardContent className="text-sm">
        {isPending ? <p className="text-muted-foreground">{t("dashboard.loading")}</p> : null}
        {isError ? (
          <p className="text-destructive">
            {t("about.runtime.loadFailed", { error: error.message })}
          </p>
        ) : null}
        {data ? (
          <dl className="grid gap-2">
            <Row label={t("about.runtime.profile")} value={data.profile} />
            <Row label={t("about.runtime.version")} value={data.version} />
            <Row label={t("about.runtime.goVersion")} value={data.goVersion} />
            <Row label={t("about.runtime.wailsVersion")} value={data.wailsVersion} />
            <Row label={t("about.runtime.sqliteVersion")} value={data.sqliteVersion} />
            <Row label={t("about.runtime.platform")} value={data.platform} />
            <Row label={t("about.runtime.appId")} value={data.appId} mono />
            <Row label={t("about.runtime.dataDir")} value={data.dataDir} mono />
            <Row label={t("about.runtime.logDir")} value={data.logDir} mono />
            <Row
              label={t("about.runtime.ports")}
              value={`${data.proxyPort} / ${data.controlPort}`}
            />
            <Row label={t("about.runtime.uptime")} value={uptime} mono />
          </dl>
        ) : null}
      </CardContent>
    </Card>
  )
}

/** Ticks once per second while the app is running. */
function useUptime(startedAt?: string): string {
  const [now, setNow] = useState(() => Date.now())

  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 1000)
    return () => clearInterval(timer)
  }, [])

  if (!startedAt) {
    return "--:--:--"
  }
  const started = Date.parse(startedAt)
  if (Number.isNaN(started)) {
    return "--:--:--"
  }
  return formatDuration(Math.max(0, now - started))
}

function Row({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-baseline justify-between gap-4">
      <dt className="text-muted-foreground shrink-0">{label}</dt>
      <Tooltip content={value}>
        <dd className={`truncate text-right ${mono ? "font-mono text-xs" : ""}`}>{value}</dd>
      </Tooltip>
    </div>
  )
}
