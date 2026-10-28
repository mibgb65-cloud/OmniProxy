/** The desktop theme is a UI preference, independent of the OS setting. */
export type Theme = "light" | "dark"

export const defaultTheme: Theme = "dark"

/** Validate values loaded from localStorage before applying them to the UI. */
export function isTheme(value: unknown): value is Theme {
  return value === "light" || value === "dark"
}

export function applyTheme(theme: Theme) {
  document.documentElement.classList.toggle("dark", theme === "dark")
  document.documentElement.style.colorScheme = theme
}

/**
 * 主题圆形扩散的半径：点击点到视口四角的最远距离。
 * 以它为半径，扩散结束时圆形一定能盖满整个窗口。
 */
export function themeRevealRadius(x: number, y: number, width: number, height: number): number {
  return Math.hypot(Math.max(x, width - x), Math.max(y, height - y))
}
