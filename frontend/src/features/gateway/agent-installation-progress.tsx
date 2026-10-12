import { LoaderCircle } from "lucide-react"
import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import type { AgentInstallation } from "../../services/agent-installation"
import "./agent-installation-progress.css"

const phaseLabels = {
  preparing: "agents.install.preparing",
  downloading: "agents.install.downloading",
  installing: "agents.install.installingStage",
  verifying: "agents.install.verifying",
} as const

export function AgentInstallationProgress({
  item,
  label,
  starting,
}: {
  item?: AgentInstallation
  label: string
  starting: boolean
}) {
  const { t } = useTranslation()
  const active = !!item && ["checking", "installing", "verifying"].includes(item.installState)
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    if (!active) return
    const timer = setInterval(() => setNow(Date.now()), 1000)
    return () => clearInterval(timer)
  }, [active])
  if (!active && !starting) return null
  const reported = starting ? undefined : item?.progress
  const phase =
    starting || item?.installState === "checking"
      ? "preparing"
      : item?.installState === "verifying"
        ? "verifying"
        : reported?.phase === "downloading"
          ? "downloading"
          : reported?.phase === "installing"
            ? "installing"
            : "preparing"
  const percent =
    typeof reported?.percent === "number" &&
    Number.isFinite(reported.percent) &&
    reported.percent >= 0 &&
    reported.percent <= 100
      ? reported.percent
      : undefined
  const start = starting ? NaN : Date.parse(item?.startedAt ?? "")
  const elapsed = Number.isFinite(start) ? Math.max(0, Math.floor((now - start) / 1000)) : null
  const elapsedText =
    elapsed === null ? null : `${Math.floor(elapsed / 60)}:${String(elapsed % 60).padStart(2, "0")}`
  const phaseLabel = t(phaseLabels[phase])
  return (
    <div className="bg-muted/25 space-y-2.5 rounded-lg border p-3">
      <div className="flex flex-wrap items-center justify-between gap-2 text-xs">
        <span className="flex items-center gap-2">
          <LoaderCircle
            aria-hidden="true"
            className="text-muted-foreground size-3.5 motion-safe:animate-spin"
          />
          {phaseLabel}
        </span>
        <span className="text-muted-foreground tabular-nums">
          {percent === undefined ? t("agents.install.waitingProgress") : `${Math.round(percent)}%`}
        </span>
      </div>
      <div
        role="progressbar"
        aria-label={t("agents.install.progressLabel", { client: label })}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={percent}
        aria-valuetext={
          percent === undefined ? phaseLabel : `${phaseLabel} ${Math.round(percent)}%`
        }
        className="bg-muted h-1.5 overflow-hidden rounded-full"
      >
        <div
          className={
            percent === undefined
              ? "agent-installation-progress__indeterminate bg-primary/65 h-full w-1/3 rounded-full"
              : "bg-primary h-full rounded-full motion-safe:transition-[width]"
          }
          style={percent === undefined ? undefined : { width: `${percent}%` }}
        />
      </div>
      <div className="text-muted-foreground flex flex-wrap items-center justify-between gap-2 text-xs tabular-nums">
        <span>
          {phase === "downloading" && reported && reported.totalBytes > 0
            ? t("agents.install.downloaded", {
                downloaded: formatBytes(reported.downloadedBytes),
                total: formatBytes(reported.totalBytes),
              })
            : t("agents.install.phaseHint")}
        </span>
        {elapsedText ? <span>{t("agents.install.elapsed", { time: elapsedText })}</span> : null}
      </div>
    </div>
  )
}

function formatBytes(bytes: number) {
  if (!Number.isFinite(bytes) || bytes < 0) return "—"
  const units = ["B", "KB", "MB", "GB", "TB"]
  const unit = Math.min(Math.floor(Math.log2(Math.max(1, bytes)) / 10), units.length - 1)
  return `${(bytes / 1024 ** unit).toFixed(unit ? 1 : 0)} ${units[unit]}`
}
