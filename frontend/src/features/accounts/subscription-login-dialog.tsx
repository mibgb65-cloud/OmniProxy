import { useId, useState } from "react"
import { Dialog } from "radix-ui"
import { ArrowUpRight, LoaderCircle, LockKeyhole, LogIn, X } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Input } from "../../components/ui/input"
import { Tooltip } from "../../components/ui/tooltip"
import { validAccountName } from "./account-input"
import { useSubscriptionLogin } from "./use-subscription-login"
import { SubscriptionLoginStatus } from "./subscription-login-status"
import { SubscriptionLoginMethod } from "./subscription-login-method"
import { ProviderBrandIcon } from "../providers/provider-brand-icon"

export function SubscriptionLoginDialog({
  providerId,
  accountId,
  disabled = false,
}: {
  providerId: string
  accountId?: string
  disabled?: boolean
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const label = t(accountId ? "subscription.relogin" : "subscription.add")
  const trigger = (
    <Dialog.Trigger asChild>
      <Button
        type="button"
        variant={accountId ? "ghost" : "outline"}
        size={accountId ? "icon-sm" : "sm"}
        disabled={disabled}
        aria-label={label}
        className="hover:text-foreground cursor-pointer"
      >
        {accountId ? <LogIn className="size-4" aria-hidden="true" /> : label}
      </Button>
    </Dialog.Trigger>
  )
  return (
    <Dialog.Root open={open} onOpenChange={setOpen}>
      {accountId ? <Tooltip content={label}>{trigger}</Tooltip> : trigger}
      {open ? <LoginContent providerId={providerId} accountId={accountId} /> : null}
    </Dialog.Root>
  )
}

function LoginContent({ providerId, accountId }: { providerId: string; accountId?: string }) {
  const { t } = useTranslation()
  const id = useId()
  const [name, setName] = useState("")
  const [mode, setMode] = useState("browser")
  const login = useSubscriptionLogin({ providerId, accountId })
  const waiting = login.view?.state === "waiting"
  const complete = login.view?.state === "completed"
  const busy = waiting || login.starting
  const invalid = !validAccountName(name)
  return (
    <Dialog.Portal>
      <Dialog.Overlay className="motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 fixed inset-0 z-40 bg-black/55" />
      <Dialog.Content
        className="bg-card text-card-foreground motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=open]:zoom-in-95 fixed top-1/2 left-1/2 z-50 flex max-h-[calc(100dvh-2rem)] w-[min(560px,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 flex-col overflow-hidden rounded-2xl border shadow-2xl outline-none"
        onEscapeKeyDown={(event) => {
          if (login.starting) event.preventDefault()
        }}
        onInteractOutside={(event) => {
          if (busy) event.preventDefault()
        }}
      >
        <div className="flex shrink-0 items-start justify-between gap-4 px-6 pt-6 pb-5">
          <div className="flex min-w-0 items-start gap-3.5">
            <div className="border-border/60 flex size-12 shrink-0 items-center justify-center rounded-xl border bg-zinc-950">
              <ProviderBrandIcon kind="openai" />
            </div>
            <div className="min-w-0 space-y-1.5 pt-0.5">
              <Dialog.Title className="text-lg font-semibold tracking-tight">
                {t("subscription.title")}
              </Dialog.Title>
              <Dialog.Description className="text-muted-foreground text-sm leading-6">
                {t("subscription.description")}
              </Dialog.Description>
            </div>
          </div>
          <Dialog.Close asChild>
            <Button
              variant="ghost"
              size="icon-sm"
              className="text-muted-foreground -mt-1 -mr-1 shrink-0 cursor-pointer"
              disabled={login.starting}
              aria-label={t("subscription.closeLabel")}
            >
              <X className="size-4" aria-hidden="true" />
            </Button>
          </Dialog.Close>
        </div>
        <form
          className="flex min-h-0 flex-col"
          onSubmit={(event) => {
            event.preventDefault()
            if (!invalid) void login.start({ providerId, accountId, name: name.trim(), mode })
          }}
        >
          <div className="min-h-0 space-y-5 overflow-y-auto px-6 pb-6">
            {!busy && !complete ? (
              <SubscriptionLoginMethod id={id} value={mode} onChange={setMode} />
            ) : null}
            {!accountId && !complete && !busy ? (
              <div className="space-y-1.5">
                <label htmlFor={id} className="text-sm font-medium">
                  {t("accounts.name")}
                </label>
                <Input
                  id={id}
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                  maxLength={256}
                  disabled={busy}
                  aria-invalid={invalid}
                  aria-describedby={invalid ? `${id}-error` : undefined}
                  placeholder={t("subscription.namePlaceholder")}
                />
                {invalid ? (
                  <p id={`${id}-error`} className="text-destructive text-xs">
                    {t("accounts.nameInvalid")}
                  </p>
                ) : null}
              </div>
            ) : null}
            <SubscriptionLoginStatus login={login} />
            <p className="text-muted-foreground flex items-start gap-2 text-xs leading-5">
              <LockKeyhole className="mt-0.5 size-3.5 shrink-0" aria-hidden="true" />
              {t("subscription.secureStorage")}
            </p>
          </div>
          <div className="bg-muted/20 flex shrink-0 flex-wrap items-center justify-end gap-2 border-t px-6 py-4">
            <Dialog.Close asChild>
              <Button
                type="button"
                variant={complete ? "default" : "outline"}
                disabled={login.starting}
                className="min-w-20 cursor-pointer"
              >
                {t(complete ? "subscription.done" : "subscription.close")}
              </Button>
            </Dialog.Close>
            {!waiting && !complete ? (
              <Button
                type="submit"
                disabled={login.starting || invalid}
                className="min-w-28 cursor-pointer"
              >
                {login.starting ? (
                  <LoaderCircle
                    className="size-4 animate-spin motion-reduce:animate-none"
                    aria-hidden="true"
                  />
                ) : null}
                {t(login.starting ? "subscription.starting" : "subscription.start")}
                {!login.starting ? <ArrowUpRight className="size-4" aria-hidden="true" /> : null}
              </Button>
            ) : null}
            {waiting ? (
              <Button
                type="button"
                variant="outline"
                onClick={() => void login.cancel()}
                className="cursor-pointer"
              >
                {t("accounts.cancel")}
              </Button>
            ) : null}
          </div>
        </form>
      </Dialog.Content>
    </Dialog.Portal>
  )
}
