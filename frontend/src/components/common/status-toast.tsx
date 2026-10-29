import { useState, type ReactNode } from "react"
import { createPortal } from "react-dom"
import { Toast } from "radix-ui"
import { CircleCheck, Info, TriangleAlert, X } from "lucide-react"
import { Button } from "../ui/button"

// 沿用全局悬浮反馈样式；Portal 脱离弹窗滚动层，避免被边界裁切。
export function StatusToast({
  title,
  description,
  closeLabel,
  tone = "success",
}: {
  title: string
  description?: ReactNode
  closeLabel: string
  tone?: "success" | "info" | "warning"
}) {
  const [open, setOpen] = useState(true)
  const Icon = tone === "warning" ? TriangleAlert : tone === "info" ? Info : CircleCheck
  return createPortal(
    <div className="pointer-events-none fixed inset-x-4 bottom-4 z-[100]">
      <Toast.Provider duration={5000} label={title}>
        <Toast.Root
          open={open}
          onOpenChange={setOpen}
          type="background"
          data-slot="global-status-toast"
          className="bg-popover text-popover-foreground motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=open]:slide-in-from-bottom-2 motion-safe:data-[state=closed]:animate-out motion-safe:data-[state=closed]:fade-out-0 motion-safe:data-[state=closed]:slide-out-to-bottom-2 pointer-events-auto flex w-full max-w-md items-start gap-3 rounded-xl border p-4 shadow-2xl motion-safe:duration-150"
        >
          <Icon
            className={`mt-0.5 size-5 shrink-0 ${tone === "warning" ? "text-amber-600 dark:text-amber-400" : tone === "info" ? "text-blue-600 dark:text-blue-400" : "text-emerald-600 dark:text-emerald-400"}`}
            aria-hidden="true"
          />
          <div className="min-w-0 flex-1 space-y-1">
            <Toast.Title className="text-sm leading-6 font-medium">{title}</Toast.Title>
            {description ? (
              <Toast.Description className="text-muted-foreground space-y-1 text-xs leading-5">
                {description}
              </Toast.Description>
            ) : null}
          </div>
          <Toast.Close asChild>
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              aria-label={closeLabel}
              className="text-muted-foreground shrink-0 cursor-pointer"
            >
              <X aria-hidden="true" />
            </Button>
          </Toast.Close>
        </Toast.Root>
        <Toast.Viewport
          label={title}
          className="m-0 flex list-none justify-center p-0 outline-none"
        />
      </Toast.Provider>
    </div>,
    document.body,
  )
}
