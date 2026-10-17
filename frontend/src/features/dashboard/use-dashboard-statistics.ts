import { useQuery, useQueryClient } from "@tanstack/react-query"
import { Events } from "@wailsio/runtime"
import { useEffect, useState } from "react"
import { readDashboard } from "../../services/dashboard"
import { dashboardRange, type DashboardPeriod } from "./dashboard-model"

/** 首页和网关共用本地统计缓存；隐藏的网关面板不发起读取。 */
export function useDashboardStatistics(active = true) {
  const client = useQueryClient()
  const [period, setPeriod] = useState<DashboardPeriod>(1)
  const [now, setNow] = useState(() => Date.now())
  const range = dashboardRange(period, new Date(now))
  const statistics = useQuery({
    queryKey: ["dashboard-statistics", range.boundaries],
    queryFn: () => readDashboard(range),
    enabled: active,
    retry: false,
    staleTime: 5000,
  })
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 30000)
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
  return { period, setPeriod, now, statistics }
}
