import { useState } from "react"
import { Toast } from "radix-ui"
import { CircleCheck, X } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"

export function AgentConfigSuccessToast() {
  const { t } = useTranslation()
  const [open, setOpen] = useState(true)
  return (
    <div className="pointer-events-none absolute inset-x-4 top-4 z-[100]">
      <Toast.Provider duration={5000} label={t("agents.successNotification")}>
        <Toast.Root
          open={open}
          onOpenChange={setOpen}
          type="background"
          data-slot="agent-config-success-toast"
          className="bg-popover text-popover-foreground motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=open]:slide-in-from-top-2 motion-safe:data-[state=closed]:animate-out motion-safe:data-[state=closed]:fade-out-0 pointer-events-auto flex w-full max-w-md items-center gap-3 rounded-xl border border-emerald-500/40 p-4 shadow-2xl"
        >
          <CircleCheck
            className="size-6 shrink-0 text-emerald-600 dark:text-emerald-400"
            aria-hidden="true"
          />
          <Toast.Title className="min-w-0 flex-1 text-sm leading-6 font-medium">
            {t("agents.success")}
          </Toast.Title>
          <Toast.Close asChild>
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              aria-label={t("agents.dismissSuccess")}
              className="text-muted-foreground shrink-0 cursor-pointer"
            >
              <X aria-hidden="true" />
            </Button>
          </Toast.Close>
        </Toast.Root>
        {/* 位于弹窗非滚动层，既不会随目录滚走，也保留模态框内的焦点和播报语义。 */}
        <Toast.Viewport
          label={t("agents.successNotification")}
          className="m-0 flex list-none justify-center p-0 outline-none"
        />
      </Toast.Provider>
    </div>
  )
}
