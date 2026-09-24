/** Frontend-only UI types. Backend models come from the generated Wails bindings (frontend/bindings). */

import type { LucideIcon } from "lucide-react"

export interface NavItem {
  /** Router path, e.g. "/accounts". */
  to: string
  /** Sidebar label. */
  label: string
  /** Lucide icon component. */
  icon: LucideIcon
}
