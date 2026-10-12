import { useQuery, useQueryClient } from "@tanstack/react-query"
import { Events } from "@wailsio/runtime"
import { useEffect, useState } from "react"
import { readDashboard } from "../../services/dashboard"
import { listRoutes } from "../../services/routing"
import { listAgentClients } from "../../services/agents"
import { useAccounts } from "../accounts/use-accounts"
import { useProviders } from "../providers/use-providers"
import { dashboardIssues, dashboardRange, type DashboardPeriod } from "./dashboard-model"
import { useDashboardResources } from "./use-dashboard-resources"

export function useDashboard() {
  const client = useQueryClient()
  const [period, setPeriod] = useState<DashboardPeriod>(1)
  const [now, setNow] = useState(() => Date.now())
  const [refreshing, setRefreshing] = useState(false)
  const range = dashboardRange(period, new Date(now))
  const statistics = useQuery({
    queryKey: ["dashboard-statistics", range.boundaries],
    queryFn: () => readDashboard(range),
    retry: false,
    staleTime: 5_000,
  })
  const accounts = useAccounts()
  const providers = useProviders()
  const routes = useQuery({ queryKey: ["routes"], queryFn: listRoutes })
  const agents = useQuery({ queryKey: ["agent-clients"], queryFn: listAgentClients })
  const resources = useDashboardResources(accounts.data ?? [], providers.data ?? [], now)
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 30_000)
    return () => clearInterval(timer)
  }, [])
  useEffect(() => {
    const off = Events.On("activity:updated", () => {
      void client.invalidateQueries(
        {
          queryKey: ["dashboard-statistics"],
          refetchType: document.visibilityState === "hidden" ? "none" : "active",
        },
        { cancelRefetch: false },
      )
    })
    const visible = () => {
      if (document.visibilityState === "visible")
        void client.refetchQueries(
          { queryKey: ["dashboard-statistics"], type: "active", stale: true },
          { cancelRefetch: false },
        )
    }
    document.addEventListener("visibilitychange", visible)
    return () => {
      off()
      document.removeEventListener("visibilitychange", visible)
    }
  }, [client])
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
