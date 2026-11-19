import { useMutation, useQuery } from "@tanstack/react-query"
import { Clipboard } from "@wailsio/runtime"
import { Copy, FileJson } from "lucide-react"
import { useMemo } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Tooltip } from "../../components/ui/tooltip"
import { readRequestContent } from "../../services/activity"
import { formatBytes } from "./activity-format"

/** 正文通常是 JSON；解析失败时按原文展示，不隐藏或改写用户内容。 */
function prettyContent(value: string) {
  try {
    return JSON.stringify(JSON.parse(value), null, 2)
  } catch {
    return value
  }
}

export function RequestContentSection({ id }: { id: string }) {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: ["request-content", id],
    queryFn: () => readRequestContent(id),
    enabled: !!id,
    retry: false,
  })
  const copy = useMutation({ mutationFn: (value: string) => Clipboard.SetText(value) })
  const data = query.data
  const shown = useMemo(() => (data?.available ? prettyContent(data.content) : ""), [data])

  return (
    <section className="space-y-2 rounded-lg border p-3">
      <div className="flex items-center justify-between gap-3">
        <h3 className="flex items-center gap-2 text-sm font-medium">
          <FileJson className="size-4" aria-hidden="true" />
          {t("activity.contentTitle")}
        </h3>
        {data?.available ? (
          <Tooltip content={t(copy.isSuccess ? "activity.copied" : "activity.contentCopy")}>
            <Button
              variant="ghost"
              size="icon-sm"
              aria-label={t("activity.contentCopy")}
              disabled={copy.isPending}
              onClick={() => copy.mutate(data.content)}
            >
              <Copy aria-hidden="true" />
            </Button>
          </Tooltip>
        ) : null}
      </div>
      {query.isPending ? (
        <p role="status" className="text-muted-foreground text-sm">
          {t("activity.contentLoading")}
        </p>
      ) : query.isError ? (
        <p role="alert" className="text-destructive text-sm">
          {t("activity.contentFailed")}
        </p>
      ) : data?.available ? (
        <>
          <p className="text-muted-foreground text-xs leading-5">
            {t("activity.contentSize", {
              stored: formatBytes(data.storedBytes),
              original: formatBytes(data.originalBytes),
            })}
          </p>
          {data.truncated ? (
            <p role="status" className="text-xs text-amber-600 dark:text-amber-400">
              {t("activity.contentTruncated", { original: formatBytes(data.originalBytes) })}
            </p>
          ) : null}
          <pre className="bg-muted/30 max-h-64 overflow-auto rounded-lg border p-3 font-mono text-xs break-words whitespace-pre-wrap">
            {shown}
          </pre>
        </>
      ) : (
        <>
          <p className="text-muted-foreground text-sm">{t("activity.contentMissing")}</p>
          <p className="text-muted-foreground text-xs leading-5">
            {t("activity.contentMissingHint")}
          </p>
        </>
      )}
    </section>
  )
}
