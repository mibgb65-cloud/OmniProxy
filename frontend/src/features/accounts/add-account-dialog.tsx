import { useState } from "react"
import { Dialog } from "radix-ui"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { useOpenGeneration } from "../../lib/use-open-generation"
import type {
  CreateAccountInput,
  CreateAccountsInput,
  ProviderConfiguration,
} from "../../services/omniproxy"
import { AccountForm } from "./account-form"
import { SubscriptionLoginDialog } from "./subscription-login-dialog"
import { DuMateImportDialog } from "../dumate/dumate-import-dialog"

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
  const generation = useOpenGeneration(open)

  // DuMate 只有登录态导入，不接受 API Key；不能复用通用账号表单。
  if (provider.kind === "dumate") {
    return <DuMateImportDialog providerId={provider.id} />
  }

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
        {/* 常驻表单：退出动画需要 Radix 的关闭态有机会渲染；每次打开换 key 保证空白草稿。 */}
        <AccountForm
          key={generation}
          provider={provider}
          onSubmit={create}
          onBatchSubmit={createBatch}
        />
      </Dialog.Root>
    </div>
  )
}
