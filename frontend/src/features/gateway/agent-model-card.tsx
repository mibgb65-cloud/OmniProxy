import { useTranslation } from "react-i18next"
import { cn } from "../../lib/utils"
import { SelectField } from "../../components/ui/select-field"

type EffortControl = {
  value: string
  disabled: boolean
  onChange: (effort: string) => void
}

export function AgentModelCard({
  model,
  isDefault = false,
  selection,
  effort,
}: {
  model: {
    id: string
    name: string
    metadata?: {
      contextWindow?: number | null
      reasoningEfforts?: { value: string; description?: string }[] | null
    }
  }
  isDefault?: boolean
  selection?: { checked: boolean; disabled: boolean; onChange: (checked: boolean) => void }
  effort?: EffortControl
}) {
  const { t } = useTranslation()
  const Container = selection ? "label" : "div"
  const levels = model.metadata?.reasoningEfforts ?? []
  // 只有上游公布过档位的模型才提供选择；未确认能力时保持「跟随客户端」。
  const showEffort = effort !== undefined && levels.length > 0
  return (
    <div
      className={cn(
        "bg-muted/20 flex h-full min-w-0 flex-col gap-2 rounded-lg border p-3",
        selection?.checked && "border-primary/25 bg-primary/5",
      )}
    >
      <Container
        className={cn(
          "flex min-w-0 items-start gap-2.5",
          selection &&
            "hover:bg-accent/40 focus-within:ring-ring cursor-pointer rounded-md focus-within:ring-2",
          selection?.disabled && "cursor-not-allowed opacity-60",
        )}
      >
        {selection ? (
          <input
            type="checkbox"
            className="accent-primary mt-0.5 size-4 shrink-0 cursor-pointer"
            aria-label={t("agents.selectModel", { model: model.name || model.id })}
            checked={selection.checked}
            disabled={selection.disabled}
            onChange={(event) => selection.onChange(event.target.checked)}
          />
        ) : null}
        <div className="min-w-0 flex-1 space-y-2">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <span className="min-w-0 text-sm font-medium wrap-anywhere">
              {model.name || model.id}
            </span>
            {isDefault ? (
              <span className="bg-muted text-muted-foreground rounded-md border px-2 py-0.5 text-xs">
                {t("agents.defaultModel")}
              </span>
            ) : null}
          </div>
          <p className="text-muted-foreground font-mono text-xs wrap-anywhere">{model.id}</p>
          {model.metadata ? (
            <p className="text-muted-foreground text-xs">
              {model.metadata.contextWindow
                ? t("models.contextWindow", { count: model.metadata.contextWindow })
                : t("models.metadataUnknown")}
            </p>
          ) : null}
        </div>
      </Container>
      {showEffort ? (
        <div className="flex flex-wrap items-center gap-2 border-t pt-2">
          <span className="text-muted-foreground text-xs">{t("agents.effort")}</span>
          <SelectField
            className="h-8 min-w-24 flex-1 text-xs"
            aria-label={t("agents.effortFor", { model: model.name || model.id })}
            disabled={effort.disabled}
            value={effort.value}
            onValueChange={effort.onChange}
            options={[
              { value: "", label: t("agents.effortFollowClient") },
              ...levels.map((level) => ({
                value: level.value,
                label: level.description ? `${level.description}（${level.value}）` : level.value,
              })),
            ]}
          />
        </div>
      ) : null}
    </div>
  )
}
