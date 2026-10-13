import { screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { getGatewayInfo } from "../../services/gateway"
import { listAccounts, listProviders } from "../../services/omniproxy"
import { listDefaultRoutes, listRoutes, setModelPreferredProvider } from "../../services/routing"
import { useUIStore } from "../../stores/ui-store"
import { selectOption } from "../../test/select-option"
import {
  accounts,
  defaultMapping,
  providers,
  renderRouting,
  listModelCatalogMock,
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

beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(listDefaultRoutes).mockResolvedValue([defaultMapping])
  vi.mocked(listRoutes).mockResolvedValue([])
  vi.mocked(listModelCatalogMock).mockResolvedValue([])
  vi.mocked(listProviders).mockResolvedValue(providers)
  vi.mocked(listAccounts).mockResolvedValue(accounts)
  vi.mocked(setModelPreferredProvider).mockResolvedValue()
  vi.mocked(getGatewayInfo).mockResolvedValue({
    runtime: { state: "stopped", address: "", port: 3001, startedAt: "", errorCode: "" },
    models: ["deepseek-flash"],
    protocol: "Chat Completions",
    requestTimeoutSeconds: 120,
  })
})

it("按模型直接展示服务商，键盘切到服务商视图后可保存偏好并保留视图", async () => {
  const user = userEvent.setup()
  renderRouting()
  const available = await screen.findByRole("list", { name: "deepseek-flash 的可用服务商" })
  expect(within(available).getByText("DeepSeek")).toBeInTheDocument()
  expect(within(available).getByText("备用 DeepSeek")).toBeInTheDocument()
  expect(screen.getByRole("button", { name: "按模型" })).toHaveAttribute("aria-pressed", "true")
  screen.getByRole("button", { name: "按服务商" }).focus()
  await user.keyboard("{Enter}")
  const backup = screen.getByRole("region", { name: "备用 DeepSeek" })
  expect(within(backup).getByText("deepseek-flash")).toBeInTheDocument()
  await selectOption(user, within(backup).getByRole("combobox"), "备用 DeepSeek")
  await waitFor(() =>
    expect(setModelPreferredProvider).toHaveBeenCalledWith("deepseek-flash", "backup_ds"),
  )
  await user.click(screen.getByRole("tab", { name: "模型目录" }))
  await user.click(screen.getByRole("tab", { name: "默认映射" }))
  expect(screen.getByRole("button", { name: "按服务商" })).toHaveAttribute("aria-pressed", "true")
})

it("服务商视图按当前分组搜索，切回模型视图保留搜索词", async () => {
  const user = userEvent.setup()
  renderRouting()
  await screen.findByRole("list", { name: "deepseek-flash 的可用服务商" })
  await user.click(screen.getByRole("button", { name: "按服务商" }))
  const search = screen.getByRole("textbox", { name: "搜索默认模型映射" })
  await user.type(search, "备用")
  expect(screen.queryByRole("region", { name: "DeepSeek" })).not.toBeInTheDocument()
  expect(screen.getByRole("region", { name: "备用 DeepSeek" })).toBeInTheDocument()
  await user.click(screen.getByRole("button", { name: "按模型" }))
  expect(search).toHaveValue("备用")
  expect(screen.getByRole("list", { name: "deepseek-flash 的可用服务商" })).toBeInTheDocument()
  await user.clear(search)
  await user.type(search, "missing")
  await user.click(screen.getByRole("button", { name: "按服务商" }))
  expect(screen.getByText("没有符合搜索条件的默认模型。")).toBeInTheDocument()
})

it("服务商视图保留没有候选的模型及失效偏好恢复入口", async () => {
  vi.mocked(listDefaultRoutes).mockResolvedValue([
    {
      ...defaultMapping,
      providers: [],
      preferredProviderId: "deleted",
      preferredProviderName: "已删除配置",
      preview: { ...defaultMapping.preview, routable: false, errorCode: "no_enabled_account" },
    },
  ])
  const user = userEvent.setup()
  renderRouting()
  await screen.findByText("目标提供商没有启用账号")
  await user.click(screen.getByRole("button", { name: "按服务商" }))
  const unavailable = screen.getByRole("region", { name: "暂无可用服务商" })
  expect(within(unavailable).getByText("deepseek-flash")).toBeInTheDocument()
  expect(within(unavailable).getByText("目标提供商没有启用账号")).toBeInTheDocument()
  await selectOption(user, within(unavailable).getByRole("combobox"), "自动选择服务商")
  await waitFor(() => expect(setModelPreferredProvider).toHaveBeenCalledWith("deepseek-flash", ""))
})
