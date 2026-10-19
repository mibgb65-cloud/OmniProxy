import { CloudDownload, RefreshCw } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card } from "../../components/ui/card"
import { versionChannel } from "./current-release"

export function UpdateVersionCard({
  version,
  checking,
  disabled,
  onCheck,
}: {
  version: string | undefined
  checking: boolean
  disabled: boolean
  onCheck: () => void
}) {
  const { t } = useTranslation()
  const channel = version ? versionChannel(version) : undefined
  return (
    <Card className="grid grid-cols-[auto_minmax(0,1fr)] items-center gap-3 p-4 shadow-none @min-[28rem]/updates:grid-cols-[auto_minmax(0,1fr)_auto]">
      <div className="bg-muted/50 text-muted-foreground flex size-10 items-center justify-center rounded-xl border">
        <CloudDownload className="size-5" strokeWidth={1.5} aria-hidden="true" />
      </div>
      <div className="min-w-0 space-y-1.5">
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1.5">
          <span className="text-sm font-medium">{t("updates.currentVersion")}</span>
          <code className="bg-muted/70 max-w-full rounded-md border px-2 py-0.5 text-sm font-semibold [overflow-wrap:anywhere]">
            {version ?? "…"}
          </code>
          {channel ? (
            <span className="bg-muted/40 text-muted-foreground rounded-md border px-2 py-0.5 text-[0.6875rem] font-medium">
              {t(`updates.${channel}Channel`)}
            </span>
          ) : null}
        </div>
        <p className="text-muted-foreground text-xs leading-relaxed">
          {t("about.update.description")}
        </p>
      </div>
      <Button
        size="sm"
        variant="outline"
        className="col-span-2 @min-[28rem]/updates:col-span-1"
        disabled={disabled}
        onClick={onCheck}
      >
        <RefreshCw
          className={checking ? "motion-safe:animate-spin" : undefined}
          aria-hidden="true"
        />
        {checking ? t("about.update.checking") : t("about.update.check")}
      </Button>
    </Card>
  )
}
