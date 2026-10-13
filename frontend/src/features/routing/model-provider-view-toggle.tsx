import { Boxes, Network } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../../components/ui/button"

export type ModelProviderView = "model" | "provider"

export function ModelProviderViewToggle({
  value,
  onChange,
}: {
  value: ModelProviderView
  onChange: (value: ModelProviderView) => void
}) {
  const { t } = useTranslation()
  return (
    <div
      role="group"
      aria-label={t("routing.grouping.label")}
      className="bg-muted inline-flex max-w-full flex-wrap gap-1 rounded-lg p-1"
    >
      {(["model", "provider"] as const).map((view) => {
        const Icon = view === "model" ? Boxes : Network
        return (
          <Button
            key={view}
            type="button"
            variant="ghost"
            className="text-muted-foreground aria-pressed:border-border aria-pressed:bg-background aria-pressed:text-foreground min-h-11 border border-transparent aria-pressed:shadow-sm motion-reduce:transition-none"
            aria-pressed={value === view}
            onClick={() => onChange(view)}
          >
            <Icon aria-hidden="true" />
            {t(view === "model" ? "routing.grouping.byModel" : "routing.grouping.byProvider")}
          </Button>
        )
      })}
    </div>
  )
}
