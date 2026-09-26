import { useLayoutEffect, type ReactNode } from "react"
import { applyTheme } from "../lib/theme"
import { useUIStore } from "../stores/ui-store"

/** Keep document colours in sync after the user changes the theme. */
export function ThemeProvider({ children }: { children: ReactNode }) {
  const theme = useUIStore((s) => s.theme)

  useLayoutEffect(() => {
    applyTheme(theme)
  }, [theme])

  return <>{children}</>
}
