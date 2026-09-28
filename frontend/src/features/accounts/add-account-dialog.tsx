import { useState } from "react"
import { Dialog } from "radix-ui"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import type { CreateAccountInput, ProviderConfiguration } from "../../services/omniproxy"
import { AccountForm } from "./account-form"

export function AddAccountDialog({
  provider,
  onCreate,
}: {
  provider: ProviderConfiguration
  onCreate: (input: CreateAccountInput) => Promise<void>
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)

  const create = async (input: CreateAccountInput) => {
    await onCreate(input)
    setOpen(false)
  }

  return (
    <Dialog.Root open={open} onOpenChange={setOpen}>
      <Dialog.Trigger asChild>
        <Button type="button" variant="outline" size="sm" className="shrink-0 cursor-pointer">
          {t("accounts.add")}
        </Button>
      </Dialog.Trigger>
      {open ? <AccountForm provider={provider} onSubmit={create} /> : null}
    </Dialog.Root>
  )
}
