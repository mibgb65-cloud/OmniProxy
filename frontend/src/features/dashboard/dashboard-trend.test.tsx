import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"
import type { DashboardInfo, DashboardPoint, DashboardSummary } from "../../services/dashboard"
import { DashboardTrend } from "./dashboard-trend"

const empty: DashboardSummary = {
  requests: 0,
  success: 0,
  failed: 0,
  cancelled: 0,
  pending: 0,
  inputTokens: "0",
  outputTokens: "0",
  totalTokens: "0",
  completeUsage: 0,
  successRate: null,
  averageLatencyMs: null,
}

// 本地零点：横轴小时标签必须与本地时区无关。
const midnight = new Date(2026, 9, 9, 0, 0, 0, 0).getTime() / 1000

function point(index: number, counts: Partial<DashboardSummary> = {}): DashboardPoint {
  const summary = { ...empty, ...counts } as DashboardSummary
  return {
    start: midnight + index * 3600,
    end: midnight + (index + 1) * 3600,
    summary: {
      ...summary,
      requests: summary.success + summary.failed + summary.cancelled + summary.pending,
      completeUsage: summary.success,
    },
  }
}

function info(points: DashboardPoint[], summary = empty): DashboardInfo {
  return {
    summary,
    points,
    recent: [],
    historyIncomplete: false,
    capturedFrom: midnight,
    updatedAt: "2026-10-09T00:00:00Z",
  }
}

describe("dashboard trend chart", () => {
  it("renders readable axis ticks and keeps the last hour label", () => {
    const points = Array.from({ length: 24 }, (_, index) =>
      index === 0 ? point(index, { success: 584, failed: 4 }) : point(index),
    )
    render(<DashboardTrend data={info(points, points[0]!.summary)} period={1} loading={false} />)
    expect(screen.getByRole("group", { name: "调用趋势" })).toBeInTheDocument()
    for (const tick of ["800", "600", "400", "200"]) {
      expect(screen.getByText(tick)).toBeInTheDocument()
    }
    expect(screen.getAllByText("0").length).toBeGreaterThan(0)
    for (const hour of ["00", "05", "10", "15", "20", "23"]) {
      expect(screen.getByText(hour)).toBeInTheDocument()
    }
    // 每列仍保留精确值的可访问名称，悬浮与键盘焦点共用同一份内容。
    expect(screen.getByLabelText(/请求数 588/)).toBeInTheDocument()
  })

  it("uses calendar labels and only the axis ticks for a week", () => {
    const points = Array.from({ length: 7 }, (_, index) => point(index * 24, { success: 12 }))
    render(<DashboardTrend data={info(points, points[0]!.summary)} period={7} loading={false} />)
    expect(screen.getByText("16")).toBeInTheDocument()
    expect(screen.getAllByText(/^10\/\d{2}$/).length).toBeGreaterThanOrEqual(2)
  })

  it("scrolls the plot with the rail thumb instead of the native scrollbar", () => {
    const points = Array.from({ length: 30 }, (_, index) =>
      index === 0 ? point(index * 24, { success: 40 }) : point(index * 24),
    )
    render(<DashboardTrend data={info(points, points[0]!.summary)} period={30} loading={false} />)
    const viewport = screen.getByRole("group", { name: "调用趋势" })
    const rail = viewport.nextElementSibling as HTMLDivElement
    const thumb = rail.firstElementChild as HTMLDivElement
    Object.defineProperties(viewport, {
      clientWidth: { configurable: true, value: 300 },
      scrollWidth: { configurable: true, value: 600 },
    })
    Object.defineProperty(rail, "clientWidth", { configurable: true, value: 300 })
    Object.defineProperty(thumb, "offsetWidth", {
      configurable: true,
      get: () => Number.parseFloat(thumb.style.width) || 0,
    })

    fireEvent.scroll(viewport)
    expect(rail).toHaveAttribute("data-scrollable", "true")
    expect(thumb.style.width).toBe("150px")

    viewport.scrollLeft = 150
    fireEvent.scroll(viewport)
    expect(thumb.style.transform).toBe("translateX(75px)")

    rail.getBoundingClientRect = () => ({ left: 0 }) as DOMRect
    fireEvent.pointerDown(rail, { button: 0, pointerId: 1, clientX: 225 })
    expect(viewport.scrollLeft).toBe(300)
    expect(rail).toHaveAttribute("data-dragging", "true")
    fireEvent.pointerUp(rail, { pointerId: 1 })
    expect(rail).not.toHaveAttribute("data-dragging")
  })

  it("drops the chart and shows the empty message when nothing was recorded", () => {
    render(<DashboardTrend data={info([point(0), point(1)])} period={1} loading={false} />)
    expect(screen.getByText("所选时段暂无已记录的调用。")).toBeInTheDocument()
    expect(screen.queryByRole("group", { name: "调用趋势" })).not.toBeInTheDocument()
  })
})
