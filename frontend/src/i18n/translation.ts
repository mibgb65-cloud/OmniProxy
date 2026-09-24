import { en } from "./locales/en"
import { zh } from "./locales/zh"

/** Every translatable string in the UI is keyed by this union. */
export type TranslationKey = keyof typeof zh

/** Supported UI languages. */
export type Language = "zh" | "en"

const dictionaries: Record<Language, Record<TranslationKey, string>> = { zh, en }

/**
 * translate looks up `key` for `language` and substitutes `{name}`
 * placeholders from `vars`. A missing key falls back to the key itself so a
 * typo is visible in the UI instead of crashing; with the typed dictionaries
 * this should never happen at compile time.
 */
export function translate(
  language: Language,
  key: TranslationKey,
  vars?: Record<string, string | number>,
): string {
  const template = dictionaries[language][key] ?? key
  if (!vars) {
    return template
  }
  return template.replace(/\{(\w+)\}/g, (match, name: string) =>
    name in vars ? String(vars[name]) : match,
  )
}

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
