import { render, screen, within } from "@testing-library/react"
import { beforeEach, describe, expect, it } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import type { DashboardModel } from "../../services/dashboard"
import { DashboardModelShare } from "./dashboard-model-share"

function model(name: string, tokens: string, requests: number): DashboardModel {
  return { model: name, inputTokens: tokens, outputTokens: "0", totalTokens: tokens, requests }
}

beforeEach(() => {
  useUIStore.setState({ language: "zh" })
})

describe("dashboard model share", () => {
  it("lists the leading models with their share and folds the rest into other", () => {
    const models = Array.from({ length: 8 }, (_, index) =>
      model(`model-${index + 1}`, String((8 - index) * 100), 8 - index),
    )
    render(
      <I18nProvider>
        <DashboardModelShare models={models} loading={false} />
      </I18nProvider>,
    )
    expect(screen.getByRole("heading", { name: "模型 Token 占比" })).toBeInTheDocument()
    expect(screen.getByText("model-1")).toBeInTheDocument()
    expect(screen.getByText("model-6")).toBeInTheDocument()
    expect(screen.queryByText("model-7")).not.toBeInTheDocument()
    // 余量：model-7 + model-8 = 200 + 100 Token，请求数 2 + 1。
    const other = screen.getByText("其他 2 个模型").closest("li")!
    expect(within(other).getByText("300")).toBeInTheDocument()
    expect(within(other).getByText("3 请求")).toBeInTheDocument()
  })

  it("keeps an honest empty state instead of zero shares", () => {
    const { rerender } = render(
      <I18nProvider>
        <DashboardModelShare models={[]} loading />
      </I18nProvider>,
    )
    expect(screen.getByText("正在读取运行统计…")).toBeInTheDocument()
    rerender(
      <I18nProvider>
        <DashboardModelShare models={[]} loading={false} />
      </I18nProvider>,
    )
    expect(screen.getByText("所选时段尚未收到 Token 用量报告。")).toBeInTheDocument()
    // 请求数已知但 Token 未报告时，不把 0 当成占比。
    rerender(
      <I18nProvider>
        <DashboardModelShare models={[model("model-1", "0", 3)]} loading={false} />
      </I18nProvider>,
    )
    expect(screen.getByText("所选时段尚未收到 Token 用量报告。")).toBeInTheDocument()
  })
})
