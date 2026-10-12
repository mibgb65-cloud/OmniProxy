import { ArrowDown } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"

export function LogFooter({
  dropped,
  level,
  paused,
  following,
  pending,
  onJump,
}: {
  dropped: boolean
  level: string
  paused: boolean
  following: boolean
  pending: number
  onJump: () => void
}) {
  const { t } = useTranslation()
  return (
    <footer className="text-muted-foreground flex shrink-0 flex-wrap items-center justify-between gap-2 text-xs">
      <span>{t(dropped ? "logs.evicted" : "logs.retention", { level })}</span>
      {paused || !following ? (
        <Button type="button" size="sm" variant="outline" onClick={onJump}>
          <ArrowDown aria-hidden="true" />
          {pending ? t("logs.newEntries", { count: pending }) : t("logs.followLatest")}
        </Button>
      ) : (
        <span>{t("logs.following")}</span>
      )}
    </footer>
  )
}
