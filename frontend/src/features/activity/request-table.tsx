import { CircleHelp } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Tooltip } from "../../components/ui/tooltip"
import {
  stateLabel,
  requestTime,
  formatCachedHit,
  formatCost,
  formatSpeed,
  formatFirstToken,
} from "./activity-format"
import { agentTitleKey } from "../gateway/agent-clients"
import { formatTokens, type RequestItem } from "../../services/activity"
import type { TranslationKey } from "../../i18n/translation"

type Column = { key: TranslationKey; hint?: TranslationKey }

const columns: readonly Column[] = [
  { key: "activity.time" },
  { key: "activity.model" },
  { key: "activity.agent" },
  { key: "activity.account" },
  { key: "activity.state" },
  { key: "activity.input" },
  { key: "activity.output" },
  { key: "activity.cached", hint: "activity.cachedHint" },
  { key: "activity.cost", hint: "activity.costHint" },
  { key: "activity.speed", hint: "activity.speedHint" },
  { key: "activity.firstToken", hint: "activity.firstTokenHint" },
  { key: "activity.latency" },
]

function ColumnHint({ hint, label }: { hint: TranslationKey; label: string }) {
  const { t } = useTranslation()
  return (
    <Tooltip content={t(hint)}>
      <button
        type="button"
        aria-label={t("activity.columnHint", { column: label })}
        className="text-muted-foreground/70 hover:text-foreground focus-visible:ring-ring inline-flex size-4 items-center justify-center rounded-full transition-colors focus-visible:ring-2 focus-visible:outline-none"
      >
        <CircleHelp className="size-3" aria-hidden="true" />
      </button>
    </Tooltip>
  )
}

export function RequestTable({
  items,
  onSelect,
}: {
  items: RequestItem[]
  onSelect: (id: string) => void
}) {
  const { t, i18n } = useTranslation()
  const language = i18n.resolvedLanguage
  return (
    <div className="overflow-x-auto rounded-xl border">
      <table className="w-full text-left text-sm">
        <thead className="bg-muted/30 text-muted-foreground text-xs">
          <tr>
            {columns.map(({ key, hint }) => (
              <th key={key} className="px-4 py-3 font-medium">
                <span className="inline-flex items-center gap-1 whitespace-nowrap">
                  {t(key)}
                  {hint ? <ColumnHint hint={hint} label={t(key)} /> : null}
                </span>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {items.map((item) => {
            const agentKey = agentTitleKey(item.client)
            const cachedHit = formatCachedHit(
              item.cachedInputTokens,
              item.cachedInputKnown,
              item.inputTokens,
              item.inputKnown,
            )
            return (
              <tr key={item.id} className="hover:bg-muted/20 border-t">
                <td className="px-4 py-3">
                  <Button
                    variant="link"
                    className="h-auto p-0 text-xs"
                    aria-label={t("activity.openDetails", { id: item.id })}
                    onClick={() => onSelect(item.id)}
                  >
                    {requestTime(item.startedAt)}
                  </Button>
                  <p className="text-muted-foreground mt-1 font-mono text-[10px]">
                    {item.id.slice(0, 12)}
                  </p>
                </td>
                <td className="px-4 py-3 font-mono text-xs">
                  <Tooltip content={item.model || item.inputModel}>
                    <span className="block max-w-48 truncate">
                      {item.model || item.inputModel || "—"}
                    </span>
                  </Tooltip>
                </td>
                <td className="px-4 py-3">
                  <span className="block max-w-32 truncate text-xs">
                    {agentKey ? t(agentKey) : "—"}
                  </span>
                </td>
                <td className="px-4 py-3">
                  <span className="block max-w-52 truncate text-xs">
                    {item.email || item.accountName || t("activity.noAccount")}
                  </span>
                  {item.workspaceId ? (
                    <Tooltip content={item.workspaceId}>
                      <span className="text-muted-foreground font-mono text-[10px]">
                        {item.workspaceId.slice(0, 8)}…
                      </span>
                    </Tooltip>
                  ) : null}
                  {item.accountDeleted ? (
                    <span className="text-muted-foreground block text-[10px]">
                      {t("activity.deleted")}
                    </span>
                  ) : null}
                </td>
                <td className="px-4 py-3">
                  <span
                    className={`rounded-full px-2 py-1 text-xs whitespace-nowrap ${item.state === "success" ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400" : item.state === "failed" ? "bg-red-500/10 text-red-600 dark:text-red-400" : "bg-muted text-muted-foreground"}`}
                  >
                    {t(stateLabel(item.state))}
                  </span>
                  {(item.attemptCount ?? 0) > 1 ? (
                    <p className="text-muted-foreground mt-1 text-xs">
                      {t("failover.switched", { count: (item.attemptCount ?? 1) - 1 })}
                    </p>
                  ) : null}
                  {!item.usageComplete ? (
                    <p className="text-muted-foreground mt-1 text-[10px]">
                      {t(
                        item.inputKnown || item.outputKnown
                          ? "activity.partial"
                          : "activity.unknown",
                      )}
                    </p>
                  ) : null}
                </td>
                <td className="px-4 py-3 font-mono text-xs tabular-nums">
                  {item.inputKnown ? formatTokens(item.inputTokens, language) : "—"}
                </td>
                <td className="px-4 py-3 font-mono text-xs tabular-nums">
                  {item.outputKnown ? formatTokens(item.outputTokens, language) : "—"}
                </td>
                <td className="px-4 py-3 font-mono text-xs tabular-nums">
                  {cachedHit === "—" ? (
                    "—"
                  ) : (
                    <Tooltip
                      content={t("activity.cachedDetail", {
                        cached: formatTokens(item.cachedInputTokens, language),
                        input: formatTokens(item.inputTokens, language),
                      })}
                    >
                      <span className="cursor-default">{cachedHit}</span>
                    </Tooltip>
                  )}
                </td>
                <td className="px-4 py-3 font-mono text-xs whitespace-nowrap tabular-nums">
                  {formatCost(item.costMin, item.costMax, item.costKnown)}
                </td>
                <td className="px-4 py-3 font-mono text-xs whitespace-nowrap tabular-nums">
                  {formatSpeed(item)}
                </td>
                <td className="px-4 py-3 font-mono text-xs whitespace-nowrap tabular-nums">
                  {formatFirstToken(item.firstTokenMs, item.state)}
                </td>
                <td className="px-4 py-3 font-mono text-xs whitespace-nowrap">
                  {item.state === "pending" || item.errorCode === "process_interrupted"
                    ? "—"
                    : `${item.latencyMs} ms`}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
