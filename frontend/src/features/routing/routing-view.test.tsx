import { selectOption } from "../../test/select-option"
import { act, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { getGatewayInfo } from "../../services/gateway"
import { listProviders, listAccounts } from "../../services/omniproxy"
import {
  createRoute,
  updateRoute,
  setRouteEnabled,
  deleteRoute,
  listRoutes,
  previewRoute,
  type RouteItem,
} from "../../services/routing"
import { useUIStore } from "../../stores/ui-store"
import { providers, accounts, ruleItem, renderRouting } from "./routing-test-helpers"

vi.mock("../../services/gateway", () => ({ getGatewayInfo: vi.fn() }))
vi.mock("../../services/omniproxy", () => ({ listProviders: vi.fn(), listAccounts: vi.fn() }))
vi.mock("../../services/routing", () => ({
  createRoute: vi.fn(),
  updateRoute: vi.fn(),
  setRouteEnabled: vi.fn(),
  deleteRoute: vi.fn(),
  listRoutes: vi.fn(),
  listDefaultRoutes: vi.fn().mockResolvedValue([]),
  setModelPreferredProvider: vi.fn(),
  previewRoute: vi.fn(),
}))

let saved: RouteItem[] = []

describe("路由页面", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    useUIStore.setState({ language: "zh" })
    saved = []
    vi.mocked(getGatewayInfo).mockResolvedValue({
      runtime: { state: "stopped", address: "", port: 3001, startedAt: "", errorCode: "" },
      models: ["deepseek-flash", "deepseek-v4-pro"],
      protocol: "Chat Completions",
      requestTimeoutSeconds: 120,
    })
    vi.mocked(listProviders).mockResolvedValue(providers)
    vi.mocked(listAccounts).mockResolvedValue(accounts)
    vi.mocked(listRoutes).mockImplementation(async () => saved)
    vi.mocked(createRoute).mockImplementation(async (input) => {
      const item = { ...ruleItem, rule: { ...ruleItem.rule, ...input } }
      saved = [item]
      return item.rule
    })
    vi.mocked(updateRoute).mockImplementation(async (id, input) => {
      const item = { ...ruleItem, rule: { ...ruleItem.rule, id, ...input } }
      saved = [item]
      return item.rule
    })
    vi.mocked(setRouteEnabled).mockImplementation(async (id, enabled) => {
      saved = saved.map((item) => ({ ...item, rule: { ...item.rule, enabled } }))
      return { ...ruleItem.rule, id, enabled }
    })
    vi.mocked(deleteRoute).mockImplementation(async () => {
      saved = []
    })
    vi.mocked(previewRoute).mockResolvedValue({
      inputModel: "coding-fast",
      matched: true,
      ruleId: "route_one",
      model: "deepseek-flash",
      providerId: "builtin_deepseek",
      providerName: "DeepSeek",
      accountId: "acct_one",
      accountName: "My account",
      candidateCount: 1,
      routable: true,
      errorCode: "",
    })
  })

  it("空状态不创建示例数据，新增规则后刷新列表并恢复焦点", async () => {
    const user = userEvent.setup()
    renderRouting("rules")
    expect(await screen.findByText("尚未添加自定义规则")).toBeInTheDocument()
    expect(createRoute).not.toHaveBeenCalled()
    const add = screen.getAllByRole("button", { name: "新增规则" })[0]
    await user.click(add)
    const dialog = screen.getByRole("dialog")
    const alias = within(dialog).getByRole("textbox", { name: "客户端模型名" })
    expect(alias).toHaveFocus()
    await user.type(alias, "coding-fast")
    await selectOption(
      user,
      within(dialog).getByRole("combobox", { name: "提供商配置" }),
      "DeepSeek",
    )
    await selectOption(
      user,
      within(dialog).getByRole("combobox", { name: "实际模型 ID" }),
      "deepseek-flash",
    )
    await selectOption(
      user,
      within(dialog).getByRole("combobox", { name: "账号范围" }),
      "My account",
    )
    await user.click(within(dialog).getByRole("button", { name: "保存规则" }))
    await waitFor(() =>
      expect(createRoute).toHaveBeenCalledWith({
        alias: "coding-fast",
        providerId: "builtin_deepseek",
        model: "deepseek-flash",
        accountId: "acct_one",
        enabled: true,
      }),
    )
    expect(await screen.findByRole("table", { name: "模型映射规则" })).toHaveTextContent(
      "coding-fast",
    )
    await waitFor(() => expect(add).toHaveFocus())
  })

  it("重复名称留在弹窗内并定位输入，不显示原始错误", async () => {
    const user = userEvent.setup()
    saved = [ruleItem]
    vi.mocked(updateRoute).mockRejectedValueOnce(
      new Error("routing: duplicate alias private details"),
    )
    renderRouting("rules")
    await user.click(await screen.findByRole("button", { name: "编辑规则“coding-fast”" }))
    const dialog = screen.getByRole("dialog")
    await user.click(within(dialog).getByRole("button", { name: "保存规则" }))
    expect(await screen.findByText("此模型名已有规则，请使用其他名称。")).toBeInTheDocument()
    expect(within(dialog).getByRole("textbox", { name: "客户端模型名" })).toHaveFocus()
    expect(screen.queryByText(/private details/)).not.toBeInTheDocument()
  })

  it("编辑模型并切换开关，规则筛选与搜索反映实际数据", async () => {
    const user = userEvent.setup()
    saved = [ruleItem]
    renderRouting("rules")
    await user.click(await screen.findByRole("button", { name: "编辑规则“coding-fast”" }))
    const dialog = screen.getByRole("dialog")
    await selectOption(
      user,
      within(dialog).getByRole("combobox", { name: "实际模型 ID" }),
      "deepseek-v4-pro",
    )
    await user.click(within(dialog).getByRole("button", { name: "保存规则" }))
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
    expect(screen.getByRole("table")).toHaveTextContent("deepseek-v4-pro")
    await user.click(screen.getByRole("switch", { name: "切换规则“coding-fast”的启用状态" }))
    await waitFor(() => expect(screen.getByRole("switch")).not.toBeChecked())
    await selectOption(user, screen.getByRole("combobox", { name: "规则状态" }), "已启用")
    expect(screen.getByText("没有符合筛选条件的规则。")).toBeInTheDocument()
    await selectOption(user, screen.getByRole("combobox", { name: "规则状态" }), "全部规则")
    await user.type(screen.getByRole("textbox", { name: "搜索路由规则" }), "missing")
    expect(screen.getByText("没有符合筛选条件的规则。")).toBeInTheDocument()
  })

  it("删除须确认，取消不调用接口", async () => {
    const user = userEvent.setup()
    saved = [ruleItem]
    renderRouting("rules")
    await user.click(await screen.findByRole("button", { name: "删除规则“coding-fast”" }))
    await user.click(screen.getByRole("button", { name: "取消" }))
    expect(deleteRoute).not.toHaveBeenCalled()
    await user.click(screen.getByRole("button", { name: "删除规则“coding-fast”" }))
    await user.click(screen.getByRole("button", { name: "确认删除" }))
    expect(await screen.findByText("尚未添加自定义规则")).toBeInTheDocument()
  })

  it("匹配预览只提交模型名，展示目标和账号元数据", async () => {
    const user = userEvent.setup()
    renderRouting("preview")
    await user.type(await screen.findByRole("textbox", { name: "客户端模型名" }), "coding-fast")
    await user.click(screen.getByRole("button", { name: "预览匹配" }))
    expect(await screen.findByText("命中精确规则")).toBeInTheDocument()
    expect(screen.getByText("当前选择：My account")).toBeInTheDocument()
    expect(previewRoute).toHaveBeenCalledWith("coding-fast")
  })

  it("读取失败显示重试，不误报空规则", async () => {
    const user = userEvent.setup()
    vi.mocked(listRoutes).mockRejectedValueOnce(new Error("private database error"))
    renderRouting("rules")
    expect(await screen.findByRole("alert")).toHaveTextContent("无法读取路由配置")
    expect(screen.queryByText("尚未添加自定义规则")).not.toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "刷新路由配置" }))
    expect(await screen.findByText("尚未添加自定义规则")).toBeInTheDocument()
  })

  it("异常规则显示具体原因并支持英文", async () => {
    saved = [{ ...ruleItem, errorCode: "account_disabled" }]
    renderRouting("rules")
    expect(await screen.findByText("固定账号已停用")).toBeInTheDocument()
    act(() => useUIStore.setState({ language: "en" }))
    expect(await screen.findByRole("heading", { name: "Routing" })).toBeInTheDocument()
    expect(screen.getByText("Fixed account is disabled")).toBeInTheDocument()
  })
})
