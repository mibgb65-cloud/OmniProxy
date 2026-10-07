import { create } from "zustand"
import { persist } from "zustand/middleware"
import { defaultLanguage, isLanguage, type Language } from "../i18n/translation"
import { defaultTheme, isTheme, type Theme } from "../lib/theme"

export type AccountView = "cards" | "list"

export function isAccountView(value: unknown): value is AccountView {
  return value === "cards" || value === "list"
}

interface UIState {
  sidebarCollapsed: boolean
  toggleSidebar: () => void
  /** Selected UI language; persisted to localStorage. */
  language: Language
  setLanguage: (language: Language) => void
  /** Selected colour theme; persisted to localStorage. */
  theme: Theme
  toggleTheme: () => void
  /** 账号详情的展示方式；持久化保存，不包含账号数据。 */
  accountView: AccountView
  setAccountView: (view: AccountView) => void
}

/**
 * UI state only (sidebar, language, theme, account view). Backend data belongs in TanStack
 * Query. Persist preferences so they survive restarts.
 */
export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      sidebarCollapsed: false,
      toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),
      language: defaultLanguage,
      setLanguage: (language) => set({ language }),
      theme: defaultTheme,
      toggleTheme: () => set((s) => ({ theme: s.theme === "dark" ? "light" : "dark" })),
      accountView: "cards",
      setAccountView: (accountView) => set({ accountView }),
    }),
    {
      name: "omniproxy-ui-state",
      // Rehydrate defensively: a corrupted or foreign value cannot select an
      // unsupported language or theme.
      merge: (persisted, current) => {
        const stored = persisted as Partial<UIState> | undefined
        return {
          ...current,
          sidebarCollapsed:
            typeof stored?.sidebarCollapsed === "boolean"
              ? stored.sidebarCollapsed
              : current.sidebarCollapsed,
          language: isLanguage(stored?.language) ? stored.language : current.language,
          theme: isTheme(stored?.theme) ? stored.theme : current.theme,
          accountView: isAccountView(stored?.accountView)
            ? stored.accountView
            : current.accountView,
        }
      },
    },
  ),
)
