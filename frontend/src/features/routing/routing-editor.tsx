import { zodResolver } from "@hookform/resolvers/zod"
import { X } from "lucide-react"
import { Dialog } from "radix-ui"
import { FormProvider, useForm } from "react-hook-form"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { z } from "zod"
import { Button } from "../../components/ui/button"
import { RoutingEditorFields } from "./routing-editor-fields"
import type { TranslationKey } from "../../i18n/translation"
import type { RouteInput, RouteItem } from "../../services/routing"
import type { AccountItem, ProviderConfiguration } from "../../services/omniproxy"
import type { DiscoveredModel } from "../../services/models"
import { routingSaveErrorKey } from "./routing-errors"

const schema = z.object({
  alias: z.string().regex(/^[A-Za-z0-9_.:/-]{1,128}$/),
  providerId: z.string().min(1).max(128),
  model: z.string().regex(/^[A-Za-z0-9_.:/-]{1,128}$/),
  accountId: z.string().max(128),
  enabled: z.boolean(),
})

export function RoutingEditor({
  initial,
  providers,
  accounts,
  models,
  catalog = [],
  onSubmit,
  onClose,
  onRestoreFocus,
}: {
  initial: RouteItem | null
  providers: ProviderConfiguration[]
  accounts: AccountItem[]
  models: string[]
  catalog?: DiscoveredModel[]
  onSubmit: (input: RouteInput) => Promise<void>
  onClose: () => void
  onRestoreFocus: () => void
}) {
  const { t } = useTranslation()
  const [saveError, setSaveError] = useState<TranslationKey | null>(null)
  const methods = useForm<RouteInput>({
    resolver: zodResolver(schema),
    defaultValues: {
      alias: initial?.rule.alias ?? "",
      providerId: initial?.rule.providerId ?? "",
      model: initial?.rule.model ?? "",
      accountId: initial?.rule.accountId ?? "",
      enabled: initial?.rule.enabled ?? true,
    },
  })
  const {
    handleSubmit,
    setFocus,
    setError,
    formState: { isSubmitting },
  } = methods
  const submit = handleSubmit(async (input) => {
    setSaveError(null)
    if (
      !providers.some(
        (item) =>
          item.id === input.providerId &&
          (item.kind === "deepseek" ||
            item.kind === "atria" ||
            item.kind === "mimo" ||
            (["openai", "google"].includes(item.kind) &&
              catalog.some((model) => model.providerId === item.id))),
      ) ||
      !models.includes(input.model)
    ) {
      setError("providerId", { type: "validate" })
      setFocus("providerId")
      return
    }
    if (
      input.accountId &&
      !accounts.some((item) => item.id === input.accountId && item.providerId === input.providerId)
    ) {
      setError("accountId", { type: "validate" })
      setFocus("accountId")
      return
    }
    try {
      await onSubmit(input)
    } catch (error) {
      const key = routingSaveErrorKey(error)
      if (key === "routing.aliasDuplicate") {
        setError("alias", { type: "server" })
        setFocus("alias")
      } else setSaveError(key)
    }
  })
  return (
    <Dialog.Portal>
      <Dialog.Overlay className="fixed inset-0 z-40 bg-black/50" />
      <Dialog.Content
        className="bg-card text-card-foreground fixed top-1/2 left-1/2 z-50 max-h-[calc(100dvh-2rem)] w-[min(520px,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-xl border p-6 shadow-xl outline-none"
        onOpenAutoFocus={(event) => {
          event.preventDefault()
          setFocus("alias")
        }}
        onCloseAutoFocus={(event) => {
          event.preventDefault()
          onRestoreFocus()
        }}
        onEscapeKeyDown={(event) => {
          if (isSubmitting) event.preventDefault()
        }}
        onInteractOutside={(event) => {
          if (isSubmitting) event.preventDefault()
        }}
      >
        <div className="mb-5 flex items-start justify-between gap-3">
          <div>
            <Dialog.Title className="text-base font-semibold">
              {t(initial ? "routing.editTitle" : "routing.newTitle")}
            </Dialog.Title>
            <Dialog.Description className="text-muted-foreground mt-2 text-sm leading-5">
              {t("routing.formDescription")}
            </Dialog.Description>
          </div>
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label={t("routing.close")}
            disabled={isSubmitting}
            onClick={onClose}
          >
            <X aria-hidden="true" />
          </Button>
        </div>
        <FormProvider {...methods}>
          <form onSubmit={submit} className="space-y-4">
            <RoutingEditorFields
              providers={providers}
              accounts={accounts}
              models={models}
              catalog={catalog}
            />
            {saveError ? (
              <p role="alert" className="text-destructive text-sm">
                {t(saveError)}
              </p>
            ) : null}
            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" disabled={isSubmitting} onClick={onClose}>
                {t("routing.cancel")}
              </Button>
              <Button type="submit" disabled={isSubmitting}>
                {t(isSubmitting ? "routing.saving" : "routing.save")}
              </Button>
            </div>
          </form>
        </FormProvider>
      </Dialog.Content>
    </Dialog.Portal>
  )
}
