import type { RefObject } from "react"
import { useTranslation } from "react-i18next"
import { Input } from "../../components/ui/input"
import { MAX_BATCH_KEYS, MAX_BATCH_TEXT_BYTES, MAX_KEY_BYTES } from "./account-input"
import type { useAccountForm } from "./use-account-form"

export function AccountCredentialsFields({
  form,
  nameRef,
}: {
  form: ReturnType<typeof useAccountForm>
  nameRef: RefObject<HTMLInputElement | null>
}) {
  const { t } = useTranslation()
  if (form.mode === "batch")
    return (
      <div className="space-y-1.5">
        <label htmlFor="account-batch-keys" className="text-sm font-medium">
          {t("accounts.apiKeys")}
        </label>
        <textarea
          id="account-batch-keys"
          rows={6}
          maxLength={MAX_BATCH_TEXT_BYTES}
          value={form.batchText}
          onChange={(event) => form.setBatchText(event.target.value)}
          disabled={form.saving}
          autoComplete="off"
          autoCapitalize="none"
          spellCheck={false}
          placeholder={t("accounts.batchPlaceholder")}
          aria-invalid={form.invalidKey}
          aria-describedby={
            form.invalidKey ? "account-batch-hint account-key-error" : "account-batch-hint"
          }
          className={`border-input dark:bg-input/30 placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-ring/50 min-h-36 w-full resize-y rounded-md border bg-transparent px-3 py-2 text-sm shadow-xs outline-none focus-visible:ring-[3px] disabled:opacity-50 ${form.batchText ? "[-webkit-text-security:disc]" : ""}`}
        />
        <p id="account-batch-hint" className="text-muted-foreground text-xs">
          {t("accounts.batchHint", { max: MAX_BATCH_KEYS })}
        </p>
        <p className="text-muted-foreground text-xs" role="status">
          {t("accounts.batchCount", { count: form.batchCount })}
        </p>
        {form.invalidKey ? (
          <p id="account-key-error" className="text-destructive text-xs">
            {t("accounts.batchInvalid", { max: MAX_BATCH_KEYS })}
          </p>
        ) : null}
      </div>
    )
  return (
    <>
      <div className="space-y-1.5">
        <label htmlFor="account-name" className="text-sm font-medium">
          {t("accounts.name")}
        </label>
        <Input
          ref={nameRef}
          id="account-name"
          value={form.name}
          onChange={(event) => form.setName(event.target.value)}
          disabled={form.saving}
          maxLength={256}
          placeholder={t("accounts.namePlaceholder")}
          aria-invalid={form.invalidName}
          aria-describedby={
            form.invalidName ? "account-name-hint account-name-error" : "account-name-hint"
          }
        />
        <p id="account-name-hint" className="text-muted-foreground text-xs">
          {t("accounts.nameHint")}
        </p>
        {form.invalidName ? (
          <p id="account-name-error" className="text-destructive text-xs">
            {t("accounts.nameInvalid")}
          </p>
        ) : null}
      </div>
      <div className="space-y-1.5">
        <label htmlFor="account-api-key" className="text-sm font-medium">
          {t("accounts.apiKey")}
        </label>
        <Input
          id="account-api-key"
          type="password"
          value={form.apiKey}
          onChange={(event) => form.setAPIKey(event.target.value)}
          disabled={form.saving}
          maxLength={MAX_KEY_BYTES}
          autoComplete="off"
          autoCapitalize="none"
          spellCheck={false}
          aria-invalid={form.invalidKey}
          aria-describedby={
            form.invalidKey ? "account-key-hint account-key-error" : "account-key-hint"
          }
        />
        <p id="account-key-hint" className="text-muted-foreground text-xs">
          {t("accounts.keyHint")}
        </p>
        {form.invalidKey ? (
          <p id="account-key-error" className="text-destructive text-xs">
            {t("accounts.keyInvalid")}
          </p>
        ) : null}
      </div>
    </>
  )
}
