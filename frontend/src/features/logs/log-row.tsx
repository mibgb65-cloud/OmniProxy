import { Tooltip } from "../../components/ui/tooltip"
import { memo } from "react"
import { useMutation } from "@tanstack/react-query"
import { Clipboard } from "@wailsio/runtime"
import { Check, ChevronRight, Copy } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import type { LogEntry } from "../../services/logs"
import { formatLog } from "./log-stream"

const levelStyles: Record<string, string> = {
  DEBUG: "text-muted-foreground bg-muted",
  INFO: "bg-blue-50 text-blue-700 dark:bg-blue-500/15 dark:text-blue-300",
  WARN: "bg-amber-50 text-amber-800 dark:bg-amber-500/15 dark:text-amber-300",
  ERROR: "bg-red-50 text-red-700 dark:bg-red-500/15 dark:text-red-300",
}

const formatters = {
  zh: new Intl.DateTimeFormat("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    fractionalSecondDigits: 3,
    hourCycle: "h23",
  }),
  en: new Intl.DateTimeFormat("en-GB", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    fractionalSecondDigits: 3,
    hourCycle: "h23",
  }),
}

function logTime(time: string, language: string) {
  const date = new Date(time)
  return Number.isNaN(date.getTime())
    ? "—"
    : formatters[language === "zh" ? "zh" : "en"].format(date)
}

export const LogRow = memo(function LogRow({
  entry,
  onOpenRequest,
}: {
  entry: LogEntry
  onOpenRequest?: (id: string) => void
}) {
  const { t, i18n } = useTranslation()
  const copy = useMutation({ mutationFn: () => Clipboard.SetText(formatLog(entry)) })
  const requestId = entry.fields?.request_id
  return (
    <div
      data-log-id={entry.id}
      className="hover:bg-muted/30 grid min-w-0 grid-cols-[92px_56px_minmax(72px,100px)_minmax(0,1fr)_32px] items-start gap-3 border-b px-3 py-2.5 text-xs last:border-b-0"
    >
      <Tooltip content={entry.time}>
        <time dateTime={entry.time} className="pt-1 font-mono tabular-nums">
          {logTime(entry.time, i18n.resolvedLanguage ?? "zh")}
        </time>
      </Tooltip>
      <span
        className={`${levelStyles[entry.level] ?? levelStyles.INFO} mt-0.5 rounded px-1.5 py-1 text-center text-[10px] font-semibold`}
      >
        {entry.level}
      </span>
      <span className="text-muted-foreground pt-1 wrap-anywhere">{entry.module}</span>
      <details className="group min-w-0 pt-1">
        <summary className="focus-visible:ring-ring/50 flex cursor-pointer list-none items-start gap-1.5 rounded outline-none focus-visible:ring-2 [&::-webkit-details-marker]:hidden">
          <ChevronRight
            className="mt-0.5 size-3 shrink-0 group-open:rotate-90"
            aria-hidden="true"
          />
          <span className="min-w-0 font-mono leading-5 wrap-anywhere whitespace-pre-wrap">
            {entry.message}
          </span>
        </summary>
        <dl className="bg-muted/30 mt-2 space-y-1 rounded-md border p-3 leading-5">
          {onOpenRequest && requestId && /^[a-f0-9]{32}$/.test(requestId) ? (
            <Button size="sm" variant="outline" onClick={() => onOpenRequest(requestId)}>
              {t("activity.request")}
            </Button>
          ) : null}
          <div>
            <dt className="text-muted-foreground">{t("logs.time")}</dt>
            <dd className="font-mono wrap-anywhere">{entry.time}</dd>
          </div>
          {Object.entries(entry.fields ?? {})
            .sort(([a], [b]) => a.localeCompare(b))
            .map(([key, value]) => (
              <div key={key}>
                <dt className="text-muted-foreground font-mono wrap-anywhere">{key}</dt>
                <dd className="font-mono wrap-anywhere whitespace-pre-wrap">{value}</dd>
              </div>
            ))}
        </dl>
      </details>
      <div>
        <Tooltip content={copy.isError ? t("logs.copyFailed") : undefined}>
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            disabled={copy.isPending}
            aria-label={t("logs.copyEntry", { id: entry.id })}
            className={copy.isError ? "text-destructive" : undefined}
            onClick={() => copy.mutate()}
          >
            {copy.isSuccess ? (
              <Check className="size-3.5" aria-hidden="true" />
            ) : (
              <Copy className="size-3.5" aria-hidden="true" />
            )}
          </Button>
        </Tooltip>
        {copy.isSuccess ? (
          <span role="status" className="sr-only">
            {t("logs.copied")}
          </span>
        ) : null}
        {copy.isError ? (
          <span role="alert" className="sr-only">
            {t("logs.copyFailed")}
          </span>
        ) : null}
      </div>
    </div>
  )
})
