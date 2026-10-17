import { create } from "zustand"

export type AppPreferencesTab = "settings" | "about"

type AppPreferencesState = {
  open: boolean
  tab: AppPreferencesTab
  setOpen: (open: boolean) => void
  setTab: (tab: AppPreferencesTab) => void
  show: (tab: AppPreferencesTab) => void
}

// 弹窗只属于当前会话，不持久化，避免重启应用后意外遮住工作页面。
export const useAppPreferences = create<AppPreferencesState>((set) => ({
  open: false,
  tab: "settings",
  setOpen: (open) => set({ open }),
  setTab: (tab) => set({ tab }),
  show: (tab) => set({ open: true, tab }),
}))
