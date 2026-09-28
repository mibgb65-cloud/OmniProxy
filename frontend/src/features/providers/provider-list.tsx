import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent } from "../../components/ui/card"
import type { ProviderConfiguration } from "../../services/omniproxy"
import { isKnownProviderKind, providerNameKeys } from "./provider-labels"

export function ProviderList({
  items,
  onEdit,
  onToggle,
  onDelete,
  busy,
}: {
  items: ProviderConfiguration[]
  onEdit: (item: ProviderConfiguration) => void
  onToggle: (item: ProviderConfiguration) => Promise<void>
  onDelete: (id: string) => Promise<boolean>
  busy: boolean
}) {
  const { t } = useTranslation()
  const [confirmId, setConfirmId] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)

  const toggle = async (item: ProviderConfiguration) => {
    setActionError(null)
    try {
      await onToggle(item)
    } catch {
      setActionError(t("providers.actionFailed"))
    }
  }

  const remove = async (id: string) => {
    setActionError(null)
    if (await onDelete(id)) setConfirmId(null)
  }

  return (
    <section aria-labelledby="configured-providers-title" className="space-y-4">
      <div>
        <h2 id="configured-providers-title" className="text-lg font-semibold">
          {t("providers.configured")}
        </h2>
        <p className="text-muted-foreground mt-1 text-sm">{t("providers.configuredDescription")}</p>
      </div>
      {actionError ? (
        <p role="alert" className="text-destructive text-sm">
          {actionError}
        </p>
      ) : null}
      {items.length === 0 ? (
        <Card className="dark:bg-card/75">
          <CardContent className="text-muted-foreground text-sm">
            {t("providers.configuredEmpty")}
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {items.map((item) => {
            const kindName = isKnownProviderKind(item.kind)
              ? t(providerNameKeys[item.kind])
              : item.kind
            return (
              <Card key={item.id} className="dark:bg-card/75 gap-0 py-0 shadow-sm">
                <CardContent className="space-y-3 px-5 py-4">
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-2">
                        <h3 className="font-medium">{item.name}</h3>
                        <span className="text-muted-foreground rounded-full border px-2 py-0.5 text-xs">
                          {kindName}
                        </span>
                        <span
                          className={`rounded-full px-2 py-0.5 text-xs ${item.enabled ? "bg-blue-50 text-blue-700 dark:bg-blue-500/15 dark:text-blue-300" : "bg-muted text-muted-foreground"}`}
                        >
                          {item.enabled ? t("providers.enabled") : t("providers.disabled")}
                        </span>
                      </div>
                      <p className="text-muted-foreground mt-1 text-xs break-all">
                        {item.baseUrl || t("providers.defaultBaseUrl")}
                      </p>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => onEdit(item)}
                        disabled={busy}
                        className="cursor-pointer"
                      >
                        {t("providers.edit")}
                      </Button>
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => void toggle(item)}
                        disabled={busy}
                        className="cursor-pointer"
                      >
                        {item.enabled ? t("providers.disable") : t("providers.enable")}
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => setConfirmId(item.id)}
                        disabled={busy}
                        className="text-destructive cursor-pointer"
                      >
                        {t("providers.delete")}
                      </Button>
                    </div>
                  </div>
                  {confirmId === item.id ? (
                    <div className="bg-muted/40 flex flex-wrap items-center justify-between gap-2 rounded-md border px-3 py-2 text-sm">
                      <span>{t("providers.deleteConfirm", { name: item.name })}</span>
                      <div className="flex gap-2">
                        <Button
                          type="button"
                          variant="destructive"
                          size="sm"
                          onClick={() => void remove(item.id)}
                          disabled={busy}
                          className="cursor-pointer"
                        >
                          {t("providers.delete")}
                        </Button>
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          onClick={() => setConfirmId(null)}
                          disabled={busy}
                          className="cursor-pointer"
                        >
                          {t("providers.cancel")}
                        </Button>
                      </div>
                    </div>
                  ) : null}
                </CardContent>
              </Card>
            )
          })}
        </div>
      )}
    </section>
  )
}
