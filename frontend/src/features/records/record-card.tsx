import { useTranslation } from "react-i18next"
import { formatBytes, requestTime, stateLabel } from "../activity/activity-format"
import { requestAgentTitleKey } from "../gateway/agent-clients"
import type { RecordedItem } from "../../services/activity"

/** 一条已保存正文的摘要卡片；点击打开对应请求详情。 */
export function RecordCard({ item, onOpen }: { item: RecordedItem; onOpen: (id: string) => void }) {
  const { t } = useTranslation()
  const agentKey = requestAgentTitleKey(item.client)
  const time = requestTime(item.startedAt)
  const stateClass =
    item.state === "success"
      ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
      : item.state === "failed"
        ? "bg-red-500/10 text-red-600 dark:text-red-400"
        : "bg-muted text-muted-foreground"
  return (
    <button
      type="button"
      aria-label={t("records.openRecord", { time })}
      onClick={() => onOpen(item.id)}
      className="hover:bg-muted/20 focus-visible:ring-ring w-full rounded-xl border p-4 text-left transition-colors focus-visible:ring-2 focus-visible:outline-none"
    >
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="text-sm font-medium tabular-nums">{time}</p>
          <p className="text-muted-foreground mt-1 font-mono text-[10px]">{item.id.slice(0, 12)}</p>
        </div>
        <div className="flex flex-wrap items-center gap-2 text-xs">
          <span className={`rounded-full px-2 py-1 whitespace-nowrap ${stateClass}`}>
            {t(stateLabel(item.state))}
          </span>
          {item.truncated ? (
            <span className="rounded-full bg-amber-500/10 px-2 py-1 whitespace-nowrap text-amber-600 dark:text-amber-400">
              {t("records.truncated")}
            </span>
          ) : null}
          <span className="text-muted-foreground tabular-nums">
            {t("records.size", {
              original: formatBytes(item.originalBytes),
              stored: formatBytes(item.storedBytes),
            })}
          </span>
        </div>
      </div>
      <p className="mt-3 line-clamp-3 text-sm break-words">
        {item.preview || t("records.noPreview")}
      </p>
      <div className="text-muted-foreground mt-3 flex flex-wrap items-center justify-between gap-2 text-xs">
        <span className="min-w-0 truncate font-mono">
          {item.model || "—"} · {agentKey ? t(agentKey) : "—"} · {item.accountName || "—"}
        </span>
        <span className="text-primary shrink-0 font-medium">{t("records.openRequest")}</span>
      </div>
    </button>
  )
}
