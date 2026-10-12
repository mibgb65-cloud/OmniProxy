import { act, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { getGatewayInfo } from "../../services/gateway"
import { listAccounts, listProviders } from "../../services/omniproxy"
import {
  createRoute,
  listDefaultRoutes,
  listRoutes,
  setModelPreferredProvider,
  type DefaultModelRoute,
} from "../../services/routing"
import { useUIStore } from "../../stores/ui-store"
import { selectOption } from "../../test/select-option"
import {
  accounts,
  defaultMapping,
  providers,
  renderRouting,
  listModelCatalogMock as listModelCatalog,
  syncAccountModelsMock as syncAccountModels,
} from "./routing-test-helpers"

vi.mock("../../services/gateway", () => ({ getGatewayInfo: vi.fn() }))
vi.mock("../../services/omniproxy", () => ({ listProviders: vi.fn(), listAccounts: vi.fn() }))
vi.mock("../../services/routing", () => ({
  createRoute: vi.fn(),
  updateRoute: vi.fn(),
  setRouteEnabled: vi.fn(),
  deleteRoute: vi.fn(),
  listRoutes: vi.fn(),
  listDefaultRoutes: vi.fn(),
  setModelPreferredProvider: vi.fn(),
  previewRoute: vi.fn(),
}))

let mappings: DefaultModelRoute[]
const selector = () => screen.getByRole("combobox", { name: "deepseek-flash 的优先服务商" })

describe("默认模型映射与服务商优先选择", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useUIStore.setState({ language: "zh" })
    mappings = [structuredClone(defaultMapping)]
    vi.mocked(listDefaultRoutes).mockImplementation(async () => mappings)
    vi.mocked(listRoutes).mockResolvedValue([])
    vi.mocked(listModelCatalog).mockResolvedValue([])
    vi.mocked(listProviders).mockResolvedValue(providers)
    vi.mocked(listAccounts).mockResolvedValue(
      accounts.map((a) => ({ ...a, authMethod: "api_key" })),
    )
    vi.mocked(getGatewayInfo).mockResolvedValue({
      runtime: { state: "stopped", address: "", port: 3001, startedAt: "", errorCode: "" },
      models: ["deepseek-flash"],
      protocol: "Chat Completions",
      requestTimeoutSeconds: 120,
    })
    vi.mocked(setModelPreferredProvider).mockImplementation(async (model, providerId) => {
      const preferred = providerId !== ""
      mappings = mappings.map((item) =>
        item.model !== model
          ? item
          : {
              ...item,
              preferredProviderId: providerId,
              preferredProviderName: preferred ? "备用 DeepSeek" : "",
              preview: {
                ...item.preview,
                providerId: preferred ? "backup_ds" : "builtin_deepseek",
                providerName: preferred ? "备用 DeepSeek" : "DeepSeek",
                preferredProviderId: providerId,
                preferenceApplied: preferred,
              },
            },
      )
    })
  })

  it("自动展示目录映射，不自动创建规则；选择和恢复自动选择都会保存", async () => {
    const user = userEvent.setup()
    renderRouting()
    expect(await screen.findByText("当前使用：DeepSeek · deepseek-flash")).toBeInTheDocument()
    expect(createRoute).not.toHaveBeenCalled()
    expect(selector()).toHaveTextContent("自动选择服务商")
    await selectOption(user, selector(), "备用 DeepSeek")
    await waitFor(() =>
      expect(setModelPreferredProvider).toHaveBeenCalledWith("deepseek-flash", "backup_ds"),
    )
    expect(await screen.findByText("当前使用：备用 DeepSeek · deepseek-flash")).toBeInTheDocument()
    await waitFor(() => expect(selector()).toBeEnabled())
    await selectOption(user, selector(), "自动选择服务商")
    await waitFor(() =>
      expect(setModelPreferredProvider).toHaveBeenCalledWith("deepseek-flash", ""),
    )
    expect(await screen.findByText("当前使用：DeepSeek · deepseek-flash")).toBeInTheDocument()
  })

  it("保存失败保留原值并隐藏原始错误", async () => {
    vi.mocked(setModelPreferredProvider).mockRejectedValueOnce(new Error("secret token"))
    const user = userEvent.setup()
    renderRouting()
    await screen.findByText("当前使用：DeepSeek · deepseek-flash")
    await selectOption(user, selector(), "备用 DeepSeek")
    expect(await screen.findByRole("alert")).toHaveTextContent("优先服务商保存失败")
    expect(selector()).toHaveTextContent("自动选择服务商")
    expect(screen.queryByText(/secret token/)).not.toBeInTheDocument()
  })

  it("失效偏好保留显示并说明回落，用户可以恢复自动选择", async () => {
    mappings = [
      {
        ...defaultMapping,
        providers: defaultMapping.providers!.slice(0, 1),
        preferredProviderId: "backup_ds",
        preferredProviderName: "备用 DeepSeek",
        preview: {
          ...defaultMapping.preview,
          preferredProviderId: "backup_ds",
          preferenceApplied: false,
        },
      },
    ]
    const user = userEvent.setup()
    renderRouting()
    expect(await screen.findByText("优先服务商当前不可用，已使用 DeepSeek。")).toBeInTheDocument()
    expect(selector()).toHaveTextContent("当前不可用")
    await selectOption(user, selector(), "自动选择服务商")
    await waitFor(() =>
      expect(setModelPreferredProvider).toHaveBeenCalledWith("deepseek-flash", ""),
    )
  })

  it("自定义规则覆盖时显示原因并禁用偏好选择", async () => {
    mappings = [
      {
        ...defaultMapping,
        preview: {
          ...defaultMapping.preview,
          matched: true,
          ruleId: "route_one",
          model: "deepseek-v4-pro",
        },
      },
    ]
    renderRouting()
    expect(await screen.findByText("自定义规则覆盖")).toBeInTheDocument()
    expect(selector()).toBeDisabled()
    expect(screen.getByText("此模型由已启用的自定义规则控制。")).toBeInTheDocument()
  })

  it("官方目录同步后自动出现新映射，支持搜索与英文", async () => {
    vi.mocked(syncAccountModels).mockResolvedValue([
      {
        accountId: "acct_one",
        providerId: "builtin_deepseek",
        model: "deepseek-synced",
        name: "Synced Model",
        fetchedAt: "now",
      },
    ])
    const user = userEvent.setup()
    renderRouting()
    await screen.findByText("当前使用：DeepSeek · deepseek-flash")
    await user.click(screen.getByRole("tab", { name: "模型目录" }))
    await selectOption(
      user,
      screen.getByRole("combobox", { name: "同步账号" }),
      "My account · DeepSeek",
    )
    mappings = [...mappings, { ...defaultMapping, model: "deepseek-synced", name: "Synced Model" }]
    expect(
      screen.queryByRole("combobox", { name: "deepseek-synced 的优先服务商" }),
    ).not.toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "同步官方模型" }))
    await user.click(screen.getByRole("tab", { name: "默认映射" }))
    expect(
      await screen.findByRole("combobox", { name: "deepseek-synced 的优先服务商" }),
    ).toBeInTheDocument()
    await user.type(screen.getByRole("textbox", { name: "搜索默认模型映射" }), "synced")
    expect(
      screen.queryByRole("combobox", { name: "deepseek-flash 的优先服务商" }),
    ).not.toBeInTheDocument()
    act(() => useUIStore.setState({ language: "en" }))
    expect(
      await screen.findByRole("heading", { name: "Default model mappings" }),
    ).toBeInTheDocument()
  })
})
