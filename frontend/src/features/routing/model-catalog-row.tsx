import { useTranslation } from "react-i18next"
import type { AccountItem } from "../../services/omniproxy"
import type { DiscoveredModel } from "../../services/models"

export function ModelCatalogRow({
  model,
  account,
}: {
  model: DiscoveredModel
  account?: AccountItem
}) {
  const { t } = useTranslation()
  return (
    <li className="space-y-1.5 py-2 text-xs">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <div className="min-w-0 space-y-1">
          <p className="font-medium wrap-anywhere">{model.name || model.model}</p>
          {model.name && model.name !== model.model ? (
            <code className="text-muted-foreground wrap-anywhere">{model.model}</code>
          ) : null}
        </div>
        <span className="text-muted-foreground min-w-0 wrap-anywhere">
          {account?.authEmail || account?.name}
        </span>
      </div>
      <p className="text-muted-foreground">
        {model.metadata?.contextWindow
          ? t("models.contextWindow", { count: model.metadata.contextWindow })
          : t("models.metadataUnknown")}
        {model.metadata?.reasoningEfforts?.length
          ? ` · ${t("models.reasoningEfforts", { values: model.metadata.reasoningEfforts.map((effort) => effort.value).join(" / ") })}`
          : ""}
      </p>
    </li>
  )
}
