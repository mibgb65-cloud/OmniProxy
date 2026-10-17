import { useTranslation } from "react-i18next"
import { cn } from "../../lib/utils"

export function AgentModelCard({
  model,
  isDefault = false,
  selection,
}: {
  model: { id: string; name: string; metadata?: { contextWindow?: number | null } }
  isDefault?: boolean
  selection?: { checked: boolean; disabled: boolean; onChange: (checked: boolean) => void }
}) {
  const { t } = useTranslation()
  const Container = selection ? "label" : "div"
  return (
    <Container
      className={cn(
        "bg-muted/20 block h-full min-w-0 rounded-lg border p-3",
        selection &&
          "hover:bg-accent/40 focus-within:ring-ring flex cursor-pointer items-start gap-2.5 focus-within:ring-2",
        selection?.checked && "border-primary/25 bg-primary/5",
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
  )
}
