import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Tooltip } from "../../components/ui/tooltip"
import { stateLabel, requestTime } from "./activity-format"
import { formatTokens, type RequestItem } from "../../services/activity"

export function RequestTable({
  items,
  onSelect,
}: {
  items: RequestItem[]
  onSelect: (id: string) => void
}) {
  const { t, i18n } = useTranslation()
  return (
    <div className="overflow-x-auto rounded-xl border">
      <table className="w-full text-left text-sm">
        <thead className="bg-muted/30 text-muted-foreground text-xs">
          <tr>
            {(
              [
                "activity.time",
                "activity.model",
                "activity.account",
                "activity.state",
                "activity.input",
                "activity.output",
                "activity.latency",
              ] as const
            ).map((key) => (
              <th key={key} className="px-4 py-3 font-medium">
                {t(key)}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
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
                {!item.usageComplete ? (
                  <p className="text-muted-foreground mt-1 text-[10px]">
                    {t(
                      item.inputKnown || item.outputKnown ? "activity.partial" : "activity.unknown",
                    )}
                  </p>
                ) : null}
              </td>
              <td className="px-4 py-3 font-mono text-xs tabular-nums">
                {item.inputKnown ? formatTokens(item.inputTokens, i18n.resolvedLanguage) : "—"}
              </td>
              <td className="px-4 py-3 font-mono text-xs tabular-nums">
                {item.outputKnown ? formatTokens(item.outputTokens, i18n.resolvedLanguage) : "—"}
              </td>
              <td className="px-4 py-3 font-mono text-xs whitespace-nowrap">
                {item.state === "pending" || item.errorCode === "process_interrupted"
                  ? "—"
                  : `${item.latencyMs} ms`}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
