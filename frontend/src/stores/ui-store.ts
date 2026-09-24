import { create } from "zustand"
import { persist } from "zustand/middleware"
import { defaultLanguage, isLanguage, type Language } from "../i18n/translation"

interface UIState {
  sidebarCollapsed: boolean
  toggleSidebar: () => void
  /** Selected UI language; persisted to localStorage. */
  language: Language
  setLanguage: (language: Language) => void
}

/**
 * Non-persistent UI state only (sidebar, theme, selections). Backend data
 * belongs in TanStack Query, never in this store. The language preference is
 * persisted so the choice survives restarts.
 */
export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      sidebarCollapsed: false,
      toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),
      language: defaultLanguage,
      setLanguage: (language) => set({ language }),
    }),
    {
      name: "omniproxy-ui-state",
      // Rehydrate defensively: a corrupted or foreign value falls back to the
      // default language instead of breaking translations.
      merge: (persisted, current) => {
        const stored = (persisted as Partial<UIState> | undefined)?.language
        return {
          ...current,
          ...(persisted as Partial<UIState>),
          language: isLanguage(stored) ? stored : current.language,
        }
      },
    },
  ),
)
