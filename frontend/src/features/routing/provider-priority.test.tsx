import { act, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { getGatewayInfo } from "../../services/gateway"
import { listAccounts, listProviders } from "../../services/omniproxy"
import {
  listDefaultRoutes,
  listRoutes,
  previewRoute,
  setModelProviderOrder,
  type DefaultModelRoute,
} from "../../services/routing"
import { useUIStore } from "../../stores/ui-store"
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
  setModelProviderOrder: vi.fn(),
  previewRoute: vi.fn(),
}))
let mappings: DefaultModelRoute[]
beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  mappings = [structuredClone(defaultMapping)]
  vi.mocked(listDefaultRoutes).mockImplementation(async () => mappings)
  vi.mocked(listRoutes).mockResolvedValue([])
  vi.mocked(listModelCatalogMock).mockResolvedValue([])
  vi.mocked(listProviders).mockResolvedValue(providers)
  vi.mocked(listAccounts).mockResolvedValue(accounts)
  vi.mocked(getGatewayInfo).mockResolvedValue({
    runtime: { state: "stopped", address: "", port: 3001, startedAt: "", errorCode: "" },
    models: ["deepseek-flash"],
    protocol: "Chat Completions",
    requestTimeoutSeconds: 120,
  })
  vi.mocked(setModelProviderOrder).mockImplementation(async (_model, ids) => {
    const item = mappings[0]
    const order = ids.map((id) => ({
      id,
      name: item.providers!.find((provider) => provider.id === id)?.name ?? id,
      available: true,
    }))
    mappings = [
      {
        ...item,
        providerOrder: order,
        preferredProviderId: ids[0] ?? "",
        preferredProviderName: order[0]?.name ?? "",
      },
    ]
  })
})

it("通过键盘调整服务商顺序，保存后两种视图均显示顺序，重新打开配置不会丢失", async () => {
  const user = userEvent.setup()
  renderRouting()
  const configure = await screen.findByRole("button", {
    name: /配置 deepseek-flash 的服务商优先级/,
  })
  await user.click(configure)
  const dialog = screen.getByRole("dialog")
  const up = within(dialog).getByRole("button", { name: "提高 备用 DeepSeek 的优先级" })
  up.focus()
  await user.keyboard("{Enter}")
  const entries = within(dialog).getAllByRole("listitem")
  expect(entries[0]).toHaveTextContent("备用 DeepSeek")
  expect(entries[1]).toHaveTextContent("DeepSeek")
  await user.click(within(dialog).getByRole("button", { name: "保存优先级" }))
  await waitFor(() =>
    expect(setModelProviderOrder).toHaveBeenCalledWith("deepseek-flash", [
      "backup_ds",
      "builtin_deepseek",
    ]),
  )
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
  expect(configure).toHaveFocus()
  const saved = screen.getByRole("list", { name: "deepseek-flash 已保存的服务商顺序" })
  expect(within(saved).getAllByRole("listitem")[0]).toHaveTextContent("1. 备用 DeepSeek")
  await user.click(screen.getByRole("button", { name: "按服务商" }))
  const primary = within(screen.getByRole("region", { name: "DeepSeek" }))
  await user.click(primary.getByRole("button", { name: /配置 deepseek-flash/ }))
  expect(within(screen.getByRole("dialog")).getAllByRole("listitem")[0]).toHaveTextContent(
    "备用 DeepSeek",
  )
  await user.click(screen.getByRole("button", { name: "恢复自动选择" }))
  await waitFor(() => expect(setModelProviderOrder).toHaveBeenLastCalledWith("deepseek-flash", []))
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
  expect(
    primary.queryByRole("list", { name: "deepseek-flash 已保存的服务商顺序" }),
  ).not.toBeInTheDocument()
})

it("保存失败保留已有配置与弹窗草稿，不显示原始错误；取消返回焦点", async () => {
  vi.mocked(setModelProviderOrder).mockRejectedValue(new Error("secret token"))
  const user = userEvent.setup()
  renderRouting()
  const configure = await screen.findByRole("button", {
    name: /配置 deepseek-flash 的服务商优先级/,
  })
  await user.click(configure)
  await user.click(screen.getByRole("button", { name: "提高 备用 DeepSeek 的优先级" }))
  await user.click(screen.getByRole("button", { name: "保存优先级" }))
  expect(await screen.findByRole("alert")).toHaveTextContent("已有配置未改变")
  expect(screen.queryByText("secret token")).not.toBeInTheDocument()
  expect(mappings[0].providerOrder).toEqual([])
  expect(within(screen.getByRole("dialog")).getAllByRole("listitem")[0]).toHaveTextContent(
    "备用 DeepSeek",
  )
  await user.click(screen.getByRole("button", { name: "取消" }))
  await waitFor(() => expect(configure).toHaveFocus())
})

it("自定义规则控制模型时禁用优先级入口", async () => {
  mappings[0].preview.matched = true
  renderRouting()
  expect(
    await screen.findByRole("button", { name: /配置 deepseek-flash 的服务商优先级/ }),
  ).toBeDisabled()
})

it("预览明确显示第几优先服务商，中英文一致", async () => {
  vi.mocked(previewRoute).mockResolvedValue({
    ...defaultMapping.preview,
    providerPriority: 2,
    preferredProviderId: "backup_ds",
    preferenceApplied: false,
  })
  const user = userEvent.setup()
  renderRouting("preview")
  await user.type(await screen.findByRole("textbox", { name: "客户端模型名" }), "deepseek-flash")
  await user.click(screen.getByRole("button", { name: "预览匹配" }))
  expect(await screen.findByText("使用第 2 优先服务商")).toBeInTheDocument()
  act(() => useUIStore.setState({ language: "en" }))
  expect(await screen.findByText("Using provider priority 2")).toBeInTheDocument()
})
