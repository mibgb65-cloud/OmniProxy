import { create } from "zustand"

interface UIState {
  sidebarCollapsed: boolean
  toggleSidebar: () => void
}

/**
 * Non-persistent UI state only (sidebar, theme, selections). Backend data
 * belongs in TanStack Query, never in this store.
 */
export const useUIStore = create<UIState>()((set) => ({
  sidebarCollapsed: false,
  toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),
}))
