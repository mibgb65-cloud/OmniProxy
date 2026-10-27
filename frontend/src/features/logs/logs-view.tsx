import { useDeferredValue, useMemo, useState } from "react"
import { useMutation } from "@tanstack/react-query"
import { Download, FolderOpen, Pause, Play, RefreshCw, Trash2 } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { ScrollRegion } from "../../components/common/scroll-region"
import { LogRow } from "./log-row"
import { LogFooter } from "./log-footer"
import { exportLogs, openLogDirectory } from "../../services/logs"
import { useUIStore } from "../../stores/ui-store"
import { filterLogs, type LogLevel } from "./log-stream"
import { LogFilters } from "./log-filters"
import { useLiveLogs } from "./use-live-logs"
import { useLogFollow } from "./use-log-follow"

export function LogsView({
  requestId = "",
  onOpenRequest,
}: {
  requestId?: string
  onOpenRequest?: (id: string) => void
}) {
  const { t } = useTranslation()
  const language = useUIStore((state) => state.language)
  const logs = useLiveLogs()
  const [levels, setLevels] = useState<LogLevel[]>(["INFO", "WARN", "ERROR"])
  const [module, setModule] = useState("")
  const [keyword, setKeyword] = useState(requestId)
  const search = useDeferredValue(keyword)
  const entries = useMemo(
    () => filterLogs(logs.data?.entries ?? [], levels, module, search),
    [logs.data?.entries, levels, module, search],
  )
  const follow = useLogFollow(entries, logs.data?.cursor ?? 0)
  const exportAction = useMutation({
    mutationFn: () =>
      exportLogs(
        entries.map((entry) => entry.id),
        language,
      ),
  })
  const directoryAction = useMutation({ mutationFn: openLogDirectory })
  const pending = logs.paused ? logs.pending : follow.unread

  return (
    // app-page-fit 扣掉标题栏与根页面上下留白，日志列表占满剩余窗口高度。
    <section className="app-page-fit flex min-h-0 flex-col gap-4">
      <header className="flex shrink-0 flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-semibold tracking-tight">{t("pages.logs.title")}</h1>
        <div className="flex flex-wrap gap-2">
          <Button
            type="button"
            variant="outline"
            disabled={!entries.length || exportAction.isPending}
            onClick={() => exportAction.mutate()}
          >
            <Download aria-hidden="true" />
            {t(exportAction.isPending ? "logs.exporting" : "logs.export")}
          </Button>
          <Button
            type="button"
            variant="outline"
            disabled={directoryAction.isPending}
            onClick={() => directoryAction.mutate()}
          >
            <FolderOpen aria-hidden="true" />
            {t("logs.openDirectory")}
          </Button>
        </div>
      </header>
      <LogFilters
        levels={levels}
        onLevels={setLevels}
        module={module}
        onModule={setModule}
        modules={logs.data?.modules ?? []}
        keyword={keyword}
        onKeyword={setKeyword}
      />
      <div className="flex shrink-0 flex-wrap items-center justify-between gap-2 text-xs">
        <div className="flex flex-wrap items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={!logs.query.data}
            onClick={logs.togglePaused}
          >
            {logs.paused ? <Play aria-hidden="true" /> : <Pause aria-hidden="true" />}
            {t(logs.paused ? "logs.resume" : "logs.pause")}
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            disabled={!logs.query.data || logs.clear.isPending}
            onClick={() => logs.clear.mutate()}
          >
            <Trash2 aria-hidden="true" />
            {t("logs.clear")}
          </Button>
          <span className="text-muted-foreground">
            {t(logs.paused ? "logs.paused" : "logs.live")}
          </span>
        </div>
        <span className="text-muted-foreground">
          {t("logs.count", {
            visible: entries.length,
            total: logs.data?.entries.length ?? 0,
            capacity: logs.data?.capacity ?? 5000,
          })}
        </span>
      </div>
      {logs.query.isError ? (
        <div
          role="alert"
          className="text-destructive flex items-center gap-3 rounded-lg border p-3 text-sm"
        >
          {t("logs.loadFailed")}
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => void logs.query.refetch()}
          >
            <RefreshCw aria-hidden="true" />
            {t("logs.retry")}
          </Button>
        </div>
      ) : null}
      {logs.clear.isError || directoryAction.isError || exportAction.isError ? (
        <p role="alert" className="text-destructive text-sm">
          {t(
            logs.clear.isError
              ? "logs.clearFailed"
              : directoryAction.isError
                ? "logs.directoryFailed"
                : "logs.exportFailed",
          )}
        </p>
      ) : null}
      {exportAction.data && !exportAction.data.cancelled ? (
        <p role="status" className="text-muted-foreground text-xs">
          {t("logs.exported", {
            count: exportAction.data.count,
            missing: exportAction.data.missing,
          })}
        </p>
      ) : null}
      <div className="bg-card flex min-h-32 min-w-0 flex-1 flex-col overflow-hidden rounded-xl border">
        <div className="text-muted-foreground bg-muted/30 grid shrink-0 grid-cols-[92px_56px_minmax(72px,100px)_minmax(0,1fr)_32px] gap-3 border-b px-3 py-2 text-xs">
          <span>{t("logs.time")}</span>
          <span>{t("logs.level")}</span>
          <span>{t("logs.module")}</span>
          <span>{t("logs.message")}</span>
          <span className="sr-only">{t("logs.actions")}</span>
        </div>
        <ScrollRegion
          as="section"
          className="min-h-0 flex-1"
          ariaLabel={t("logs.list")}
          viewportRef={follow.viewportRef}
          onScroll={follow.onScroll}
        >
          {logs.query.isPending ? (
            <p role="status" className="text-muted-foreground p-8 text-center text-sm">
              {t("logs.loading")}
            </p>
          ) : entries.length ? (
            entries.map((entry) => (
              <LogRow
                key={`${entry.id}:${entry.time}`}
                entry={entry}
                onOpenRequest={onOpenRequest}
              />
            ))
          ) : (
            <p className="text-muted-foreground p-8 text-center text-sm">
              {t(logs.data?.entries.length ? "logs.noMatches" : "logs.empty")}
            </p>
          )}
        </ScrollRegion>
      </div>
      <LogFooter
        dropped={logs.data?.dropped ?? false}
        level={logs.data?.recordedLevel ?? "—"}
        paused={logs.paused}
        following={follow.following}
        pending={pending}
        onJump={() => {
          follow.jump()
          logs.resume()
        }}
      />
    </section>
  )
}
