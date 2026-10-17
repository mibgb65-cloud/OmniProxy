import { useState } from "react"
import { useTranslation } from "react-i18next"
import { ScrollRegion } from "../../components/common/scroll-region"
import { Button } from "../../components/ui/button"
import { Input } from "../../components/ui/input"
import { SelectField } from "../../components/ui/select-field"
import type { AgentPlan } from "../../services/agents"
import { AgentModelCard } from "./agent-model-card"
import "./agent-model-grid.css"

const modeLabels = {
  "local-catalog": "agents.mode.local",
  "remote-catalog": "agents.mode.remote",
  "desktop-model-list": "agents.mode.desktop",
  "cli-model-list": "agents.mode.cli",
  "client-default": "agents.mode.default",
  "opencode-model-list": "agents.mode.opencode",
  "harness-model-list": "agents.mode.harness",
  "kilo-model-list": "agents.mode.kilocode",
  "qwen-model-list": "agents.mode.qwenCode",
  "mimo-code-model-list": "agents.mode.mimoCode",
} as const

type ModelSelection = {
  ids: string[]
  defaultModel: string
  disabled: boolean
  onChange: (ids: string[]) => void
  onDefaultChange: (id: string) => void
}

export function AgentCatalogPreview({
  plan,
  selection,
  expanded = false,
}: {
  plan: AgentPlan
  selection?: ModelSelection
  expanded?: boolean
}) {
  const { t } = useTranslation()
  const [query, setQuery] = useState("")
  const [only1m, setOnly1m] = useState(false)
  const models = plan.models ?? []
  const modeName =
    plan.mode === "remote-catalog" && selection && selection.ids.length !== models.length
      ? "local-catalog"
      : plan.mode
  const mode = Object.hasOwn(modeLabels, modeName)
    ? modeLabels[modeName as keyof typeof modeLabels]
    : "agents.mode.default"
  const count = selection?.ids.length ?? models.length
  const visibleModels = models.filter(
    (model) =>
      (!only1m || model.id.endsWith("[1m]")) &&
      `${model.name} ${model.id}`.toLowerCase().includes(query.trim().toLowerCase()),
  )
  return (
    <section className="bg-muted/30 space-y-3 rounded-lg border p-4 text-sm">
      <dl className="space-y-2">
        <div className="flex flex-wrap justify-between gap-2">
          <dt id="agent-default-model-label" className="text-muted-foreground">
            {t(
              plan.mode === "harness-model-list" ? "agents.suggestedModel" : "agents.defaultModel",
            )}
          </dt>
          <dd className="max-w-full min-w-0 sm:w-80">
            {selection && plan.mode !== "harness-model-list" ? (
              <SelectField
                aria-labelledby="agent-default-model-label"
                value={selection.defaultModel}
                disabled={selection.disabled || count === 0}
                onValueChange={selection.onDefaultChange}
                options={models
                  .filter((model) => selection.ids.includes(model.id))
                  .map((model) => ({ value: model.id, label: model.name }))}
                placeholder={t("agents.choose")}
              />
            ) : (
              <span className="font-mono wrap-anywhere">
                {selection?.defaultModel ?? plan.defaultModel}
              </span>
            )}
          </dd>
        </div>
        <div className="flex flex-wrap justify-between gap-2">
          <dt className="text-muted-foreground">{t("agents.catalogMode")}</dt>
          <dd>{t(mode)}</dd>
        </div>
      </dl>
      <details open>
        <summary className="cursor-pointer font-medium">
          {t("agents.catalogCount", { count })}
        </summary>
        {selection ? (
          <div className="mt-3 space-y-2">
            <p className="text-muted-foreground text-xs">{t("agents.modelSelectionHint")}</p>
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span aria-live="polite" aria-atomic="true" className="text-muted-foreground text-xs">
                {t("agents.modelsSelected", { count, total: models.length })}
              </span>
              <div className="flex gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  disabled={selection.disabled || count === models.length}
                  onClick={() => selection.onChange(models.map((model) => model.id))}
                >
                  {t("agents.selectAllModels")}
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  disabled={selection.disabled || count === 0}
                  onClick={() => selection.onChange([])}
                >
                  {t("agents.clearModels")}
                </Button>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <Input
                type="search"
                className="min-w-0 flex-[1_1_14rem]"
                aria-label={t("agents.searchModels")}
                placeholder={t("agents.searchModels")}
                value={query}
                onChange={(event) => setQuery(event.target.value)}
              />
              <Button
                variant="outline"
                aria-pressed={only1m}
                className="aria-pressed:bg-accent aria-pressed:text-accent-foreground shrink-0"
                onClick={() => setOnly1m((value) => !value)}
              >
                {t("agents.only1mModels")}
              </Button>
            </div>
            <p aria-live="polite" aria-atomic="true" className="text-muted-foreground text-xs">
              {t("agents.connectedModelsCount", {
                count: visibleModels.length,
                total: models.length,
              })}
            </p>
            {count === 0 ? (
              <p role="alert" className="text-destructive text-xs">
                {t("agents.selectAtLeastOneModel")}
              </p>
            ) : null}
          </div>
        ) : null}
        <ScrollRegion
          as="section"
          ariaLabel={t("agents.catalogCount", { count })}
          className={expanded ? "mt-2 max-h-[min(26rem,45dvh)] border-t" : "mt-2 max-h-48 border-t"}
          viewportClassName="pt-3"
        >
          <ul className="agent-model-grid grid gap-2">
            {visibleModels.map((model) => (
              <li key={model.id} className="min-w-0">
                <AgentModelCard
                  model={model}
                  isDefault={model.id === (selection?.defaultModel ?? plan.defaultModel)}
                  selection={
                    selection
                      ? {
                          checked: selection.ids.includes(model.id),
                          disabled: selection.disabled,
                          onChange: (checked) =>
                            selection.onChange(
                              checked
                                ? [...selection.ids, model.id]
                                : selection.ids.filter((id) => id !== model.id),
                            ),
                        }
                      : undefined
                  }
                />
              </li>
            ))}
          </ul>
          {visibleModels.length === 0 ? (
            <p className="text-muted-foreground py-3 text-xs">{t("agents.noMatchingModels")}</p>
          ) : null}
        </ScrollRegion>
      </details>
      <p className="text-muted-foreground text-xs leading-5">{t("agents.catalogRefreshHint")}</p>
    </section>
  )
}
