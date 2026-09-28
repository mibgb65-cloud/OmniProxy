import { useEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card"
import { Input } from "../../components/ui/input"
import type { ProviderConfiguration, UpdateProviderInput } from "../../services/omniproxy"

type FieldError =
  | "providers.nameRequired"
  | "providers.nameTooLong"
  | "providers.urlInvalid"
  | "providers.urlSecure"

function validate(name: string, baseUrl: string): { name?: FieldError; baseUrl?: FieldError } {
  const errors: { name?: FieldError; baseUrl?: FieldError } = {}
  if (!name.trim()) errors.name = "providers.nameRequired"
  else if (
    Array.from(name.trim()).length > 80 ||
    new TextEncoder().encode(name.trim()).length > 256
  )
    errors.name = "providers.nameTooLong"

  if (baseUrl.trim()) {
    try {
      const url = new URL(baseUrl.trim())
      if (
        !url.hostname ||
        url.username ||
        url.password ||
        url.search ||
        url.hash ||
        !["http:", "https:"].includes(url.protocol)
      ) {
        errors.baseUrl = "providers.urlInvalid"
      } else if (
        url.protocol === "http:" &&
        !(
          ["localhost", "[::1]"].includes(url.hostname.toLowerCase()) ||
          url.hostname.startsWith("127.")
        )
      ) {
        errors.baseUrl = "providers.urlSecure"
      }
    } catch {
      errors.baseUrl = "providers.urlInvalid"
    }
  }
  return errors
}

export function ProviderForm({
  providerName,
  initial,
  onSubmit,
  onCancel,
}: {
  providerName: string
  initial?: ProviderConfiguration
  onSubmit: (input: UpdateProviderInput) => Promise<void>
  onCancel: () => void
}) {
  const { t } = useTranslation()
  const nameRef = useRef<HTMLInputElement>(null)
  const [name, setName] = useState(initial?.name ?? providerName)
  const [baseUrl, setBaseUrl] = useState(initial?.baseUrl ?? "")
  const [enabled, setEnabled] = useState(initial?.enabled ?? true)
  const [errors, setErrors] = useState<ReturnType<typeof validate>>({})
  const [saveError, setSaveError] = useState(false)
  const [saving, setSaving] = useState(false)

  useEffect(() => nameRef.current?.focus(), [])

  const submit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const nextErrors = validate(name, baseUrl)
    setErrors(nextErrors)
    if (nextErrors.name || nextErrors.baseUrl) return
    setSaveError(false)
    setSaving(true)
    try {
      await onSubmit({ name: name.trim(), baseUrl: baseUrl.trim(), enabled })
    } catch {
      setSaveError(true)
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card className="dark:bg-card/75 gap-0" id="provider-editor">
      <CardHeader className="pb-4">
        <CardTitle className="text-base">
          {initial ? t("providers.editTitle") : t("providers.newTitle", { name: providerName })}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-1.5">
            <label htmlFor="provider-name" className="text-sm font-medium">
              {t("providers.name")}
            </label>
            <Input
              ref={nameRef}
              id="provider-name"
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder={t("providers.namePlaceholder")}
              maxLength={256}
              aria-invalid={Boolean(errors.name)}
            />
            {errors.name ? <p className="text-destructive text-xs">{t(errors.name)}</p> : null}
          </div>
          <div className="space-y-1.5">
            <label htmlFor="provider-base-url" className="text-sm font-medium">
              {t("providers.baseUrl")}
            </label>
            <Input
              id="provider-base-url"
              type="url"
              value={baseUrl}
              onChange={(event) => setBaseUrl(event.target.value)}
              placeholder={t("providers.baseUrlPlaceholder")}
              maxLength={2048}
              aria-invalid={Boolean(errors.baseUrl)}
            />
            <p className="text-muted-foreground text-xs">{t("providers.baseUrlHint")}</p>
            {errors.baseUrl ? (
              <p className="text-destructive text-xs">{t(errors.baseUrl)}</p>
            ) : null}
          </div>
          <label className="flex cursor-pointer items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={enabled}
              onChange={(event) => setEnabled(event.target.checked)}
              className="accent-blue-600"
            />
            {t("providers.enabled")}
          </label>
          {saveError ? (
            <p role="alert" className="text-destructive text-sm">
              {t("providers.saveFailed")}
            </p>
          ) : null}
          <div className="flex flex-wrap gap-2">
            <Button type="submit" disabled={saving} className="cursor-pointer">
              {saving ? t("providers.saving") : t("providers.save")}
            </Button>
            <Button
              type="button"
              variant="outline"
              onClick={onCancel}
              disabled={saving}
              className="cursor-pointer"
            >
              {t("providers.cancel")}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  )
}
