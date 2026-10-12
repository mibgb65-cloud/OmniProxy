import { Power, PowerOff } from "lucide-react"
import { Button } from "../ui/button"
import { Tooltip } from "../ui/tooltip"
import { cn } from "../../lib/utils"

export function PowerButton({
  ariaLabel,
  actionLabel,
  enabled,
  disabled,
  onChange,
}: {
  ariaLabel: string
  actionLabel: string
  enabled: boolean
  disabled: boolean
  onChange: (enabled: boolean) => void
}) {
  const Icon = enabled ? Power : PowerOff
  return (
    <Tooltip content={actionLabel}>
      <Button
        type="button"
        variant="ghost"
        size="icon-sm"
        aria-label={ariaLabel}
        aria-pressed={enabled}
        disabled={disabled}
        onClick={() => onChange(!enabled)}
        className={cn(
          "cursor-pointer rounded-lg border transition-colors",
          enabled
            ? "border-emerald-200 bg-emerald-50 text-emerald-700 hover:bg-emerald-100 dark:border-emerald-800 dark:bg-emerald-950 dark:text-emerald-300 dark:hover:bg-emerald-900"
            : "border-border bg-muted/40 text-muted-foreground hover:bg-muted hover:text-foreground",
        )}
      >
        <Icon className="size-4" aria-hidden="true" />
      </Button>
    </Tooltip>
  )
}
