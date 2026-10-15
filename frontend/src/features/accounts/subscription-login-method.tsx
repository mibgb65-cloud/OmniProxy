import { Globe2, Keyboard, Check } from "lucide-react"
import { useTranslation } from "react-i18next"
import { cn } from "../../lib/utils"

export function SubscriptionLoginMethod({
  id,
  value,
  onChange,
}: {
  id: string
  value: string
  onChange: (value: string) => void
}) {
  const { t } = useTranslation()
  return (
    <fieldset className="space-y-3">
      <legend className="text-sm font-medium">{t("subscription.mode")}</legend>
      <div className="grid gap-3 min-[480px]:grid-cols-2">
        {(["browser", "device"] as const).map((mode) => {
          const selected = value === mode
          const Icon = mode === "browser" ? Globe2 : Keyboard
          const descriptionId = `${id}-${mode}-hint`
          return (
            <label
              key={mode}
              className={cn(
                "focus-within:ring-ring/50 relative flex cursor-pointer flex-col gap-3 rounded-xl border p-4 transition-colors focus-within:ring-2 motion-reduce:transition-none",
                selected
                  ? "border-primary/65 bg-primary/[0.04]"
                  : "border-border hover:border-foreground/30 hover:bg-muted/40",
              )}
            >
              <input
                type="radio"
                name={`${id}-mode`}
                value={mode}
                checked={selected}
                onChange={() => onChange(mode)}
                aria-label={t(`subscription.${mode}`)}
                aria-describedby={descriptionId}
                className="sr-only"
              />
              <span className="flex items-center justify-between gap-3">
                <Icon className="text-foreground size-5" aria-hidden="true" />
                <span
                  className={cn(
                    "flex size-4 shrink-0 items-center justify-center rounded-full border",
                    selected ? "border-primary bg-primary text-primary-foreground" : "border-input",
                  )}
                  aria-hidden="true"
                >
                  {selected ? <Check className="size-3" strokeWidth={3} /> : null}
                </span>
              </span>
              <span className="space-y-1.5">
                <span className="block text-sm font-medium">{t(`subscription.${mode}`)}</span>
                <span id={descriptionId} className="text-muted-foreground block text-xs leading-5">
                  {t(`subscription.${mode}Hint`)}
                </span>
              </span>
            </label>
          )
        })}
      </div>
    </fieldset>
  )
}
