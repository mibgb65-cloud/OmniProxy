import { useEffect, type ReactNode } from "react"
import i18next from "i18next"
import { useUIStore } from "../stores/ui-store"
import { initI18n } from "./config"

// Initialize the shared i18next instance at module load so the very first
// render already has translations available.
void initI18n()

/**
 * Keeps i18next in sync with the language preference in the Zustand UI store
 * and maintains document-level language metadata (`<html lang>`, window
 * title).
 */
export function I18nProvider({ children }: { children: ReactNode }) {
  const language = useUIStore((s) => s.language)

  useEffect(() => {
    if (i18next.resolvedLanguage !== language) {
      void i18next.changeLanguage(language)
    }
    document.documentElement.lang = language === "zh" ? "zh-CN" : "en"
    document.title = i18next.t("app.title", { lng: language })
  }, [language])

  return <>{children}</>
}
