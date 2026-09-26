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
