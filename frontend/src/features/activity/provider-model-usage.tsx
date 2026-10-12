import { useTranslation } from "react-i18next"
import { Card, CardContent, CardHeader } from "../../components/ui/card"
import { Tooltip } from "../../components/ui/tooltip"
import { formatTokens, type ProviderModelUsage } from "../../services/activity"

export function ProviderModelUsageView({ items }: { items: ProviderModelUsage[] }) {
  const { t, i18n } = useTranslation()
  const groups = new Map<string, { name: string; items: ProviderModelUsage[] }>()
  for (const item of items) {
    const group = groups.get(item.providerId) ?? { name: item.providerName, items: [] }
    group.items.push(item)
    groups.set(item.providerId, group)
  }
  return (
    <Card className="min-w-0 gap-4">
      <CardHeader>
        <h2 className="text-base font-semibold">{t("activity.providerModels")}</h2>
        <p className="text-muted-foreground text-sm">{t("activity.providerModelsHint")}</p>
      </CardHeader>
      <CardContent className="space-y-4">
        {groups.size ? (
          [...groups].map(([id, group]) => (
            <section
              key={id}
              aria-label={group.name || t("activity.provider")}
              className="overflow-hidden rounded-lg border"
            >
              <div className="bg-muted/20 flex flex-wrap items-center justify-between gap-2 border-b px-4 py-3">
                <h3 className="text-sm font-semibold">{group.name || t("activity.provider")}</h3>
                {id.startsWith("legacy:") ? (
                  <span className="text-muted-foreground text-xs">
                    {t("activity.legacyProvider")}
                  </span>
                ) : (
                  <span className="text-muted-foreground text-xs">
                    {t("activity.modelCount", { count: group.items.length })}
                  </span>
                )}
              </div>
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead className="text-muted-foreground">
                    <tr>
                      {(
                        [
                          "activity.model",
                          "activity.input",
                          "activity.output",
                          "activity.total",
                          "activity.requests",
                        ] as const
                      ).map((key) => (
                        <th key={key} className="px-4 py-3 font-medium whitespace-nowrap">
                          {t(key)}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {group.items.map((item) => (
                      <tr key={item.model} className="border-t">
                        <td className="px-4 py-3">
                          <Tooltip content={item.model}>
                            <span className="block max-w-52 truncate font-mono">
                              {item.model || t("activity.noModel")}
                            </span>
                          </Tooltip>
                        </td>
                        <td className="px-4 py-3 font-mono tabular-nums">
                          {formatTokens(item.summary.inputTokens, i18n.resolvedLanguage)}
                        </td>
                        <td className="px-4 py-3 font-mono tabular-nums">
                          {formatTokens(item.summary.outputTokens, i18n.resolvedLanguage)}
                        </td>
                        <td className="px-4 py-3 font-mono font-medium tabular-nums">
                          {formatTokens(item.summary.totalTokens, i18n.resolvedLanguage)}
                        </td>
                        <td className="px-4 py-3 tabular-nums">{item.summary.requests}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>
          ))
        ) : (
          <p className="text-muted-foreground py-4 text-sm">{t("activity.empty")}</p>
        )}
      </CardContent>
    </Card>
  )
}
