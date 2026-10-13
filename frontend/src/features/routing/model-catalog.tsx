import { SelectField } from "../../components/ui/select-field"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Download, RefreshCw } from "lucide-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import type { AccountItem, ProviderConfiguration } from "../../services/omniproxy"
import { listModelCatalog, syncAccountModels } from "../../services/models"
import { ModelCatalogGroups } from "./model-catalog-groups"
import { ModelProviderViewToggle, type ModelProviderView } from "./model-provider-view-toggle"

export function ModelCatalog({
  accounts,
  providers,
}: {
  accounts: AccountItem[]
  providers: ProviderConfiguration[]
}) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const [accountId, setAccountId] = useState("")
  const [view, setView] = useState<ModelProviderView>("model")
  const catalog = useQuery({ queryKey: ["model-catalog"], queryFn: listModelCatalog })
  const sync = useMutation({
    mutationFn: (id: string) => syncAccountModels(id),
    onSuccess: async (data) => {
      client.setQueryData(["model-catalog"], data)
      await Promise.all([
        client.invalidateQueries({ queryKey: ["routes"] }),
        client.invalidateQueries({ queryKey: ["default-routes"] }),
        client.invalidateQueries({ queryKey: ["gateway-runtime"] }),
        client.invalidateQueries({ queryKey: ["agent-clients"] }),
      ])
    },
  })
  const eligible = accounts.filter((a) =>
    providers.some(
      (p) =>
        p.id === a.providerId &&
        ((p.kind === "deepseek" && a.authMethod === "api_key") ||
          (p.kind === "openai" &&
            a.authMethod === "subscription_login" &&
            a.authState === "authenticated")),
    ),
  )
  const models = catalog.data ?? []
  return (
    <Card className="min-w-0 gap-4">
      <CardHeader>
        <h2 className="flex items-center gap-2 text-base font-semibold">
          <Download className="size-4" aria-hidden="true" />
          {t("models.title")}
        </h2>
      </CardHeader>
      <CardContent className="space-y-4">
        <ModelProviderViewToggle value={view} onChange={setView} />
        <div className="flex flex-wrap items-end gap-3">
          <div className="min-w-0 flex-1 space-y-2">
            <label htmlFor="model-sync-account" className="text-xs font-medium">
              {t("models.account")}
            </label>
            <SelectField
              id="model-sync-account"
              aria-label={t("models.account")}
              disabled={sync.isPending}
              value={accountId}
              onValueChange={(value) => {
                setAccountId(value)
                sync.reset()
              }}
              options={[
                { value: "", label: t("models.choose") },
                ...eligible.map((a) => ({
                  value: a.id,
                  label: `${a.authEmail || a.name} · ${providers.find((p) => p.id === a.providerId)?.name ?? ""}`,
                })),
              ]}
            />
          </div>
          <Button
            variant="outline"
            disabled={!accountId || sync.isPending}
            onClick={() => sync.mutate(accountId)}
          >
            <RefreshCw
              aria-hidden="true"
              className={sync.isPending ? "motion-safe:animate-spin" : ""}
            />
            {t(sync.isPending ? "models.syncing" : "models.sync")}
          </Button>
        </div>
        {sync.isError || catalog.isError ? (
          <p role="alert" className="text-destructive text-sm">
            {t("models.failed")}
          </p>
        ) : null}
        {models.length ? (
          <div className="min-w-0 space-y-3">
            <p className="text-muted-foreground text-xs">
              {t("models.catalogCount", { count: models.length })}
            </p>
            <ModelCatalogGroups
              models={models}
              accounts={accounts}
              providers={providers}
              view={view}
            />
          </div>
        ) : (
          <p className="text-muted-foreground text-xs">{t("models.empty")}</p>
        )}
      </CardContent>
    </Card>
  )
}
