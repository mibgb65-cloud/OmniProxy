import { useId, useState } from "react"
import { Dialog } from "radix-ui"
import { LoaderCircle, LogIn, X } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Input } from "../../components/ui/input"
import { Tooltip } from "../../components/ui/tooltip"
import { validAccountName } from "./account-input"
import { useSubscriptionLogin } from "./use-subscription-login"
import { SubscriptionLoginStatus } from "./subscription-login-status"

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
      <Dialog.Overlay className="fixed inset-0 z-40 bg-black/50" />
      <Dialog.Content
        className="bg-card text-card-foreground fixed top-1/2 left-1/2 z-50 max-h-[calc(100dvh-2rem)] w-[min(500px,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-xl border p-6 shadow-xl outline-none"
        onEscapeKeyDown={(event) => {
          if (login.starting) event.preventDefault()
        }}
        onInteractOutside={(event) => {
          if (busy) event.preventDefault()
        }}
      >
        <div className="mb-5 flex items-start justify-between gap-3">
          <div className="space-y-2">
            <Dialog.Title className="font-semibold">{t("subscription.title")}</Dialog.Title>
            <Dialog.Description className="text-muted-foreground text-sm">
              {t("subscription.description")}
            </Dialog.Description>
          </div>
          <Dialog.Close asChild>
            <Button
              variant="ghost"
              size="icon"
              disabled={login.starting}
              aria-label={t("subscription.closeLabel")}
            >
              <X className="size-4" />
            </Button>
          </Dialog.Close>
        </div>
        <form
          className="space-y-4"
          onSubmit={(event) => {
            event.preventDefault()
            if (!invalid) void login.start({ providerId, accountId, name: name.trim(), mode })
          }}
        >
          {!accountId && !complete ? (
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
                placeholder={t("subscription.namePlaceholder")}
              />
              {invalid ? (
                <p className="text-destructive text-xs">{t("accounts.nameInvalid")}</p>
              ) : null}
            </div>
          ) : null}
          {!busy && !complete ? (
            <fieldset className="space-y-2">
              <legend className="text-sm font-medium">{t("subscription.mode")}</legend>
              <div className="flex flex-wrap gap-4">
                {(["browser", "device"] as const).map((value) => (
                  <label key={value} className="flex cursor-pointer items-center gap-2 text-sm">
                    <input
                      type="radio"
                      name={id + "-mode"}
                      checked={mode === value}
                      onChange={() => setMode(value)}
                    />
                    {t(`subscription.${value}`)}
                  </label>
                ))}
              </div>
            </fieldset>
          ) : null}
          <SubscriptionLoginStatus login={login} />
          <p className="text-muted-foreground text-xs">{t("subscription.scope")}</p>
          <div className="flex gap-2">
            {!waiting && !complete ? (
              <Button type="submit" disabled={login.starting || invalid} className="cursor-pointer">
                {login.starting ? (
                  <LoaderCircle className="size-4 animate-spin motion-reduce:animate-none" />
                ) : null}
                {t(login.starting ? "subscription.starting" : "subscription.start")}
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
            <Dialog.Close asChild>
              <Button
                type="button"
                variant="outline"
                disabled={login.starting}
                className="cursor-pointer"
              >
                {t(complete ? "subscription.done" : "subscription.close")}
              </Button>
            </Dialog.Close>
          </div>
        </form>
      </Dialog.Content>
    </Dialog.Portal>
  )
}
