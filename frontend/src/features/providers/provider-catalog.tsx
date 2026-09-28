import { useTranslation } from "react-i18next"
import type { ProviderConfiguration, ProviderDefinition } from "../../services/omniproxy"
import { ProviderCard, type ProbeDisplay } from "./provider-card"
import { isKnownProviderKind, providerNameKeys } from "./provider-labels"

export function ProviderCatalog({
  definitions,
  items,
  busy,
  probingId,
  probeResults,
  onToggle,
  onProbe,
}: {
  definitions: ProviderDefinition[]
  items: ProviderConfiguration[]
  busy: boolean
  probingId: string | null
  probeResults: Record<string, ProbeDisplay>
  onToggle: (item: ProviderConfiguration, enabled: boolean) => void
  onProbe: (item: ProviderConfiguration) => void
}) {
  const { t } = useTranslation()
  const byKind = new Map(items.map((item) => [item.kind, item]))

  return (
    <section aria-labelledby="provider-catalog-title" className="space-y-4">
      <div>
        <h2 id="provider-catalog-title" className="text-lg font-semibold">
          {t("providers.catalog")}
        </h2>
      </div>
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
        {definitions.map((definition) => {
          const name = isKnownProviderKind(definition.kind)
            ? t(providerNameKeys[definition.kind])
            : definition.name
          const item = byKind.get(definition.kind)
          return item ? (
            <ProviderCard
              key={definition.kind}
              kind={definition.kind}
              name={name}
              enabled={item.enabled}
              disabled={busy}
              authMethods={definition.authMethods}
              plannedAuthMethods={definition.plannedAuthMethods}
              models={definition.models ?? []}
              endpoint={definition.defaultBaseUrl}
              probeResult={probeResults[item.id]?.result}
              probeError={probeResults[item.id]?.error}
              probeVisible={probeResults[item.id]?.visible}
              probePending={probingId === item.id}
              onToggle={(enabled) => onToggle(item, enabled)}
              onProbe={() => onProbe(item)}
            />
          ) : (
            <div
              key={definition.kind}
              className="border-destructive/30 rounded-lg border p-4 text-sm"
            >
              {name}: {t("providers.presetUnavailable")}
            </div>
          )
        })}
      </div>
    </section>
  )
}
