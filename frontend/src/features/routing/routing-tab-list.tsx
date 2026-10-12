import { Database, ListFilter, Network, ScanSearch } from "lucide-react"
import { useTranslation } from "react-i18next"
import { TabsList, TabsTrigger } from "../../components/ui/tabs"
import { routingTabs } from "./routing-tabs"

const tabIcons = { defaults: Network, rules: ListFilter, catalog: Database, preview: ScanSearch }
const tabLabels = {
  defaults: "routing.tabs.defaults",
  rules: "routing.tabs.rules",
  catalog: "routing.tabs.catalog",
  preview: "routing.tabs.preview",
} as const

export function RoutingTabList() {
  const { t } = useTranslation()
  return (
    <TabsList aria-label={t("routing.tabs.title")}>
      {routingTabs.map((value) => {
        const Icon = tabIcons[value]
        return (
          <TabsTrigger key={value} value={value}>
            <Icon aria-hidden="true" />
            {t(tabLabels[value])}
          </TabsTrigger>
        )
      })}
    </TabsList>
  )
}
