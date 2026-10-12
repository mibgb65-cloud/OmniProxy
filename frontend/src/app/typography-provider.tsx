import { useLayoutEffect, type ReactNode } from "react"
import { applyFontScale } from "../lib/typography"
import { useUIStore } from "../stores/ui-store"

export function TypographyProvider({ children }: { children: ReactNode }) {
  const fontScale = useUIStore((state) => state.fontScale)
  useLayoutEffect(() => applyFontScale(fontScale), [fontScale])
  return <>{children}</>
}
