import { useRef, useState } from "react"
import { X } from "lucide-react"
import { Dialog } from "radix-ui"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Input } from "../../components/ui/input"
import type { CreateAccountInput, ProviderConfiguration } from "../../services/omniproxy"
import { isKnownProviderKind, providerNameKeys } from "../providers/provider-labels"

function validName(value: string): boolean {
  const name = value.trim()
  return (
    name.length > 0 &&
    Array.from(name).length <= 80 &&
    new TextEncoder().encode(name).length <= 256 &&
    !/[\p{Cc}]/u.test(name)
  )
}

function validKey(value: string): boolean {
  const key = value.trim()
  return key.length > 0 && new TextEncoder().encode(key).length <= 8192 && !/[\p{Cc}]/u.test(key)
}

export function AccountForm({
  provider,
  onSubmit,
}: {
  provider: ProviderConfiguration
  onSubmit: (input: CreateAccountInput) => Promise<void>
}) {
  const { t } = useTranslation()
  const providerName =
    provider.source === "builtin" && isKnownProviderKind(provider.kind)
      ? t(providerNameKeys[provider.kind])
      : provider.name
  const nameRef = useRef<HTMLInputElement>(null)
  const [name, setName] = useState("")
  const [apiKey, setAPIKey] = useState("")
  const [invalidName, setInvalidName] = useState(false)
  const [invalidKey, setInvalidKey] = useState(false)
  const [saveError, setSaveError] = useState(false)
  const [saving, setSaving] = useState(false)

  const submit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const nameError = !validName(name)
    const keyError = !validKey(apiKey)
    setInvalidName(nameError)
    setInvalidKey(keyError)
    if (nameError || keyError) return
    setSaving(true)
    setSaveError(false)
    try {
      await onSubmit({ providerId: provider.id, name: name.trim(), apiKey: apiKey.trim() })
    } catch {
      setSaveError(true)
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog.Portal>
      <Dialog.Overlay className="motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 fixed inset-0 z-40 bg-black/50" />
      <Dialog.Content
        id="account-editor"
        className="bg-card text-card-foreground motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=open]:zoom-in-95 fixed top-1/2 left-1/2 z-50 max-h-[calc(100dvh-2rem)] w-[min(480px,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-xl border p-6 shadow-xl outline-none"
        onOpenAutoFocus={(event) => {
          event.preventDefault()
          nameRef.current?.focus()
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
              disabled={saving}
              aria-label={t("accounts.close")}
              className="shrink-0 cursor-pointer"
            >
              <X className="size-4" aria-hidden="true" />
            </Button>
          </Dialog.Close>
        </div>
        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-1.5">
            <label htmlFor="account-name" className="text-sm font-medium">
              {t("accounts.name")}
            </label>
            <Input
              ref={nameRef}
              id="account-name"
              value={name}
              onChange={(event) => setName(event.target.value)}
              maxLength={256}
              placeholder={t("accounts.namePlaceholder")}
              aria-invalid={invalidName}
              aria-describedby={invalidName ? "account-name-error" : undefined}
            />
            {invalidName ? (
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
              value={apiKey}
              onChange={(event) => setAPIKey(event.target.value)}
              maxLength={8192}
              autoComplete="off"
              autoCapitalize="none"
              spellCheck={false}
              aria-invalid={invalidKey}
              aria-describedby={invalidKey ? "account-key-error" : "account-key-hint"}
            />
            <p id="account-key-hint" className="text-muted-foreground text-xs">
              {t("accounts.keyHint")}
            </p>
            {invalidKey ? (
              <p id="account-key-error" className="text-destructive text-xs">
                {t("accounts.keyInvalid")}
              </p>
            ) : null}
          </div>
          {saveError ? (
            <p role="alert" className="text-destructive text-sm">
              {t("accounts.saveFailed")}
            </p>
          ) : null}
          <div className="flex flex-wrap gap-2">
            <Button type="submit" disabled={saving} className="cursor-pointer">
              {saving ? t("accounts.saving") : t("accounts.save")}
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
