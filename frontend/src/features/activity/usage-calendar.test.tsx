import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { usageDateKey } from "./usage-model"
import { UsageCalendar } from "./usage-calendar"

function today(offset = 0) {
  const date = new Date()
  date.setDate(date.getDate() + offset)
  return usageDateKey(date)
}

const days = [
  { day: today(), inputTokens: "1200", outputTokens: "300", totalTokens: "1500", requests: 4 },
  { day: today(-1), inputTokens: "20", outputTokens: "5", totalTokens: "25", requests: 1 },
]

beforeEach(() => {
  useUIStore.setState({ language: "zh" })
})

describe("yearly activity calendar", () => {
  it("summarizes the past year and keeps one cell per day", () => {
    render(
      <I18nProvider>
        <UsageCalendar days={days} />
      </I18nProvider>,
    )
    expect(screen.getByRole("heading", { name: "年度活跃度" })).toBeInTheDocument()
    expect(screen.getByText("过去一年 1,525 Token · 5 次请求")).toBeInTheDocument()
    expect(screen.getByRole("img", { name: "过去一年 1,525 Token · 5 次请求" })).toBeInTheDocument()
    // 53 周 × 7 天；未来日期也占位，保持列对齐。星期标签在热图之外单独成列。
    const grid = screen.getByRole("img", { name: /过去一年/ })
    expect(grid.children).toHaveLength(53 * 7)
    expect(screen.getByText("周一")).toBeInTheDocument()
    expect(screen.getByText("周五")).toBeInTheDocument()
    expect(screen.queryByText("周二")).not.toBeInTheDocument()
  })

  it("switches the metric without losing the recorded days", async () => {
    const user = userEvent.setup()
    render(
      <I18nProvider>
        <UsageCalendar days={days} />
      </I18nProvider>,
    )
    const tokens = screen.getByRole("button", { name: "Token" })
    const requests = screen.getByRole("button", { name: "请求数" })
    expect(tokens).toHaveAttribute("aria-pressed", "true")
    await user.click(requests)
    expect(requests).toHaveAttribute("aria-pressed", "true")
    expect(tokens).toHaveAttribute("aria-pressed", "false")
    // 合计口径不随颜色档位切换而改变。
    expect(screen.getByText("过去一年 1,525 Token · 5 次请求")).toBeInTheDocument()
  })
})
