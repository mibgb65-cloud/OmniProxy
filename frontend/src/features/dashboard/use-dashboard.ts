import { useQuery } from "@tanstack/react-query"
import { listRoutes } from "../../services/routing"
import { listAgentClients } from "../../services/agents"
import { useAccounts } from "../accounts/use-accounts"
import { useProviders } from "../providers/use-providers"
import { dashboardChecklist, dashboardIssues } from "./dashboard-model"
import { useDashboardResources } from "./use-dashboard-resources"
import { useDashboardStatistics } from "./use-dashboard-statistics"

export function useDashboard() {
  const { period, setPeriod, now, statistics } = useDashboardStatistics()
  const accounts = useAccounts()
  const providers = useProviders()
  const routes = useQuery({ queryKey: ["routes"], queryFn: listRoutes })
  const agents = useQuery({ queryKey: ["agent-clients"], queryFn: listAgentClients })
  const resources = useDashboardResources(accounts.data ?? [], providers.data ?? [], now)
  const accountList = accounts.data ?? []
  const providerList = providers.data ?? []
  const routeList = routes.data ?? []
  return {
    period,
    setPeriod,
    now,
    statistics,
    resources,
    accounts,
    providers,
    routes,
    agents,
    issues: dashboardIssues(accountList, providerList, routeList, resources.entries),
    checklist: dashboardChecklist(accountList, providerList, routeList, resources.entries),
    metadataError: accounts.isError || providers.isError || routes.isError,
    metadataPending: accounts.isPending || providers.isPending || routes.isPending,
  }
}

export type DashboardData = ReturnType<typeof useDashboard>
