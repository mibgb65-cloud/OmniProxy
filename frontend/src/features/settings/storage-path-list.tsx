import { Tooltip } from "../../components/ui/tooltip"
import { Clipboard } from "@wailsio/runtime"
import { Check, Copy } from "lucide-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import type { StorageInfo } from "../../services/omniproxy"

export function StoragePathList({ data }: { data: StorageInfo }) {
  const { t } = useTranslation()
  const [feedback, setFeedback] = useState<{
    path: string
    message: string
    error: boolean
  } | null>(null)
  const rows = [
    { label: t("settings.storage.dataDir"), path: data.dataDir },
    { label: t("settings.storage.logDir"), path: data.logDir },
    { label: t("settings.storage.databasePath"), path: data.databasePath },
  ]

  async function copyPath(path: string, label: string) {
    try {
      await Clipboard.SetText(path)
      setFeedback({ path, message: t("settings.storage.copiedPath", { label }), error: false })
    } catch {
      setFeedback({ path, message: t("settings.storage.copyFailed", { label }), error: true })
    }
  }

  return (
    <div className="space-y-2.5">
      <dl className="space-y-2.5">
        {rows.map(({ label, path }) => (
          <div
            key={label}
            className="bg-muted/15 flex min-w-0 items-center gap-2 rounded-md border px-3 py-2"
          >
            <div className="flex min-w-0 flex-1 flex-col gap-1 sm:flex-row sm:items-center sm:gap-3">
              <dt className="text-muted-foreground shrink-0 text-xs sm:w-20">{label}</dt>
              <Tooltip content={path}>
                <dd className="min-w-0 flex-1 truncate font-mono text-xs">{path}</dd>
              </Tooltip>
            </div>
            <Tooltip content={t("settings.storage.copyPath", { label })}>
              <Button
                type="button"
                size="icon-sm"
                variant="outline"
                className="text-muted-foreground hover:text-foreground size-9"
                aria-label={t("settings.storage.copyPath", { label })}
                onClick={() => void copyPath(path, label)}
              >
                {feedback?.path === path && !feedback.error ? (
                  <Check aria-hidden="true" />
                ) : (
                  <Copy aria-hidden="true" />
                )}
              </Button>
            </Tooltip>
          </div>
        ))}
      </dl>
      {feedback ? (
        <p
          role={feedback.error ? "alert" : "status"}
          className={feedback.error ? "text-destructive text-xs" : "text-muted-foreground text-xs"}
        >
          {feedback.message}
        </p>
      ) : null}
    </div>
  )
}
