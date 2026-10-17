import { useNavigate } from "@tanstack/react-router"
import { useEffect } from "react"
import { useAppPreferences, type AppPreferencesTab } from "./app-preferences-state"

export function OpenPreferencesRoute({ tab }: { tab: AppPreferencesTab }) {
  const navigate = useNavigate()
  const show = useAppPreferences((state) => state.show)
  useEffect(() => {
    show(tab)
    void navigate({ to: "/", replace: true })
  }, [navigate, show, tab])
  return null
}
