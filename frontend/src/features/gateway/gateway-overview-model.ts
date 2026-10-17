import type { DashboardPoint } from "../../services/dashboard"
import type { RequestItem } from "../../services/activity"

export function gatewayTrend(points: DashboardPoint[]) {
  const rawMaximum = Math.max(
    1,
    ...points.flatMap((point) => [point.summary.success, point.summary.failed]),
  )
  const maximum = Math.max(4, Math.ceil(rawMaximum / 4) * 4)
  const x = (index: number) => ((index + 0.5) / Math.max(1, points.length)) * 1000
  const y = (value: number) => 190 - (value / maximum) * 180
  const path = (key: "success" | "failed") =>
    points
      .map(
        (point, index) =>
          `${index === 0 ? "M" : "L"}${x(index).toFixed(2)},${y(point.summary[key]).toFixed(2)}`,
      )
      .join(" ")
  const success = path("success")
  const area = points.length
    ? `${success} L${x(points.length - 1).toFixed(2)},190 L${x(0).toFixed(2)},190 Z`
    : ""
  const peak = points.reduce<DashboardPoint | undefined>(
    (best, point) =>
      !best ||
      point.summary.success + point.summary.failed > best.summary.success + best.summary.failed
        ? point
        : best,
    undefined,
  )
  return { maximum, success, failed: path("failed"), area, peak, x }
}

export function filterGatewayRecent(items: RequestItem[], search: string, state: string) {
  const query = search.trim().toLocaleLowerCase()
  return items.filter(
    (item) =>
      (!state || item.state === state) &&
      (!query ||
        [item.model, item.inputModel, item.providerName].some((value) =>
          value.toLocaleLowerCase().includes(query),
        )),
  )
}
