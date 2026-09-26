import { Languages } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Button } from "../ui/button"
import { languageOptions } from "../../i18n/translation"
import { cn } from "../../lib/utils"
import { useUIStore } from "../../stores/ui-store"
import { SidebarLabel } from "./sidebar-label"

interface LanguageSwitcherProps {
  /** Collapsed sidebar shows a single toggle button instead of the segmented control. */
  collapsed: boolean
}

export function LanguageSwitcher({ collapsed }: LanguageSwitcherProps) {
  const { t } = useTranslation()
  const language = useUIStore((s) => s.language)
  const setLanguage = useUIStore((s) => s.setLanguage)

  const next = language === "zh" ? "en" : "zh"

  return (
    <div className="relative flex h-8 shrink-0 items-center">
      {collapsed && (
        <Button
          variant="ghost"
          size="sm"
          className="absolute inset-0 w-full transition-colors"
          aria-label={t("common.language")}
          title={t("common.language")}
          onClick={() => setLanguage(next)}
        />
      )}
      <Languages
        className="text-muted-foreground pointer-events-none absolute left-4 size-4"
        aria-hidden="true"
      />
      {/* 固定展开内容的宽度，裁切与逐字过渡不会把语言选项挤回图标上。 */}
      <div
        className="flex w-52 shrink-0 items-center justify-between gap-2 pl-10"
        inert={collapsed}
      >
        <SidebarLabel collapsed={collapsed} className="text-muted-foreground text-sm">
          {t("common.language")}
        </SidebarLabel>
        <div
          className={cn(
            "border-border flex shrink-0 rounded-md border transition-opacity duration-150 motion-reduce:transition-none",
            collapsed && "pointer-events-none opacity-0",
          )}
          role="group"
          aria-label={t("common.language")}
          aria-hidden={collapsed}
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
              aria-label={option.label}
              onClick={() => setLanguage(option.value)}
            >
              <SidebarLabel collapsed={collapsed}>{option.label}</SidebarLabel>
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}
