import i18next from "i18next"
import { initReactI18next } from "react-i18next"
import { en } from "./locales/en"
import { zh } from "./locales/zh"
import { useUIStore } from "../stores/ui-store"

/**
 * i18next resources, keyed by language first (i18next's resource format:
 * `{ [lng]: { [ns]: keys } }`). Keys inside each language are flat and dotted
 * (e.g. "nav.dashboard"), so `keySeparator: false` keeps them literal instead
 * of nested lookups.
 */
export const i18nResources = {
  zh: { translation: zh },
  en: { translation: en },
} as const

/**
 * Initializes the shared i18next instance. The initial language comes from
 * the Zustand UI store (already rehydrated from localStorage), so a returning
 * English user never sees a flash of the default language.
 *
 * Language ownership stays in the Zustand UI store; I18nProvider syncs store
 * changes into i18next via changeLanguage.
 */
export function initI18n() {
  return i18next.use(initReactI18next).init({
    resources: i18nResources,
    lng: useUIStore.getState().language,
    fallbackLng: "zh",
    keySeparator: false,
    interpolation: {
      escapeValue: false, // React escapes by itself
    },
  })
}
