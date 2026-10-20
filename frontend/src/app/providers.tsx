import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { ReactNode } from "react"
import { I18nProvider } from "../i18n/provider"
import { ThemeProvider } from "./theme-provider"
import { TooltipProvider } from "../components/ui/tooltip"
import { TypographyProvider } from "./typography-provider"
import { useModelCatalogEvents } from "../features/gateway/use-model-catalog-events"

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
})

function ModelCatalogEvents() {
  useModelCatalogEvents()
  return null
}

export function AppProviders({ children }: { children: ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <ModelCatalogEvents />
      <I18nProvider>
        <ThemeProvider>
          <TypographyProvider>
            <TooltipProvider>{children}</TooltipProvider>
          </TypographyProvider>
        </ThemeProvider>
      </I18nProvider>
    </QueryClientProvider>
  )
}
