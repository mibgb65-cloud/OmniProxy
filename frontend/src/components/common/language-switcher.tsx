import { Check, Languages } from "lucide-react"
import { DropdownMenu } from "radix-ui"
import { useTranslation } from "react-i18next"
import { isLanguage, languageOptions } from "../../i18n/translation"
import { useUIStore } from "../../stores/ui-store"

export function LanguageSwitcher() {
  const { t } = useTranslation()
  const language = useUIStore((s) => s.language)
  const setLanguage = useUIStore((s) => s.setLanguage)

  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>
        <button
          type="button"
          aria-label={t("common.language")}
          className="hover:bg-sidebar-accent focus-visible:ring-ring data-[state=open]:bg-sidebar-accent flex h-11 w-11 shrink-0 cursor-pointer items-center justify-center outline-none focus-visible:ring-2 focus-visible:ring-inset"
        >
          <Languages className="size-4 shrink-0" aria-hidden="true" />
        </button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content
          side="bottom"
          align="end"
          sideOffset={6}
          collisionPadding={12}
          loop
          aria-label={t("common.language")}
          className="bg-popover/95 text-popover-foreground motion-safe:data-[state=open]:animate-in motion-safe:data-[state=open]:fade-in-0 motion-safe:data-[state=open]:zoom-in-95 motion-safe:data-[state=closed]:animate-out motion-safe:data-[state=closed]:fade-out-0 motion-safe:data-[state=closed]:zoom-out-95 z-80 min-w-36 rounded-xl border p-1.5 shadow-[0_12px_40px_-12px_rgba(0,0,0,0.28)] backdrop-blur-xl motion-safe:duration-150"
        >
          <DropdownMenu.RadioGroup
            value={language}
            onValueChange={(value) => {
              if (isLanguage(value)) setLanguage(value)
            }}
          >
            {languageOptions.map((option) => (
              <DropdownMenu.RadioItem
                key={option.value}
                value={option.value}
                className="data-[highlighted]:bg-accent/80 relative flex min-h-9 cursor-pointer items-center rounded-lg py-2 pr-9 pl-3 text-sm outline-none select-none data-[state=checked]:bg-blue-500/8 data-[state=checked]:text-blue-700 dark:data-[state=checked]:text-blue-300"
              >
                {option.label}
                <DropdownMenu.ItemIndicator className="absolute right-3 flex size-4 items-center justify-center">
                  <Check className="size-3.5" aria-hidden="true" />
                </DropdownMenu.ItemIndicator>
              </DropdownMenu.RadioItem>
            ))}
          </DropdownMenu.RadioGroup>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  )
}
