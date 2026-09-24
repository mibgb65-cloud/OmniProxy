import { useEffect, type ReactNode } from "react"
import { useUIStore } from "../stores/ui-store"
import { translate } from "./translation"

/**
 * Keeps document-level language metadata in sync with the selected UI
 * language: `<html lang>` for accessibility and the window title.
 */
export function I18nProvider({ children }: { children: ReactNode }) {
  const language = useUIStore((s) => s.language)

  useEffect(() => {
    document.documentElement.lang = language === "zh" ? "zh-CN" : "en"
    document.title = translate(language, "app.title")
  }, [language])

  return <>{children}</>
}
