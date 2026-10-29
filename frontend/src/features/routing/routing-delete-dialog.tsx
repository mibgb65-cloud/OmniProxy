import { AlertDialog } from "radix-ui"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { useLastValue } from "../../lib/use-last-value"
import type { RouteItem } from "../../services/routing"

export function RoutingDeleteDialog({
  item,
  busy,
  onCancel,
  onConfirm,
  onRestoreFocus,
  error,
}: {
  item: RouteItem | null
  busy: boolean
  onCancel: () => void
  onConfirm: () => void
  onRestoreFocus: () => void
  error: boolean
}) {
  const { t } = useTranslation()
  // 关闭动画期间仍要显示被删除规则的别名，保留最后一次打开的目标。
  const target = useLastValue(item)
  return (
    <AlertDialog.Root
      open={item !== null}
      onOpenChange={(open) => {
        if (!open && !busy) onCancel()
      }}
    >
      <AlertDialog.Portal>
        <AlertDialog.Overlay className="motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=closed]:animate-out motion-safe:data-[state=closed]:fade-out-0 fixed inset-0 z-40 bg-black/50 motion-safe:duration-150" />
        <AlertDialog.Content
          onCloseAutoFocus={(event) => {
            event.preventDefault()
            onRestoreFocus()
          }}
          className="bg-card text-card-foreground motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=open]:zoom-in-95 motion-safe:data-[state=closed]:animate-out motion-safe:data-[state=closed]:fade-out-0 motion-safe:data-[state=closed]:zoom-out-95 fixed top-1/2 left-1/2 z-50 w-[min(420px,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 space-y-4 rounded-xl border p-6 shadow-xl motion-safe:duration-150"
        >
          <AlertDialog.Title className="text-base font-semibold">
            {t("routing.deleteTitle")}
          </AlertDialog.Title>
          <AlertDialog.Description className="text-muted-foreground text-sm">
            {t("routing.deleteDescription", { alias: target?.rule.alias ?? "" })}
          </AlertDialog.Description>
          {error ? (
            <p role="alert" className="text-destructive text-sm">
              {t("routing.actionFailed")}
            </p>
          ) : null}
          <div className="flex justify-end gap-2">
            <AlertDialog.Cancel asChild>
              <Button variant="outline" disabled={busy}>
                {t("routing.cancel")}
              </Button>
            </AlertDialog.Cancel>
            <Button variant="destructive" disabled={busy} onClick={onConfirm}>
              {t(busy ? "routing.deleting" : "routing.deleteConfirm")}
            </Button>
          </div>
        </AlertDialog.Content>
      </AlertDialog.Portal>
    </AlertDialog.Root>
  )
}
