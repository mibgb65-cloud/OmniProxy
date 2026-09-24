import "i18next"

import type { zh } from "./locales/zh"

/**
 * Typed i18next resources: `t()` only accepts keys that exist in the Chinese
 * dictionary (the source of truth), with editor autocomplete. A missing key
 * in `en` is still caught by `Record<keyof typeof zh, string>` in en.ts.
 */
declare module "i18next" {
  interface CustomTypeOptions {
    defaultNS: "translation"
    resources: {
      translation: typeof zh
    }
  }
}
