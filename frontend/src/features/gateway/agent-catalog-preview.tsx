import { useTranslation } from "react-i18next"
import type { AgentPlan } from "../../services/agents"

const modeLabels = {
  "local-catalog": "agents.mode.local",
  "remote-catalog": "agents.mode.remote",
  "desktop-model-list": "agents.mode.desktop",
  "client-default": "agents.mode.default",
} as const

export function AgentCatalogPreview({ plan }: { plan: AgentPlan }) {
  const { t } = useTranslation()
  const models = plan.models ?? []
  const mode = Object.hasOwn(modeLabels, plan.mode)
    ? modeLabels[plan.mode as keyof typeof modeLabels]
    : "agents.mode.default"
  return (
    <section className="bg-muted/30 space-y-3 rounded-lg border p-4 text-sm">
      <dl className="space-y-2">
        <div className="flex flex-wrap justify-between gap-2">
          <dt className="text-muted-foreground">{t("agents.defaultModel")}</dt>
          <dd className="min-w-0 font-mono wrap-anywhere">{plan.defaultModel}</dd>
        </div>
        <div className="flex flex-wrap justify-between gap-2">
          <dt className="text-muted-foreground">{t("agents.catalogMode")}</dt>
          <dd>{t(mode)}</dd>
        </div>
      </dl>
      <details open>
        <summary className="cursor-pointer font-medium">
          {t("agents.catalogCount", { count: models.length })}
        </summary>
        <ul className="mt-2 max-h-48 space-y-2 overflow-auto border-t pt-3">
          {models.map((model) => (
            <li key={model.id} className="space-y-1 text-xs">
              <div className="flex flex-wrap items-baseline justify-between gap-2">
                <span className="font-medium wrap-anywhere">{model.name}</span>
                <code className="text-muted-foreground min-w-0 wrap-anywhere">{model.id}</code>
              </div>
              <p className="text-muted-foreground">
                {model.metadata.contextWindow
                  ? t("models.contextWindow", { count: model.metadata.contextWindow })
                  : t("models.metadataUnknown")}
              </p>
            </li>
          ))}
        </ul>
      </details>
      <p className="text-muted-foreground text-xs leading-5">{t("agents.catalogRefreshHint")}</p>
    </section>
  )
}
