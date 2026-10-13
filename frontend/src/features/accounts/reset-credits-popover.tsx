import { useId, useState } from "react"
import { Popover } from "radix-ui"
import { Tickets, X } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Tooltip } from "../../components/ui/tooltip"
import type { AccountItem } from "../../services/omniproxy"
import { useSubscriptionQuota } from "./use-subscription-quota"
import { accountDisplayName } from "./account-display"
import { ResetCreditsContent } from "./reset-credits-content"

export function ResetCreditsPopover({
  account,
  disabled,
}: {
  account: AccountItem
  disabled: boolean
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const titleId = useId()
  const descriptionId = useId()
  const quota = useSubscriptionQuota(account)
  const count = quota.data?.resetCredits?.availableCount
  return (
    <Popover.Root
      modal={false}
      open={open}
      onOpenChange={(value) => {
        if (!busy) setOpen(value)
      }}
    >
      <Tooltip content={t("subscription.reset.title")}>
        <Popover.Trigger asChild>
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label={t("subscription.reset.title")}
            disabled={
              disabled ||
              account.authState === "logged_out" ||
              account.authState === "reauth_required"
            }
            className="hover:text-foreground relative cursor-pointer"
          >
            <Tickets className="size-4" aria-hidden="true" />
            {typeof count === "number" && count > 0 ? (
              <span className="bg-primary text-primary-foreground absolute -top-1 -right-1 min-w-4 rounded-full px-1 text-[9px] leading-4 tabular-nums">
                {count > 99 ? "99+" : count}
              </span>
            ) : null}
          </Button>
        </Popover.Trigger>
      </Tooltip>
      {open ? (
        <Popover.Portal>
          <Popover.Content
            side="bottom"
            align="start"
            sideOffset={8}
            collisionPadding={12}
            sticky="always"
            aria-labelledby={titleId}
            aria-describedby={descriptionId}
            className="bg-popover text-popover-foreground z-50 flex h-[440px] max-h-[var(--radix-popover-content-available-height)] w-[min(360px,calc(100vw-1.5rem))] flex-col gap-3 overflow-hidden rounded-xl border p-4 shadow-xl outline-none"
            onEscapeKeyDown={(event) => {
              if (busy) event.preventDefault()
            }}
            onInteractOutside={(event) => {
              if (busy) event.preventDefault()
            }}
          >
            <div className="flex items-start justify-between gap-3 border-b pb-3">
              <div className="min-w-0">
                <h3 id={titleId} className="font-semibold">
                  {t("subscription.reset.title")}
                </h3>
                <p id={descriptionId} className="text-muted-foreground mt-1 text-xs break-all">
                  {accountDisplayName(account)}
                </p>
              </div>
              <Popover.Close asChild>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  disabled={busy}
                  aria-label={t("subscription.reset.close")}
                >
                  <X className="size-4" aria-hidden="true" />
                </Button>
              </Popover.Close>
            </div>
            <ResetCreditsContent account={account} onBusyChange={setBusy} />
          </Popover.Content>
        </Popover.Portal>
      ) : null}
    </Popover.Root>
  )
}
