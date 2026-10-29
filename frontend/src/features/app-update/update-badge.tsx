import { ArrowUp, Download } from "lucide-react"
import { Dialog } from "radix-ui"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Tooltip } from "../../components/ui/tooltip"
import { cn } from "../../lib/utils"
import { useApplicationUpdate } from "./use-application-update"
import { UpdateDialog } from "./update-dialog"

export function UpdateBadge({ collapsed, onOpen }: { collapsed: boolean; onOpen: () => void }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const query = useApplicationUpdate()
  const state = query.data
  if (!state?.release) return null
  const active = ["downloading", "verifying", "installing"].includes(state.download.state)
  return (
    <Dialog.Root
      open={open}
      onOpenChange={(value) => {
        if (value) onOpen()
        setOpen(value)
      }}
    >
      <Tooltip content={t("appUpdate.badgeLabel", { version: state.release.version })} side="right">
        <Dialog.Trigger asChild>
          <Button
            variant="secondary"
            size="sm"
            className={cn(
              "text-primary bg-primary/10 hover:bg-primary/20 h-7 shrink-0 gap-1 rounded-md px-2 text-xs",
              collapsed && "absolute -top-1 right-0 size-5 p-0",
            )}
            aria-label={t("appUpdate.badgeLabel", { version: state.release.version })}
          >
            {active ? (
              <Download className="size-3.5" aria-hidden="true" />
            ) : (
              <ArrowUp className="size-3.5" aria-hidden="true" />
            )}
            {!collapsed ? (
              <span>{t(active ? "appUpdate.badgeProgress" : "appUpdate.badge")}</span>
            ) : null}
          </Button>
        </Dialog.Trigger>
      </Tooltip>
      {/* 常驻浮窗：退出动画依赖 Radix 的关闭态，条件挂载会瞬间消失。 */}
      <UpdateDialog state={state} />
    </Dialog.Root>
  )
}
