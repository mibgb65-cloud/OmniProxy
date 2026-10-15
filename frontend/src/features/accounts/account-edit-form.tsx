import { useAccountEditForm } from "./use-account-edit-form"
import { MiMoOptionsFields } from "./mimo-options"
import { MiniMaxOptionsFields } from "./minimax-options"
import { KimiPlatformFields } from "./kimi-platform"
import { X } from "lucide-react"
import { Dialog } from "radix-ui"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Input } from "../../components/ui/input"
import type { AccountItem, UpdateAccountInput } from "../../services/omniproxy"
import { MAX_KEY_BYTES } from "./account-input"
import { StepFunOptionsFields } from "./stepfun-options"
import { AccountKeyPreview } from "./account-key-preview"

export function AccountEditForm({
  account,
  onSubmit,
  onRestoreFocus,
}: {
  account: AccountItem
  onSubmit: (input: UpdateAccountInput) => Promise<void>
  onRestoreFocus: () => void
}) {
  const { t } = useTranslation()
  const {
    name,
    setName,
    apiKey,
    setAPIKey,
    invalidName,
    invalidKey,
    saveError,
    saving,
    nameRef,
    submit,
    mimoEnabled,
    mimo,
    setMiMo,
    invalidMiMo,
    minimaxEnabled,
    minimax,
    setMiniMax,
    invalidMiniMax,
    kimiEnabled,
    kimi,
    setKimi,
    invalidKimi,
    stepEnabled,
    stepfun,
    setStepFun,
    invalidStep,
  } = useAccountEditForm(account, onSubmit)
  return (
    <Dialog.Portal>
      <Dialog.Overlay className="motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 fixed inset-0 z-40 bg-black/50" />
      <Dialog.Content
        className="bg-card text-card-foreground motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=open]:zoom-in-95 fixed top-1/2 left-1/2 z-50 max-h-[calc(100dvh-2rem)] w-[min(480px,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-xl border p-6 shadow-xl outline-none"
        onOpenAutoFocus={(event) => {
          event.preventDefault()
          nameRef.current?.focus()
          nameRef.current?.select()
        }}
        onCloseAutoFocus={(event) => {
          event.preventDefault()
          onRestoreFocus()
        }}
        onEscapeKeyDown={(event) => {
          if (saving) event.preventDefault()
        }}
        onInteractOutside={(event) => {
          if (saving) event.preventDefault()
        }}
      >
        <div className="mb-5 flex items-start justify-between gap-4">
          <div className="space-y-1.5">
            <Dialog.Title className="text-base font-semibold">
              {t("accounts.edit.title")}
            </Dialog.Title>
            <Dialog.Description className="text-muted-foreground text-sm">
              {t(
                account.authMethod === "subscription_login"
                  ? "subscription.editDescription"
                  : "accounts.edit.description",
              )}
            </Dialog.Description>
          </div>
          <Dialog.Close asChild>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              disabled={saving}
              aria-label={t("accounts.edit.close")}
              className="shrink-0 cursor-pointer"
            >
              <X className="size-4" aria-hidden="true" />
            </Button>
          </Dialog.Close>
        </div>
        <form onSubmit={submit} className="space-y-4">
          {mimoEnabled ? (
            <MiMoOptionsFields
              value={mimo}
              onChange={setMiMo}
              disabled={saving}
              invalid={invalidMiMo}
            />
          ) : null}
          {minimaxEnabled ? (
            <MiniMaxOptionsFields
              value={minimax}
              onChange={setMiniMax}
              disabled={saving}
              invalid={invalidMiniMax}
            />
          ) : null}
          {kimiEnabled ? (
            <KimiPlatformFields
              value={kimi}
              onChange={setKimi}
              disabled={saving}
              invalid={invalidKimi}
            />
          ) : null}
          {stepEnabled ? (
            <StepFunOptionsFields
              value={stepfun}
              onChange={setStepFun}
              disabled={saving}
              invalid={invalidStep}
            />
          ) : null}
          <div className="space-y-1.5">
            <label htmlFor="edit-account-name" className="text-sm font-medium">
              {t("accounts.name")}
            </label>
            <Input
              ref={nameRef}
              id="edit-account-name"
              value={name}
              disabled={saving}
              maxLength={256}
              onChange={(event) => setName(event.target.value)}
              aria-invalid={invalidName}
              aria-describedby={invalidName ? "edit-name-hint edit-name-error" : "edit-name-hint"}
            />
            <p id="edit-name-hint" className="text-muted-foreground text-xs">
              {t("accounts.nameHint")}
            </p>
            {invalidName ? (
              <p id="edit-name-error" className="text-destructive text-xs">
                {t("accounts.nameInvalid")}
              </p>
            ) : null}
          </div>
          {account.authMethod !== "subscription_login" ? (
            <>
              <div className="space-y-1.5">
                <p className="text-sm font-medium">{t("accounts.edit.currentKey")}</p>
                <AccountKeyPreview account={account} />
              </div>
              <div className="space-y-1.5">
                <label htmlFor="edit-account-key" className="text-sm font-medium">
                  {t("accounts.apiKey")}
                </label>
                <Input
                  id="edit-account-key"
                  type="password"
                  value={apiKey}
                  disabled={saving}
                  maxLength={MAX_KEY_BYTES}
                  autoComplete="off"
                  autoCapitalize="none"
                  spellCheck={false}
                  onChange={(event) => setAPIKey(event.target.value)}
                  placeholder={t("accounts.edit.keyPlaceholder")}
                  aria-invalid={invalidKey}
                  aria-describedby={invalidKey ? "edit-key-hint edit-key-error" : "edit-key-hint"}
                />
                <p id="edit-key-hint" className="text-muted-foreground text-xs">
                  {t("accounts.edit.keyHint")}
                </p>
                {invalidKey ? (
                  <p id="edit-key-error" className="text-destructive text-xs">
                    {t("accounts.keyInvalid")}
                  </p>
                ) : null}
              </div>
            </>
          ) : null}
          {saveError ? (
            <p role="alert" className="text-destructive text-sm">
              {t("accounts.saveFailed")}
            </p>
          ) : null}
          <div className="flex gap-2">
            <Button type="submit" disabled={saving} className="cursor-pointer">
              {saving ? t("accounts.saving") : t("accounts.edit.save")}
            </Button>
            <Dialog.Close asChild>
              <Button type="button" variant="outline" disabled={saving} className="cursor-pointer">
                {t("accounts.cancel")}
              </Button>
            </Dialog.Close>
          </div>
        </form>
      </Dialog.Content>
    </Dialog.Portal>
  )
}
