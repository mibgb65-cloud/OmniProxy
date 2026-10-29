import { MiMoOptionsFields } from "./mimo-options"
import { MiniMaxOptionsFields } from "./minimax-options"
import { KimiPlatformFields } from "./kimi-platform"
import { useRef } from "react"
import { X } from "lucide-react"
import { Dialog, Tabs } from "radix-ui"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import type {
  CreateAccountInput,
  CreateAccountsInput,
  ProviderConfiguration,
} from "../../services/omniproxy"
import { isKnownProviderKind, providerNameKeys } from "../providers/provider-labels"
import { AccountCredentialsFields } from "./account-credentials-fields"
import { StepFunOptionsFields } from "./stepfun-options"
import { useAccountForm } from "./use-account-form"

export function AccountForm({
  provider,
  onSubmit,
  onBatchSubmit,
}: {
  provider: ProviderConfiguration
  onSubmit: (input: CreateAccountInput) => Promise<void>
  onBatchSubmit: (input: CreateAccountsInput) => Promise<void>
}) {
  const { t } = useTranslation()
  const providerName =
    provider.source === "builtin" && isKnownProviderKind(provider.kind)
      ? t(providerNameKeys[provider.kind])
      : provider.name
  const nameRef = useRef<HTMLInputElement>(null)
  const form = useAccountForm({
    providerId: provider.id,
    stepEnabled: provider.kind === "stepfun",
    kimiEnabled: provider.kind === "kimi",
    minimaxEnabled: provider.kind === "minimax",
    mimoEnabled: provider.kind === "mimo",
    onSubmit,
    onBatchSubmit,
  })
  return (
    <Dialog.Portal>
      <Dialog.Overlay className="motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=closed]:animate-out motion-safe:data-[state=closed]:fade-out-0 fixed inset-0 z-40 bg-black/50 motion-safe:duration-150" />
      <Dialog.Content
        id="account-editor"
        className="bg-card text-card-foreground motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=open]:zoom-in-95 motion-safe:data-[state=closed]:animate-out motion-safe:data-[state=closed]:fade-out-0 motion-safe:data-[state=closed]:zoom-out-95 fixed top-1/2 left-1/2 z-50 max-h-[calc(100dvh-2rem)] w-[min(480px,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-xl border p-6 shadow-xl outline-none motion-safe:duration-150"
        onOpenAutoFocus={(event) => {
          event.preventDefault()
          nameRef.current?.focus()
        }}
        onEscapeKeyDown={(event) => {
          if (form.saving) event.preventDefault()
        }}
        onInteractOutside={(event) => {
          if (form.saving) event.preventDefault()
        }}
      >
        <div className="mb-5 flex items-start justify-between gap-4">
          <div className="space-y-1.5">
            <Dialog.Title className="text-base font-semibold">
              {t("accounts.newFor", { name: providerName })}
            </Dialog.Title>
            <Dialog.Description className="text-muted-foreground text-sm">
              {t("accounts.multipleHint")}
            </Dialog.Description>
          </div>
          <Dialog.Close asChild>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              disabled={form.saving}
              aria-label={t("accounts.close")}
              className="shrink-0 cursor-pointer"
            >
              <X className="size-4" aria-hidden="true" />
            </Button>
          </Dialog.Close>
        </div>
        <Tabs.Root value={form.mode} onValueChange={form.changeMode}>
          <Tabs.List
            aria-label={t("accounts.addMode")}
            className="bg-muted mb-4 grid grid-cols-2 gap-1 rounded-lg p-1"
          >
            {(["single", "batch"] as const).map((mode) => (
              <Tabs.Trigger key={mode} value={mode} asChild disabled={form.saving}>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="data-[state=active]:bg-background cursor-pointer data-[state=active]:shadow-sm"
                >
                  {mode === "single" ? t("accounts.singleMode") : t("accounts.batchMode")}
                </Button>
              </Tabs.Trigger>
            ))}
          </Tabs.List>
          <form onSubmit={form.submit} className="space-y-4">
            {provider.kind === "mimo" ? (
              <MiMoOptionsFields
                value={form.mimo}
                onChange={form.setMiMo}
                disabled={form.saving}
                invalid={form.invalidMiMo}
              />
            ) : null}
            {provider.kind === "minimax" ? (
              <MiniMaxOptionsFields
                value={form.minimax}
                onChange={form.setMiniMax}
                disabled={form.saving}
                invalid={form.invalidMiniMax}
              />
            ) : null}
            {provider.kind === "kimi" ? (
              <KimiPlatformFields
                value={form.kimi}
                onChange={form.setKimi}
                disabled={form.saving}
                invalid={form.invalidKimi}
              />
            ) : null}
            {provider.kind === "stepfun" ? (
              <StepFunOptionsFields
                value={form.stepfun}
                onChange={form.setStepFun}
                disabled={form.saving}
              />
            ) : null}
            {(["single", "batch"] as const).map((mode) => (
              <Tabs.Content key={mode} value={mode} className="space-y-4 outline-none">
                <AccountCredentialsFields form={form} nameRef={nameRef} />
              </Tabs.Content>
            ))}
            {form.saveError ? (
              <p role="alert" className="text-destructive text-sm">
                {t("accounts.saveFailed")}
              </p>
            ) : null}
            <div className="flex flex-wrap gap-2">
              <Button type="submit" disabled={form.saving} className="cursor-pointer">
                {form.saving
                  ? t("accounts.saving")
                  : form.mode === "batch"
                    ? t("accounts.saveBatch")
                    : t("accounts.save")}
              </Button>
              <Dialog.Close asChild>
                <Button
                  type="button"
                  variant="outline"
                  disabled={form.saving}
                  className="cursor-pointer"
                >
                  {t("accounts.cancel")}
                </Button>
              </Dialog.Close>
            </div>
          </form>
        </Tabs.Root>
      </Dialog.Content>
    </Dialog.Portal>
  )
}
