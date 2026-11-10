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
  const canJump = paused || !following
  return (
    <footer className="text-muted-foreground flex shrink-0 flex-wrap items-center justify-between gap-2 text-xs">
      <span className="min-w-0">{t(dropped ? "logs.evicted" : "logs.retention", { level })}</span>
      {/* 右侧槽位固定为按钮高度：小号按钮 32px、纯文本仅 16px，若让二者各占各的高度，
          跟随状态切换时页脚会收紧／放松，挤压 flex-1 的日志列表并造成整页跳动。 */}
      <div data-slot="log-follow-slot" className="flex h-8 shrink-0 items-center">
        {canJump ? (
          <Button type="button" size="sm" variant="outline" onClick={onJump}>
            <ArrowDown aria-hidden="true" />
            {pending ? t("logs.newEntries", { count: pending }) : t("logs.followLatest")}
          </Button>
        ) : (
          <span>{t("logs.following")}</span>
        )}
      </div>
    </footer>
  )
}
