import { useState } from "react"
import { Dialog } from "radix-ui"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import type {
  CreateAccountInput,
  CreateAccountsInput,
  ProviderConfiguration,
} from "../../services/omniproxy"
import { AccountForm } from "./account-form"
import { SubscriptionLoginDialog } from "./subscription-login-dialog"

export function AddAccountDialog({
  provider,
  onCreate,
  onBatchCreate,
}: {
  provider: ProviderConfiguration
  onCreate: (input: CreateAccountInput) => Promise<void>
  onBatchCreate: (input: CreateAccountsInput) => Promise<void>
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)

  const create = async (input: CreateAccountInput) => {
    await onCreate(input)
    setOpen(false)
  }

  const createBatch = async (input: CreateAccountsInput) => {
    await onBatchCreate(input)
    setOpen(false)
  }

  return (
    <div className="flex flex-wrap gap-2">
      {["openai", "google"].includes(provider.kind) ? (
        <SubscriptionLoginDialog providerId={provider.id} />
      ) : null}
      <Dialog.Root open={open} onOpenChange={setOpen}>
        <Dialog.Trigger asChild>
          <Button type="button" variant="outline" size="sm" className="shrink-0 cursor-pointer">
            {t("accounts.add")}
          </Button>
        </Dialog.Trigger>
        {open ? (
          <AccountForm provider={provider} onSubmit={create} onBatchSubmit={createBatch} />
        ) : null}
      </Dialog.Root>
    </div>
  )
}
