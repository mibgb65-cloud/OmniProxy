import { useState } from "react"
import { Dialog } from "radix-ui"
import { Tickets, X } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Tooltip } from "../../components/ui/tooltip"
import type { AccountItem } from "../../services/omniproxy"
import { useSubscriptionQuota } from "./use-subscription-quota"
import { accountDisplayName } from "./account-display"
import { ResetCreditsContent } from "./reset-credits-content"

export function ResetCreditsDialog({
  account,
  disabled,
}: {
  account: AccountItem
  disabled: boolean
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const quota = useSubscriptionQuota(account)
  const count = quota.data?.resetCredits?.availableCount
  return (
    <Dialog.Root
      open={open}
      onOpenChange={(value) => {
        if (!busy) setOpen(value)
      }}
    >
      <Tooltip content={t("subscription.reset.title")}>
        <Dialog.Trigger asChild>
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
        </Dialog.Trigger>
      </Tooltip>
      {open ? (
        <Dialog.Portal>
          <Dialog.Overlay className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm" />
          <Dialog.Content
            aria-modal={true}
            className="bg-popover text-popover-foreground fixed top-1/2 left-1/2 z-50 flex h-[min(620px,calc(100dvh-3rem))] w-[min(720px,calc(100vw-3rem))] -translate-x-1/2 -translate-y-1/2 flex-col gap-4 rounded-xl border p-6 shadow-2xl outline-none"
            onEscapeKeyDown={(event) => {
              if (busy) event.preventDefault()
            }}
            onInteractOutside={(event) => {
              if (busy) event.preventDefault()
            }}
          >
            <div className="flex items-start justify-between gap-3 border-b pb-3">
              <div>
                <Dialog.Title className="font-semibold">
                  {t("subscription.reset.title")}
                </Dialog.Title>
                <Dialog.Description className="text-muted-foreground mt-1 text-xs break-all">
                  {accountDisplayName(account)}
                </Dialog.Description>
              </div>
              <Dialog.Close asChild>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  disabled={busy}
                  aria-label={t("subscription.reset.close")}
                >
                  <X className="size-4" aria-hidden="true" />
                </Button>
              </Dialog.Close>
            </div>
            <ResetCreditsContent account={account} onBusyChange={setBusy} />
          </Dialog.Content>
        </Dialog.Portal>
      ) : null}
    </Dialog.Root>
  )
}
