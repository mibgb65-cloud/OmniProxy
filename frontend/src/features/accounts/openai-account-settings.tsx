import { useId } from "react"
import { Popover } from "radix-ui"
import { Settings2, X } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Tooltip } from "../../components/ui/tooltip"
import { CodexWebSocketSettings } from "./codex-websocket-settings"
import { CodexQuotaActivationSettings } from "./codex-quota-activation-settings"

export function OpenAIAccountSettings() {
  const { t } = useTranslation()
  const titleId = useId()
  return (
    <Popover.Root modal={false}>
      <Tooltip content={t("subscription.settings.title")}>
        <Popover.Trigger asChild>
          <Button
            type="button"
            variant="outline"
            size="icon-sm"
            aria-label={t("subscription.settings.title")}
            className="cursor-pointer"
          >
            <Settings2 className="size-4" aria-hidden="true" />
          </Button>
        </Popover.Trigger>
      </Tooltip>
      <Popover.Portal>
        <Popover.Content
          side="bottom"
          align="start"
          sideOffset={8}
          collisionPadding={12}
          sticky="always"
          aria-labelledby={titleId}
          aria-describedby={undefined}
          className="bg-popover text-popover-foreground z-50 flex max-h-[min(560px,var(--radix-popover-content-available-height))] w-[min(420px,calc(100vw-1.5rem))] flex-col overflow-hidden rounded-xl border p-4 shadow-xl outline-none"
        >
          <div className="flex shrink-0 items-center justify-between gap-3 border-b pb-3">
            <h2 id={titleId} className="text-base font-semibold">
              {t("subscription.settings.title")}
            </h2>
            <Popover.Close asChild>
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                aria-label={t("subscription.settings.close")}
                className="cursor-pointer"
              >
                <X className="size-3.5" aria-hidden="true" />
              </Button>
            </Popover.Close>
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto pr-1">
            <CodexWebSocketSettings />
            <CodexQuotaActivationSettings />
          </div>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}
