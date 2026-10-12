import { selectOption } from "../../test/select-option"
import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { getGatewayInfo } from "../../services/gateway"
import { listProviders, listAccounts } from "../../services/omniproxy"
import { listRoutes, updateRoute, previewRoute } from "../../services/routing"
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

describe("路由异常边界", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    useUIStore.setState({ language: "zh" })
    vi.mocked(getGatewayInfo).mockResolvedValue({
      runtime: { state: "stopped", address: "", port: 3001, startedAt: "", errorCode: "" },
      models: ["deepseek-flash", "deepseek-v4-pro"],
      protocol: "Chat Completions",
      requestTimeoutSeconds: 120,
    })
    vi.mocked(listProviders).mockResolvedValue(providers)
    vi.mocked(listAccounts).mockResolvedValue(accounts)
    vi.mocked(listRoutes).mockResolvedValue([
      {
        ...ruleItem,
        rule: { ...ruleItem.rule, accountId: "deleted" },
        errorCode: "account_missing",
      },
    ])
    vi.mocked(updateRoute).mockResolvedValue(ruleItem.rule)
  })
  it("删除的固定账号保留异常选项，用户可切回账号范围修复", async () => {
    const user = userEvent.setup()
    renderRouting("rules")
    await user.click(await screen.findByRole("button", { name: "编辑规则“coding-fast”" }))
    const dialog = screen.getByRole("dialog")
    const account = within(dialog).getByRole("combobox", { name: "账号范围" })
    expect(account).toHaveTextContent("固定账号已删除")
    await selectOption(user, account, "提供商启用账号")
    await user.click(within(dialog).getByRole("button", { name: "保存规则" }))
    await waitFor(() =>
      expect(updateRoute).toHaveBeenCalledWith(
        "route_one",
        expect.objectContaining({ accountId: "" }),
      ),
    )
  })
  it("未知模型预览不误报直通成功", async () => {
    const user = userEvent.setup()
    vi.mocked(previewRoute).mockResolvedValue({
      inputModel: "unknown",
      matched: false,
      ruleId: "",
      model: "unknown",
      providerId: "",
      providerName: "",
      accountId: "",
      accountName: "",
      candidateCount: 0,
      routable: false,
      errorCode: "unsupported_model",
    })
    renderRouting("preview")
    await user.type(await screen.findByRole("textbox", { name: "客户端模型名" }), "unknown")
    await user.click(screen.getByRole("button", { name: "预览匹配" }))
    expect(await screen.findByText("未匹配可执行目标")).toBeInTheDocument()
    expect(screen.queryByText("原始模型直通")).not.toBeInTheDocument()
  })
})
