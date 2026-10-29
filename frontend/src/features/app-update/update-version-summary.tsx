import { ArrowRight, CalendarDays, ExternalLink, HardDrive } from "lucide-react"
import { useTranslation } from "react-i18next"
import { formatBytes } from "../../lib/format"
import type { ApplicationUpdate } from "../../services/updates"

/** 升级浮窗的版本摘要卡：当前版本 → 可升级版本，并附发布时间、包大小与发布页入口。 */
export function UpdateVersionSummary({ state }: { state: ApplicationUpdate }) {
  const { t } = useTranslation()
  const release = state.release
  const target = release?.version ?? state.download.version
  const published = release?.publishedAt ? new Date(release.publishedAt) : undefined
  const publishedAt =
    published && !Number.isNaN(published.getTime()) ? published.toLocaleString() : ""
  return (
    <section aria-label={t("appUpdate.releaseInfo")} className="bg-muted/40 rounded-xl border p-4">
      <div className="grid items-center gap-3 sm:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)]">
        <div className="min-w-0 space-y-1">
          <p className="text-muted-foreground text-xs">{t("appUpdate.currentLabel")}</p>
          <p className="font-mono text-sm [overflow-wrap:anywhere]">{state.currentVersion}</p>
        </div>
        <ArrowRight
          className="text-muted-foreground hidden size-4 shrink-0 sm:block"
          aria-hidden="true"
        />
        <div className="min-w-0 space-y-1 sm:text-right">
          <p className="text-muted-foreground text-xs">{t("appUpdate.targetLabel")}</p>
          <p className="text-primary font-mono text-sm font-semibold [overflow-wrap:anywhere]">
            {target}
          </p>
        </div>
      </div>
      {publishedAt || release?.size || release?.url ? (
        <div className="text-muted-foreground mt-3 flex flex-wrap items-center gap-x-4 gap-y-1.5 border-t pt-3 text-xs">
          {publishedAt ? (
            <span className="inline-flex items-center gap-1.5">
              <CalendarDays className="size-3.5 shrink-0" aria-hidden="true" />
              {publishedAt}
            </span>
          ) : null}
          {release?.size ? (
            <span className="inline-flex items-center gap-1.5">
              <HardDrive className="size-3.5 shrink-0" aria-hidden="true" />
              {formatBytes(release.size)}
            </span>
          ) : null}
          {release?.url ? (
            <a
              href={release.url}
              target="_blank"
              rel="noreferrer"
              className="text-primary inline-flex items-center gap-1 underline underline-offset-2 sm:ml-auto"
            >
              {t("about.update.openRelease")}
              <ExternalLink className="size-3 shrink-0" aria-hidden="true" />
            </a>
          ) : null}
        </div>
      ) : null}
    </section>
  )
}
