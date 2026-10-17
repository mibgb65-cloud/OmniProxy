import { act, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { Events } from "@wailsio/runtime"
import { beforeEach, expect, it, vi } from "vitest"
import { getAppInfo, listAccounts, listProviders } from "../../services/omniproxy"
import type { DashboardInfo } from "../../services/dashboard"
import { useUIStore } from "../../stores/ui-store"
import {
  accounts,
  appInfo,
  providers,
  readDashboardMock,
  renderGateway,
  resetGatewayRuntime,
} from "./gateway-test-helpers"
import { gatewayStatisticsFixture } from "./gateway-overview-test-data"

vi.mock("../../services/gateway", () => ({
  getGatewayInfo: vi.fn(),
  startGateway: vi.fn(),
  stopGateway: vi.fn(),
}))
vi.mock("../../services/omniproxy", () => ({
  getAppInfo: vi.fn(),
  listProviders: vi.fn(),
  listAccounts: vi.fn(),
}))
vi.mock("@wailsio/runtime", () => ({
  Clipboard: { SetText: vi.fn() },
  Events: { On: vi.fn(() => vi.fn()) },
}))

beforeEach(() => {
  vi.clearAllMocks()
  resetGatewayRuntime()
  useUIStore.setState({ language: "zh" })
  vi.mocked(getAppInfo).mockResolvedValue(appInfo)
  vi.mocked(listProviders).mockResolvedValue(providers)
  vi.mocked(listAccounts).mockResolvedValue(accounts)
  readDashboardMock.mockImplementation(async (query) => gatewayStatisticsFixture(query))
})

it("真实指标分别展示首字、完整耗时、成功率、跨时段在途及累计路由分布", async () => {
  const { router } = renderGateway()
  const user = userEvent.setup()
  expect(
    await within(await screen.findByRole("group", { name: "平均首字耗时" })).findByText("80 ms"),
  ).toBeInTheDocument()
  expect(
    within(screen.getByRole("group", { name: "平均首字耗时" })).getByText("成功流式样本 2 个"),
  ).toBeInTheDocument()
  expect(
    within(screen.getByRole("group", { name: "平均总耗时" })).getByText("1.2 s"),
  ).toBeInTheDocument()
  expect(
    within(screen.getByRole("group", { name: "请求成功率" })).getByText("66.7%"),
  ).toBeInTheDocument()
  expect(
    within(screen.getByRole("group", { name: "当前进行中" })).getByText("5"),
  ).toBeInTheDocument()
  expect(within(screen.getByRole("group", { name: "请求总量" })).getByText("4")).toBeInTheDocument()
  expect(screen.getByText("全部已记录路由请求，不随时段切换。")).toBeInTheDocument()
  expect(screen.getByText("70%")).toBeInTheDocument()
  expect(screen.getByRole("table")).toHaveTextContent("160 ms")
  expect(screen.getByRole("table")).toHaveTextContent("0 ms")
  await user.click(screen.getByRole("link", { name: `查看请求 ${"1".repeat(32)}` }))
  await waitFor(() => expect(router.state.location.search).toEqual({ requestId: "1".repeat(32) }))
})

it("近 7／30 天使用同一时段查询，最近记录筛选不触发模型请求或额外查询", async () => {
  const user = userEvent.setup()
  renderGateway()
  await screen.findByText("80 ms")
  await user.type(screen.getByRole("textbox", { name: "筛选最近请求的模型或提供商" }), "RIVER")
  expect(within(screen.getByRole("table")).getAllByRole("row")).toHaveLength(2)
  expect(readDashboardMock).toHaveBeenCalledTimes(1)
  await user.click(screen.getByRole("button", { name: "近 7 天" }))
  await waitFor(() => expect(readDashboardMock.mock.lastCall?.[0].boundaries).toHaveLength(8))
  await user.click(screen.getByRole("button", { name: "近 30 天" }))
  await waitFor(() => expect(readDashboardMock.mock.lastCall?.[0].boundaries).toHaveLength(31))
})

it("空白／未知首字保持未知，首次读取不将未知请求数显示为零", async () => {
  readDashboardMock.mockReturnValue(new Promise(() => {}))
  renderGateway()
  const first = await screen.findByRole("group", { name: "平均首字耗时" })
  expect(within(first).getByText("—")).toBeInTheDocument()
  expect(
    within(screen.getByRole("group", { name: "请求总量" })).queryByText("0"),
  ).not.toBeInTheDocument()
  expect(screen.queryByText("0 ms")).not.toBeInTheDocument()
})

it("本地统计失败可手动恢复，不显示原始错误", async () => {
  readDashboardMock.mockRejectedValueOnce(new Error("private database detail"))
  const user = userEvent.setup()
  renderGateway()
  expect(await screen.findByText("无法读取请求统计，请刷新重试。")).toHaveAttribute("role", "alert")
  expect(screen.queryByText(/private database/)).not.toBeInTheDocument()
  await user.click(screen.getByRole("button", { name: "刷新统计" }))
  expect(await screen.findByText("80 ms")).toBeInTheDocument()
})

it("请求事件刷新统计不使刷新按钮闪动，切走页面后保留缓存而不继续读取", async () => {
  const user = userEvent.setup()
  renderGateway()
  await screen.findByText("80 ms")
  let resolve!: (data: DashboardInfo) => void
  readDashboardMock.mockReturnValue(
    new Promise<DashboardInfo>((done) => {
      resolve = done
    }),
  )
  const entry = vi.mocked(Events.On).mock.calls.find((call) => call[0] === "activity:updated")
  if (!entry) throw new Error("missing local activity subscription")
  await act(async () => {
    entry[1]({ name: "activity:updated", data: null })
  })
  const button = screen.getByRole("button", { name: "刷新统计" })
  expect(button).toHaveAttribute("aria-busy", "false")
  expect(button).toBeEnabled()
  const query = readDashboardMock.mock.calls[0][0]
  await act(async () => resolve(gatewayStatisticsFixture(query)))
  await user.click(screen.getByRole("tab", { name: "手动接入" }))
  const count = readDashboardMock.mock.calls.length
  await act(async () => {
    entry[1]({ name: "activity:updated", data: null })
  })
  expect(readDashboardMock).toHaveBeenCalledTimes(count)
})

it("英文页面展示对应统计名称与有效零耗时", async () => {
  useUIStore.setState({ language: "en" })
  renderGateway()
  expect(
    await within(await screen.findByRole("group", { name: "Average first token" })).findByText(
      "80 ms",
    ),
  ).toBeInTheDocument()
  expect(screen.getByRole("table")).toHaveTextContent("0 ms")
})
