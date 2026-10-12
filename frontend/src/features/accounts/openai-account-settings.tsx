import { Dialog } from "radix-ui"
import { Settings2, X } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Tooltip } from "../../components/ui/tooltip"

export function OpenAIAccountSettings() {
  const { t } = useTranslation()
  return (
    <Dialog.Root>
      <Tooltip content={t("subscription.settings.title")}>
        <Dialog.Trigger asChild>
          <Button
            type="button"
            variant="outline"
            size="icon-sm"
            aria-label={t("subscription.settings.title")}
            className="cursor-pointer"
          >
            <Settings2 className="size-4" aria-hidden="true" />
          </Button>
        </Dialog.Trigger>
      </Tooltip>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm" />
        <Dialog.Content
          aria-modal={true}
          aria-describedby={undefined}
          className="bg-popover text-popover-foreground fixed top-1/2 left-1/2 z-50 flex h-[min(560px,calc(100dvh-3rem))] w-[min(800px,calc(100vw-3rem))] -translate-x-1/2 -translate-y-1/2 flex-col rounded-xl border p-6 shadow-2xl outline-none"
        >
          <div className="flex items-center justify-between gap-3 border-b pb-3">
            <Dialog.Title className="text-base font-semibold">
              {t("subscription.settings.title")}
            </Dialog.Title>
            <Dialog.Close asChild>
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                aria-label={t("subscription.settings.close")}
                className="cursor-pointer"
              >
                <X className="size-3.5" aria-hidden="true" />
              </Button>
            </Dialog.Close>
          </div>
          <div className="min-h-0 flex-1" />
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
