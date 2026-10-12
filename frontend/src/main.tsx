import { StrictMode } from "react"
import { createRoot } from "react-dom/client"
import { RouterProvider } from "@tanstack/react-router"
import { AppProviders } from "./app/providers"
import { router } from "./app/router"
import { applyTheme } from "./lib/theme"
import { applyFontScale } from "./lib/typography"
import { useUIStore } from "./stores/ui-store"
import "./styles.css"

// Apply the persisted preference before React paints the first frame.
applyTheme(useUIStore.getState().theme)
applyFontScale(useUIStore.getState().fontScale)

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <AppProviders>
      <RouterProvider router={router} />
    </AppProviders>
  </StrictMode>,
)
