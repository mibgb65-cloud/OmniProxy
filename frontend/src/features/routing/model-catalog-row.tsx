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
      <p className="text-muted-foreground min-w-0 wrap-anywhere">
        {account?.authEmail || account?.name || model.accountId}
      </p>
      <p className="text-muted-foreground wrap-anywhere">
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
