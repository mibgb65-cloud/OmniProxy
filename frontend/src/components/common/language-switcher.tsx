import { Languages } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../ui/button"
import { languageOptions } from "../../i18n/translation"
import { cn } from "../../lib/utils"
import { useUIStore } from "../../stores/ui-store"

interface LanguageSwitcherProps {
  /** Collapsed sidebar shows a single toggle button instead of the segmented control. */
  collapsed: boolean
}

export function LanguageSwitcher({ collapsed }: LanguageSwitcherProps) {
  const { t } = useTranslation()
  const language = useUIStore((s) => s.language)
  const setLanguage = useUIStore((s) => s.setLanguage)

  if (collapsed) {
    const next = language === "zh" ? "en" : "zh"
    return (
      <Button
        variant="ghost"
        size="sm"
        className="w-full justify-center"
        aria-label={t("common.language")}
        title={t("common.language")}
        onClick={() => setLanguage(next)}
      >
        <Languages className="size-4" />
      </Button>
    )
  }

  return (
    <div className="flex items-center justify-between gap-2 px-2">
      <span className="text-muted-foreground flex items-center gap-2 text-sm">
        <Languages className="size-4" />
        {t("common.language")}
      </span>
      <div
        className="border-border flex rounded-md border"
        role="group"
        aria-label={t("common.language")}
      >
        {languageOptions.map((option) => (
          <button
            key={option.value}
            type="button"
            className={cn(
              "px-2 py-1 text-xs first:rounded-l-[5px] last:rounded-r-[5px]",
              language === option.value
                ? "bg-accent text-accent-foreground font-medium"
                : "text-muted-foreground hover:text-foreground",
            )}
            aria-pressed={language === option.value}
            onClick={() => setLanguage(option.value)}
          >
            {option.label}
          </button>
        ))}
      </div>
    </div>
  )
}
