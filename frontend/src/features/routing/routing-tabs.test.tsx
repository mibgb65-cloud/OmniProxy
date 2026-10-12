import { act, isInaccessible, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { CancellablePromise } from "@wailsio/runtime"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { getGatewayInfo } from "../../services/gateway"
import { listAccounts, listProviders } from "../../services/omniproxy"
import { createRoute, listRoutes, previewRoute } from "../../services/routing"
import { useUIStore } from "../../stores/ui-store"
import { selectOption } from "../../test/select-option"
import {
  accounts,
  defaultMapping,
  providers,
  renderRouting,
  ruleItem,
  syncAccountModelsMock as syncAccountModels,
} from "./routing-test-helpers"
import { validateRoutingSearch } from "./routing-tabs"

vi.mock("../../services/gateway", () => ({ getGatewayInfo: vi.fn() }))
vi.mock("../../services/omniproxy", () => ({ listProviders: vi.fn(), listAccounts: vi.fn() }))
vi.mock("../../services/routing", () => ({
  createRoute: vi.fn(),
  updateRoute: vi.fn(),
  setRouteEnabled: vi.fn(),
  deleteRoute: vi.fn(),
  listRoutes: vi.fn(),
  listDefaultRoutes: vi.fn().mockImplementation(async () => [defaultMapping]),
  setModelPreferredProvider: vi.fn(),
  previewRoute: vi.fn(),
}))

describe("路由选项卡导航与状态保留", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useUIStore.setState({ language: "zh" })
    vi.mocked(listRoutes).mockResolvedValue([ruleItem])
    vi.mocked(listProviders).mockResolvedValue(providers)
    vi.mocked(listAccounts).mockResolvedValue(
      accounts.map((account) => ({ ...account, authMethod: "api_key" })),
    )
    vi.mocked(getGatewayInfo).mockResolvedValue({
      runtime: { state: "stopped", address: "", port: 3001, startedAt: "", errorCode: "" },
      models: ["deepseek-flash"],
      protocol: "Chat Completions",
      requestTimeoutSeconds: 120,
    })
  })

  it("默认只显示默认映射，移除说明，中英文页签保持一致", async () => {
    renderRouting("defaults", "/routing")
    expect(await screen.findByRole("tab", { name: "默认映射", selected: true })).toBeInTheDocument()
    expect(screen.getAllByRole("tab")).toHaveLength(4)
    expect(screen.getAllByRole("tabpanel")).toHaveLength(1)
    expect(screen.queryByRole("textbox", { name: "搜索路由规则" })).not.toBeInTheDocument()
    expect(screen.queryByRole("combobox", { name: "同步账号" })).not.toBeInTheDocument()
    expect(screen.queryByText(/自动生成默认模型映射，设置优先服务商/)).not.toBeInTheDocument()
    expect(syncAccountModels).not.toHaveBeenCalled()
    act(() => useUIStore.setState({ language: "en" }))
    expect(
      await screen.findByRole("tab", { name: "Default mappings", selected: true }),
    ).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Custom rules" })).toBeInTheDocument()
    expect(
      screen.queryByText(/Use automatic model mappings, choose preferred providers/),
    ).not.toBeInTheDocument()
  })

  it.each(["unknown", "%5B%22rules%22%5D", "x".repeat(300)])(
    "非法路由参数 %s 回到默认映射",
    async (tab) => {
      renderRouting("defaults", `/routing?tab=${tab}`)
      expect(
        await screen.findByRole("tab", { name: "默认映射", selected: true }),
      ).toBeInTheDocument()
      expect(screen.getAllByRole("tabpanel")).toHaveLength(1)
    },
  )

  it.each([undefined, null, 1, [], {}, ["rules"]])("非字符串参数 %j 安全回退", (tab) => {
    expect(validateRoutingSearch({ tab, __TSearchSchemaInput__: "TSearchSchemaInput" })).toEqual({
      tab: "defaults",
    })
  })

  it("直接定位预览，前进后退保留输入并隐藏非活动面板", async () => {
    const user = userEvent.setup()
    const { router } = renderRouting("preview")
    await user.type(await screen.findByRole("textbox", { name: "客户端模型名" }), "coding-fast")
    await user.click(screen.getByRole("tab", { name: "自定义规则" }))
    await waitFor(() => expect(router.state.location.search).toEqual({ tab: "rules" }))
    const panelId = screen.getByRole("tab", { name: "匹配预览" }).getAttribute("aria-controls")!
    expect(isInaccessible(document.getElementById(panelId)!)).toBe(true)
    expect(screen.queryByRole("textbox", { name: "客户端模型名" })).not.toBeInTheDocument()
    await act(async () => router.history.back())
    expect(await screen.findByRole("textbox", { name: "客户端模型名" })).toHaveValue("coding-fast")
    await act(async () => router.history.forward())
    expect(
      await screen.findByRole("tab", { name: "自定义规则", selected: true }),
    ).toBeInTheDocument()
  })

  it("方向键只移动焦点，Enter 激活且焦点不丢失", async () => {
    const user = userEvent.setup()
    renderRouting()
    const defaults = await screen.findByRole("tab", { name: "默认映射" })
    defaults.focus()
    await user.keyboard("{ArrowRight}")
    expect(screen.getByRole("tab", { name: "自定义规则" })).toHaveFocus()
    expect(defaults).toHaveAttribute("aria-selected", "true")
    await user.keyboard("{Enter}")
    expect(await screen.findByRole("tab", { name: "自定义规则", selected: true })).toHaveFocus()
  })

  it("各面板搜索和筛选往返切换后保留", async () => {
    const user = userEvent.setup()
    renderRouting()
    await user.type(await screen.findByRole("textbox", { name: "搜索默认模型映射" }), "missing")
    await user.click(screen.getByRole("tab", { name: "自定义规则" }))
    await user.type(await screen.findByRole("textbox", { name: "搜索路由规则" }), "coding")
    await selectOption(user, screen.getByRole("combobox", { name: "规则状态" }), "已停用")
    await user.click(screen.getByRole("tab", { name: "默认映射" }))
    expect(screen.getByRole("textbox", { name: "搜索默认模型映射" })).toHaveValue("missing")
    await user.click(screen.getByRole("tab", { name: "自定义规则" }))
    expect(screen.getByRole("textbox", { name: "搜索路由规则" })).toHaveValue("coding")
    expect(screen.getByRole("combobox", { name: "规则状态" })).toHaveTextContent("已停用")
    expect(screen.getByText("没有符合筛选条件的规则。")).toBeInTheDocument()
  })

  it("模型目录同步期间切换，返回后保留账号选择和结果", async () => {
    let finish!: (value: Awaited<ReturnType<typeof syncAccountModels>>) => void
    vi.mocked(syncAccountModels).mockReturnValue(
      new CancellablePromise((resolve) => {
        finish = resolve
      }),
    )
    const user = userEvent.setup()
    const { client } = renderRouting("catalog")
    await selectOption(
      user,
      await screen.findByRole("combobox", { name: "同步账号" }),
      "My account · DeepSeek",
    )
    await user.click(screen.getByRole("button", { name: "同步官方模型" }))
    expect(syncAccountModels).toHaveBeenCalledWith("acct_one")
    await user.click(screen.getByRole("tab", { name: "默认映射" }))
    await act(async () =>
      finish([
        {
          accountId: "acct_one",
          providerId: "builtin_deepseek",
          model: "deepseek-flash",
          name: "DeepSeek",
          fetchedAt: "now",
        },
      ]),
    )
    await waitFor(() => expect(client.getQueryData(["model-catalog"])).toHaveLength(1))
    await user.click(screen.getByRole("tab", { name: "模型目录" }))
    expect(screen.getByRole("combobox", { name: "同步账号" })).toHaveTextContent(
      "My account · DeepSeek",
    )
    expect(await screen.findByText("已同步 1 个账号模型")).toBeInTheDocument()
    expect(syncAccountModels).toHaveBeenCalledTimes(1)
  })

  it("预览期间切换，返回后保留输入和异步结果", async () => {
    let finish!: (value: Awaited<ReturnType<typeof previewRoute>>) => void
    vi.mocked(previewRoute).mockReturnValue(
      new Promise((resolve) => {
        finish = resolve
      }),
    )
    const user = userEvent.setup()
    renderRouting("preview")
    await user.type(await screen.findByRole("textbox", { name: "客户端模型名" }), "deepseek-flash")
    await user.click(screen.getByRole("button", { name: "预览匹配" }))
    await user.click(screen.getByRole("tab", { name: "默认映射" }))
    await act(async () => finish(defaultMapping.preview))
    await user.click(screen.getByRole("tab", { name: "匹配预览" }))
    expect(screen.getByRole("textbox", { name: "客户端模型名" })).toHaveValue("deepseek-flash")
    expect(await screen.findByText("当前选择：My account")).toBeInTheDocument()
    expect(previewRoute).toHaveBeenCalledTimes(1)
  })

  it("从默认映射新增时切到规则面板，取消后返回按钮焦点", async () => {
    const user = userEvent.setup()
    renderRouting()
    const add = await screen.findByRole("button", { name: "新增规则" })
    await waitFor(() => expect(add).toBeEnabled())
    await user.click(add)
    await user.click(screen.getByRole("button", { name: "取消" }))
    expect(
      await screen.findByRole("tab", { name: "自定义规则", selected: true }),
    ).toBeInTheDocument()
    await waitFor(() => expect(add).toHaveFocus())
    expect(createRoute).not.toHaveBeenCalled()
  })
})
