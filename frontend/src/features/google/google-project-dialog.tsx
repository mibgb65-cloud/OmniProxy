import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useState } from "react"
import { Dialog } from "radix-ui"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Input } from "../../components/ui/input"
import type { AccountItem } from "../../services/omniproxy"
import { googleError, setGoogleProject } from "../../services/google"
import { accountDisplayName } from "../accounts/account-display"

export function GoogleProjectDialog({
  account,
  projectId,
}: {
  account: AccountItem
  projectId: string
}) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const [open, setOpen] = useState(false)
  const [value, setValue] = useState(projectId)
  const mutation = useMutation({
    mutationFn: () =>
      setGoogleProject({
        accountId: account.id,
        accountUpdatedAt: account.updatedAt,
        projectId: value.trim(),
      }),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ["google-resources"] })
      setOpen(false)
    },
  })
  return (
    <Dialog.Root
      open={open}
      onOpenChange={(next) => {
        if (mutation.isPending) return
        if (next) {
          setValue(projectId)
          mutation.reset()
        }
        setOpen(next)
      }}
    >
      <Dialog.Trigger asChild>
        <Button
          variant="outline"
          size="sm"
          aria-label={t("google.bindAccount", { name: accountDisplayName(account) })}
        >
          {t("google.bind")}
        </Button>
      </Dialog.Trigger>
      <Dialog.Portal>
        <Dialog.Overlay className="motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=closed]:animate-out motion-safe:data-[state=closed]:fade-out-0 fixed inset-0 z-40 bg-black/50 motion-safe:duration-150" />
        <Dialog.Content
          className="bg-card motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=open]:zoom-in-95 motion-safe:data-[state=closed]:animate-out motion-safe:data-[state=closed]:fade-out-0 motion-safe:data-[state=closed]:zoom-out-95 fixed top-1/2 left-1/2 z-50 w-[min(480px,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 space-y-4 rounded-xl border p-6 shadow-xl motion-safe:duration-150"
          onEscapeKeyDown={(event) => {
            if (mutation.isPending) event.preventDefault()
          }}
          onInteractOutside={(event) => {
            if (mutation.isPending) event.preventDefault()
          }}
        >
          <Dialog.Title className="font-semibold">
            {t("google.bindAccount", { name: accountDisplayName(account) })}
          </Dialog.Title>
          <Dialog.Description className="text-muted-foreground text-sm">
            {t("google.bindingHint")}
          </Dialog.Description>
          <form
            className="space-y-4"
            onSubmit={(event) => {
              event.preventDefault()
              if (!mutation.isPending) mutation.mutate()
            }}
          >
            <label className="block space-y-2 text-sm">
              <span>{t("google.projectId")}</span>
              <Input
                value={value}
                maxLength={30}
                disabled={mutation.isPending}
                autoComplete="off"
                spellCheck={false}
                pattern="[a-z][a-z0-9-]{4,28}[a-z0-9]"
                aria-describedby="google-project-hint"
                onChange={(event) => setValue(event.target.value)}
              />
            </label>
            <p id="google-project-hint" className="text-muted-foreground text-xs">
              {t("google.projectHint")}
            </p>
            {mutation.isError ? (
              <p role="alert" className="text-destructive text-sm">
                {t(`google.error.${googleError(mutation.error)}`)}
              </p>
            ) : null}
            <div className="flex justify-end gap-2">
              <Dialog.Close asChild>
                <Button type="button" variant="outline" disabled={mutation.isPending}>
                  {t("google.cancel")}
                </Button>
              </Dialog.Close>
              <Button type="submit" disabled={mutation.isPending}>
                {t(mutation.isPending ? "google.saving" : "google.save")}
              </Button>
            </div>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
