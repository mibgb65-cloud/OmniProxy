import { useState } from "react"
import { Plus } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent } from "../../components/ui/card"
import type { ProviderConfiguration, ProviderDefinition } from "../../services/omniproxy"
import { ProviderCard, type ProbeDisplay } from "./provider-card"

export function ProviderList({
  items,
  definitions,
  onAdd,
  onEdit,
  onToggle,
  onDelete,
  onProbe,
  probeResults,
  probingId,
  busy,
}: {
  items: ProviderConfiguration[]
  definitions: ProviderDefinition[]
  onAdd: () => void
  onEdit: (item: ProviderConfiguration) => void
  onToggle: (item: ProviderConfiguration, enabled: boolean) => Promise<void>
  onDelete: (id: string) => Promise<boolean>
  onProbe: (item: ProviderConfiguration) => void
  probeResults: Record<string, ProbeDisplay>
  probingId: string | null
  busy: boolean
}) {
  const { t } = useTranslation()
  const [confirmId, setConfirmId] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const byKind = new Map(definitions.map((definition) => [definition.kind, definition]))

  const toggle = async (item: ProviderConfiguration, enabled: boolean) => {
    setActionError(null)
    try {
      await onToggle(item, enabled)
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
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 id="configured-providers-title" className="text-lg font-semibold">
            {t("providers.configured")}
          </h2>
          <p className="text-muted-foreground mt-1 text-sm">
            {t("providers.configuredDescription")}
          </p>
        </div>
        <Button type="button" onClick={onAdd} disabled={busy} className="cursor-pointer">
          <Plus aria-hidden="true" />
          {t("providers.add")}
        </Button>
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
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {items.map((item) => {
            const definition = byKind.get(item.kind)
            return (
              <div key={item.id} className="space-y-2">
                <ProviderCard
                  kind={item.kind}
                  name={item.name}
                  enabled={item.enabled}
                  disabled={busy}
                  authMethods={definition?.authMethods ?? ["api_key"]}
                  plannedAuthMethods={definition?.plannedAuthMethods ?? []}
                  models={definition?.models ?? []}
                  endpoint={item.baseUrl}
                  probeResult={probeResults[item.id]?.result}
                  probeError={probeResults[item.id]?.error}
                  probeVisible={probeResults[item.id]?.visible}
                  probePending={probingId === item.id}
                  onToggle={(enabled) => void toggle(item, enabled)}
                  onProbe={() => onProbe(item)}
                  actions={
                    <>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => onEdit(item)}
                        disabled={busy}
                        className="cursor-pointer"
                      >
                        {t("providers.edit")}
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
                    </>
                  }
                />
                {confirmId === item.id ? (
                  <div className="bg-muted/40 space-y-2 rounded-md border p-3 text-sm">
                    <p>{t("providers.deleteConfirm", { name: item.name })}</p>
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
              </div>
            )
          })}
        </div>
      )}
    </section>
  )
}
