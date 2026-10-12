import { Events } from "@wailsio/runtime"
import { act, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { readDashboard, type DashboardInfo } from "../../services/dashboard"
import { readSubscriptionQuota } from "../../services/subscription"
import { readAccountBalance } from "../../services/balance"
import { useUIStore } from "../../stores/ui-store"
import { subscriptionQuotaKey } from "../accounts/use-subscription-quota"
import {
  accounts,
  quota,
  renderDashboard,
  resetDashboardServices,
  snapshot,
} from "./dashboard-test-helpers"

vi.mock("../../services/dashboard", () => ({ readDashboard: vi.fn() }))
vi.mock("../../services/omniproxy", async (original) => ({
  ...(await original<typeof import("../../services/omniproxy")>()),
  listAccounts: vi.fn(),
  listProviders: vi.fn(),
}))
vi.mock("../../services/routing", async (original) => ({
  ...(await original<typeof import("../../services/routing")>()),
  listRoutes: vi.fn(),
}))
vi.mock("../../services/agents", async (original) => ({
  ...(await original<typeof import("../../services/agents")>()),
  listAgentClients: vi.fn(),
}))
vi.mock("../../services/subscription", async (original) => ({
  ...(await original<typeof import("../../services/subscription")>()),
  readSubscriptionQuota: vi.fn(),
}))
vi.mock("../../services/balance", async (original) => ({
  ...(await original<typeof import("../../services/balance")>()),
  readAccountBalance: vi.fn(),
}))
vi.mock("@wailsio/runtime", async (original) => ({
  ...(await original<typeof import("@wailsio/runtime")>()),
  Events: { On: vi.fn(() => vi.fn()) },
}))

describe("dashboard overview", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useUIStore.setState({ language: "zh" })
    resetDashboardServices()
  })

  it("renders accurate metrics and links without old diagnostic panels or automatic resource calls", async () => {
    renderDashboard()
    expect(await screen.findByText("80%")).toBeInTheDocument()
    expect(screen.getAllByText("1.5 s").length).toBeGreaterThan(0)
    expect(screen.getByText("900,719,925,474,099,314,345")).toBeInTheDocument()
    expect(screen.queryByText("绑定检查")).not.toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "测试后端连接" })).not.toBeInTheDocument()
    expect(screen.getByRole("link", { name: "查看请求 " + "a".repeat(32) })).toHaveAttribute(
      "href",
      "/requests?requestId=" + "a".repeat(32),
    )
    expect(await screen.findByText("1 条已启用路由配置异常")).toBeInTheDocument()
    expect(screen.getByText("订阅账号需要重新登录")).toBeInTheDocument()
    expect(readSubscriptionQuota).not.toHaveBeenCalled()
    expect(readAccountBalance).not.toHaveBeenCalled()
  })

  it("changes calendar ranges and chart mode without mixing the prior period's snapshot", async () => {
    const user = userEvent.setup()
    renderDashboard()
    await screen.findByText("80%")
    let finish!: (value: DashboardInfo) => void
    vi.mocked(readDashboard).mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve
        }),
    )
    await user.click(screen.getByRole("button", { name: "近 7 天" }))
    await waitFor(() => expect(finish).toBeDefined())
    expect(screen.queryByText("80%")).not.toBeInTheDocument()
    const query = vi.mocked(readDashboard).mock.calls.at(-1)![0]
    expect(query.boundaries).toHaveLength(8)
    await act(async () => {
      finish(snapshot(query))
    })
    await screen.findByText("80%")
    await user.click(screen.getByRole("button", { name: "Token" }))
    expect(screen.getByRole("button", { name: "Token" })).toHaveAttribute("aria-pressed", "true")
    expect(readDashboard).toHaveBeenCalledTimes(2)
  })

  it("shares resource caches and reads official data only when explicitly checked", async () => {
    const user = userEvent.setup()
    renderDashboard((client) => client.setQueryData(subscriptionQuotaKey(accounts[0]!), quota))
    await screen.findByText("80%")
    expect(await screen.findByText(/已缓存 1 \/ 5/)).toBeInTheDocument()
    expect(readSubscriptionQuota).not.toHaveBeenCalled()
    const button = screen.getByRole("button", { name: "检查资源" })
    await waitFor(() => expect(button).toBeEnabled())
    await user.click(button)
    expect(await screen.findByText("已检查 4 个账号，0 个查询失败。")).toBeInTheDocument()
    expect(readSubscriptionQuota).toHaveBeenCalledTimes(2)
    expect(readAccountBalance).toHaveBeenCalledTimes(2)
    expect(readSubscriptionQuota).not.toHaveBeenCalledWith("sub-out")
  })

  it("uses activity events to refresh statistics and cleans subscriptions up on leave", async () => {
    const off = vi.fn()
    vi.mocked(Events.On).mockReturnValue(off)
    const { unmount } = renderDashboard()
    await screen.findByText("80%")
    const callback = vi
      .mocked(Events.On)
      .mock.calls.find(([name]) => name === "activity:updated")?.[1]
    expect(callback).toBeDefined()
    await act(async () => {
      callback?.({ name: "activity:updated", data: "", sender: "" })
    })
    await waitFor(() => expect(readDashboard).toHaveBeenCalledTimes(2))
    unmount()
    expect(off).toHaveBeenCalledTimes(2)
  })

  it("does not fabricate empty success rates and marks incomplete history", async () => {
    vi.mocked(readDashboard).mockImplementationOnce(async (query) => {
      const data = snapshot(query)
      return {
        ...data,
        summary: {
          ...data.summary,
          requests: 0,
          success: 0,
          failed: 0,
          cancelled: 0,
          pending: 0,
          totalTokens: "0",
          inputTokens: "0",
          outputTokens: "0",
          completeUsage: 0,
          successRate: null,
          averageLatencyMs: null,
        },
        points: [],
        recent: [],
        historyIncomplete: true,
      }
    })
    renderDashboard()
    expect(await screen.findByText(/升级前部分明细已清理/)).toBeInTheDocument()
    expect(
      within(screen.getByText("成功率").closest('[data-slot="card"]')!).getByText("—"),
    ).toBeInTheDocument()
  })

  it("shows safe statistics failures and supports a local retry without checking resources", async () => {
    const user = userEvent.setup()
    vi.mocked(readDashboard).mockRejectedValueOnce(new Error("private SQL detail"))
    renderDashboard()
    expect(await screen.findByRole("alert")).toHaveTextContent("运行统计读取失败")
    expect(screen.queryByText(/private SQL detail/)).not.toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "刷新概况" }))
    expect(await screen.findByText("80%")).toBeInTheDocument()
    expect(readSubscriptionQuota).not.toHaveBeenCalled()
    expect(readAccountBalance).not.toHaveBeenCalled()
  })
})
