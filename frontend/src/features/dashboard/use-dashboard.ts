import { useQuery } from "@tanstack/react-query"
import { useState } from "react"
import { listRoutes } from "../../services/routing"
import { listAgentClients } from "../../services/agents"
import { useAccounts } from "../accounts/use-accounts"
import { useProviders } from "../providers/use-providers"
import { dashboardIssues } from "./dashboard-model"
import { useDashboardResources } from "./use-dashboard-resources"
import { useDashboardStatistics } from "./use-dashboard-statistics"

export function useDashboard() {
  const { period, setPeriod, now, statistics } = useDashboardStatistics()
  const [refreshing, setRefreshing] = useState(false)
  const accounts = useAccounts()
  const providers = useProviders()
  const routes = useQuery({ queryKey: ["routes"], queryFn: listRoutes })
  const agents = useQuery({ queryKey: ["agent-clients"], queryFn: listAgentClients })
  const resources = useDashboardResources(accounts.data ?? [], providers.data ?? [], now)
  async function refresh() {
    if (refreshing) return
    setRefreshing(true)
    try {
      await Promise.all([
        statistics.refetch({ cancelRefetch: false }),
        accounts.refetch({ cancelRefetch: false }),
        providers.refetch({ cancelRefetch: false }),
        routes.refetch({ cancelRefetch: false }),
        agents.refetch({ cancelRefetch: false }),
      ])
    } finally {
      setRefreshing(false)
    }
  }
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
    refresh,
    refreshing: refreshing || statistics.isFetching,
    issues: dashboardIssues(
      accounts.data ?? [],
      providers.data ?? [],
      routes.data ?? [],
      resources.entries,
    ),
    metadataError: accounts.isError || providers.isError || routes.isError,
    metadataPending: accounts.isPending || providers.isPending || routes.isPending,
  }
}

export type DashboardData = ReturnType<typeof useDashboard>
