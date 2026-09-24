import type { zh } from "./locales/zh"

/** Every translatable string in the UI is keyed by this union. */
export type TranslationKey = keyof typeof zh

/** Supported UI languages. */
export type Language = "zh" | "en"

/** Supported language options for the language switcher. */
export const languageOptions: readonly { value: Language; label: string }[] = [
  { value: "zh", label: "中文" },
  { value: "en", label: "English" },
]

/** Default language for a fresh install (the project's primary language). */
export const defaultLanguage: Language = "zh"

/** Type guard for values coming from persisted storage. */
export function isLanguage(value: unknown): value is Language {
  return value === "zh" || value === "en"
}
