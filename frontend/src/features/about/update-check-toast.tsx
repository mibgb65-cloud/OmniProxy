import { CircleCheck, X } from "lucide-react"
import { Toast } from "radix-ui"
import { useState } from "react"
import { createPortal } from "react-dom"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { cn } from "../../lib/utils"

export function UpdateCheckToast({ container }: { container?: HTMLElement | null }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(true)
  const notice = (
    <div
      className={cn(
        "pointer-events-none inset-x-4 bottom-4 z-[100]",
        container ? "absolute" : "fixed",
      )}
    >
      <Toast.Provider duration={5000} label={t("updates.checkNotice")}>
        <Toast.Root
          open={open}
          onOpenChange={setOpen}
          type="background"
          data-slot="update-check-toast"
          className="bg-popover text-popover-foreground motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=open]:slide-in-from-bottom-2 motion-safe:data-[state=closed]:animate-out motion-safe:data-[state=closed]:fade-out-0 pointer-events-auto flex w-full max-w-sm items-center gap-3 rounded-xl border p-3 shadow-xl"
        >
          <CircleCheck
            className="size-5 shrink-0 text-emerald-600 dark:text-emerald-400"
            aria-hidden="true"
          />
          <Toast.Title className="min-w-0 flex-1 text-sm leading-6 font-medium">
            {t("about.update.upToDate")}
          </Toast.Title>
          <Toast.Close asChild>
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              aria-label={t("updates.dismissNotice")}
              className="text-muted-foreground shrink-0"
            >
              <X aria-hidden="true" />
            </Button>
          </Toast.Close>
        </Toast.Root>
        <Toast.Viewport
          label={t("updates.checkNotice")}
          className="m-0 flex list-none justify-center p-0 outline-none"
        />
      </Toast.Provider>
    </div>
  )
  // 留在对话框焦点与可访问性边界内，但脱离内容滚动层，长日志不会裁切提示。
  return container ? createPortal(notice, container) : notice
}
