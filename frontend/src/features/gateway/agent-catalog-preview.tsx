import { useTranslation } from "react-i18next"
import { ScrollRegion } from "../../components/common/scroll-region"
import type { AgentPlan } from "../../services/agents"

const modeLabels = {
  "local-catalog": "agents.mode.local",
  "remote-catalog": "agents.mode.remote",
  "desktop-model-list": "agents.mode.desktop",
  "cli-model-list": "agents.mode.cli",
  "client-default": "agents.mode.default",
  "opencode-model-list": "agents.mode.opencode",
  "harness-model-list": "agents.mode.harness",
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
          <dt className="text-muted-foreground">
            {t(
              plan.mode === "harness-model-list" ? "agents.suggestedModel" : "agents.defaultModel",
            )}
          </dt>
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
        <ScrollRegion
          as="section"
          ariaLabel={t("agents.catalogCount", { count: models.length })}
          className="mt-2 max-h-48 border-t"
          viewportClassName="pt-3"
        >
          <ul className="space-y-2">
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
        </ScrollRegion>
      </details>
      <p className="text-muted-foreground text-xs leading-5">{t("agents.catalogRefreshHint")}</p>
    </section>
  )
}
