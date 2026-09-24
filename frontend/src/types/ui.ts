/** Frontend-only UI types. Backend models come from the generated Wails bindings (frontend/bindings). */

import type { LucideIcon } from "lucide-react"
import type { TranslationKey } from "../i18n/translation"

export interface NavItem {
  /** Router path, e.g. "/accounts". */
  to: string
  /** Translation key for the sidebar label; never a hardcoded string. */
  labelKey: TranslationKey
  /** Lucide icon component. */
  icon: LucideIcon
}
