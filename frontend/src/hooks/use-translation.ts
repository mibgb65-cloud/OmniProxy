import { useCallback } from "react"
import { useUIStore } from "../stores/ui-store"
import { translate, type TranslationKey } from "../i18n/translation"

/**
 * Returns the translate function for the currently selected language.
 * All user-facing strings must go through this hook — never hardcode
 * display text in components.
 */
export function useTranslation() {
  const language = useUIStore((s) => s.language)

  const t = useCallback(
    (key: TranslationKey, vars?: Record<string, string | number>) => translate(language, key, vars),
    [language],
  )

  return { t, language }
}
