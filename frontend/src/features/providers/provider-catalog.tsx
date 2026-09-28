import { Plus } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card"
import type { ProviderDefinition } from "../../services/omniproxy"
import { isKnownProviderKind, providerNameKeys } from "./provider-labels"

export function ProviderCatalog({
  definitions,
  counts,
  onAdd,
}: {
  definitions: ProviderDefinition[]
  counts: Record<string, number>
  onAdd: (definition: ProviderDefinition, name: string) => void
}) {
  const { t } = useTranslation()

  return (
    <section aria-labelledby="provider-catalog-title" className="space-y-4">
      <div>
        <h2 id="provider-catalog-title" className="text-lg font-semibold">
          {t("providers.catalog")}
        </h2>
        <p className="text-muted-foreground mt-1 text-sm">{t("providers.catalogDescription")}</p>
      </div>
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
        {definitions.map((definition) => {
          const name = isKnownProviderKind(definition.kind)
            ? t(providerNameKeys[definition.kind])
            : definition.name
          return (
            <Card key={definition.kind} className="gap-0 py-0 shadow-sm">
              <CardHeader className="gap-1 px-5 pt-5 pb-3">
                <CardTitle className="flex items-center justify-between gap-2 text-base">
                  <span>{name}</span>
                  <span className="text-muted-foreground text-xs font-normal">
                    {t("providers.configuredCount", { count: counts[definition.kind] ?? 0 })}
                  </span>
                </CardTitle>
              </CardHeader>
              <CardContent className="flex h-full flex-col justify-between gap-5 px-5 pb-5">
                <div className="flex flex-wrap gap-1.5">
                  {definition.authMethods?.includes("api_key") ? (
                    <span className="border-border bg-muted/50 rounded-full border px-2.5 py-1 text-xs">
                      {t("providers.apiKey")}
                    </span>
                  ) : null}
                  {definition.plannedAuthMethods?.includes("subscription_login") ? (
                    <span className="border-border text-muted-foreground rounded-full border px-2.5 py-1 text-xs">
                      {t("providers.subscriptionPlanned")}
                    </span>
                  ) : null}
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="w-full cursor-pointer"
                  onClick={() => onAdd(definition, name)}
                  aria-label={`${t("providers.add")}：${name}`}
                >
                  <Plus aria-hidden="true" />
                  {t("providers.add")}
                </Button>
              </CardContent>
            </Card>
          )
        })}
      </div>
    </section>
  )
}
