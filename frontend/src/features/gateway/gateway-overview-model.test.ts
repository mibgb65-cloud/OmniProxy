import { expect, it } from "vitest"
import { dashboardRange } from "../dashboard/dashboard-model"
import { filterGatewayRecent, gatewayTrend } from "./gateway-overview-model"
import { gatewayStatisticsFixture } from "./gateway-overview-test-data"

it("空趋势及单点趋势不生成无效坐标", () => {
  expect(gatewayTrend([]).success).toBe("")
  const points = gatewayStatisticsFixture(dashboardRange(1)).points ?? []
  const single = gatewayTrend(points.slice(0, 1))
  expect(single.success).not.toMatch(/NaN|Infinity/)
  expect(single.maximum).toBeGreaterThanOrEqual(2)
  expect(single.maximum % 4).toBe(0)
})

it("趋势峰值按成功与失败总量，筛选最近请求保持原记录不变", () => {
  const data = gatewayStatisticsFixture(dashboardRange(1))
  expect(gatewayTrend(data.points ?? []).peak?.summary.requests).toBe(3)
  const items = data.recent ?? []
  expect(filterGatewayRecent(items, "deepseek", "success")).toHaveLength(2)
  expect(filterGatewayRecent(items, "river", "failed")).toHaveLength(1)
  expect(filterGatewayRecent(items, "[missing]", "success")).toHaveLength(0)
  expect(items).toHaveLength(4)
})
