import { selectOption } from "../../test/select-option"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { act, render, screen, within } from "@testing-library/react"
import type { ComponentProps } from "react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { listModelCatalog, syncAccountModels } from "../../services/models"
import { ModelCatalog } from "./model-catalog"
import { useUIStore } from "../../stores/ui-store"
vi.mock("../../services/models", () => ({ listModelCatalog: vi.fn(), syncAccountModels: vi.fn() }))
beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
})

const catalogProps: ComponentProps<typeof ModelCatalog> = {
  accounts: [
    {
      id: "a",
      providerId: "p",
      name: "Account",
      enabled: true,
      createdAt: "",
      updatedAt: "",
      authMethod: "subscription_login",
      authState: "authenticated",
    },
  ],
  providers: [
    {
      id: "p",
      kind: "openai",
      source: "builtin",
      name: "OpenAI",
      baseUrl: "",
      enabled: true,
      createdAt: "",
      updatedAt: "",
    },
  ],
}

function renderCatalog(props = catalogProps) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <ModelCatalog {...props} />
      </QueryClientProvider>
    </I18nProvider>,
  )
}

it("Atria 静态目录显示来源和文本能力，不提供未经确认的账号同步", async () => {
  vi.mocked(listModelCatalog).mockResolvedValue([
    {
      accountId: "a",
      providerId: "p",
      model: "Atria-Dawn-Preview",
      name: "Atria-Dawn-Preview",
      fetchedAt: "",
      source: "preset",
      metadata: { contextWindow: 256000, inputModalities: ["text"] },
    },
  ])
  renderCatalog({
    providers: [{ ...catalogProps.providers[0], kind: "atria", name: "Atria" }],
    accounts: [{ ...catalogProps.accounts[0], authMethod: "api_key" }],
  })
  expect(await screen.findByText("Atria-Dawn-Preview")).toBeInTheDocument()
  expect(screen.getByText("官方预设 · 账号访问权限待验证")).toBeInTheDocument()
  expect(screen.getByText("上下文 256000 Token")).toBeInTheDocument()
  expect(screen.getByRole("button", { name: "同步官方模型" })).toBeDisabled()
  expect(syncAccountModels).not.toHaveBeenCalled()
})

it("按账号同步并在失败时保留原模型目录", async () => {
  vi.mocked(listModelCatalog).mockResolvedValue([
    { accountId: "a", providerId: "p", model: "gpt-test", name: "GPT Test", fetchedAt: "now" },
  ])
  vi.mocked(syncAccountModels).mockRejectedValue(new Error("secret token"))
  renderCatalog()
  const user = userEvent.setup()
  expect(await screen.findByText("gpt-test")).toBeInTheDocument()
  await selectOption(user, screen.getByRole("combobox"), "Account · OpenAI")
  await user.click(screen.getByRole("button", { name: "同步官方模型" }))
  expect(await screen.findByRole("alert")).toHaveTextContent("保留上次成功")
  expect(screen.getByText("gpt-test")).toBeInTheDocument()
  expect(syncAccountModels).toHaveBeenCalledWith("a")
  expect(screen.queryByText("secret token")).not.toBeInTheDocument()
})

it("MiMo 普通 API Key 可同步官方模型，发现结果替换预设", async () => {
  vi.mocked(listModelCatalog).mockResolvedValue([
    {
      accountId: "a",
      providerId: "p",
      model: "mimo-v2.6-pro",
      name: "mimo-v2.6-pro",
      fetchedAt: "",
      source: "preset",
    },
  ])
  vi.mocked(syncAccountModels).mockResolvedValue([
    {
      accountId: "a",
      providerId: "p",
      model: "mimo-v2.6-flash",
      name: "mimo-v2.6-flash",
      fetchedAt: "now",
    },
  ])
  renderCatalog({
    providers: [{ ...catalogProps.providers[0], kind: "mimo", name: "Xiaomi MiMo" }],
    accounts: [{ ...catalogProps.accounts[0], authMethod: "api_key" }],
  })
  const user = userEvent.setup()
  expect(await screen.findByText("mimo-v2.6-pro")).toBeInTheDocument()
  await selectOption(user, screen.getByRole("combobox"), "Account · Xiaomi MiMo")
  await user.click(screen.getByRole("button", { name: "同步官方模型" }))
  expect(await screen.findByText("mimo-v2.6-flash")).toBeInTheDocument()
  expect(syncAccountModels).toHaveBeenCalledWith("a")
  expect(screen.queryByText("mimo-v2.6-pro")).not.toBeInTheDocument()
})

it("River API Key 可同步官方模型，发现结果替换预设", async () => {
  vi.mocked(listModelCatalog).mockResolvedValue([
    {
      accountId: "a",
      providerId: "p",
      model: "zai-org/GLM-5.3-Flash",
      name: "zai-org/GLM-5.3-Flash",
      fetchedAt: "",
      source: "preset",
    },
  ])
  vi.mocked(syncAccountModels).mockResolvedValue([
    {
      accountId: "a",
      providerId: "p",
      model: "deepseek-ai/DeepSeek-V4.1-Flash",
      name: "deepseek-ai/DeepSeek-V4.1-Flash",
      fetchedAt: "now",
    },
  ])
  renderCatalog({
    providers: [{ ...catalogProps.providers[0], kind: "river", name: "River" }],
    accounts: [{ ...catalogProps.accounts[0], authMethod: "api_key" }],
  })
  const user = userEvent.setup()
  expect(await screen.findByText("zai-org/GLM-5.3-Flash")).toBeInTheDocument()
  await selectOption(user, screen.getByRole("combobox"), "Account · River")
  await user.click(screen.getByRole("button", { name: "同步官方模型" }))
  expect(await screen.findByText("deepseek-ai/DeepSeek-V4.1-Flash")).toBeInTheDocument()
  expect(syncAccountModels).toHaveBeenCalledWith("a")
  expect(screen.queryByText("zai-org/GLM-5.3-Flash")).not.toBeInTheDocument()
})

it("模型与服务商双向分组，同一服务商的账号共用模型标题并保留各自能力", async () => {
  vi.mocked(listModelCatalog).mockResolvedValue([
    {
      accountId: "a",
      providerId: "p",
      model: "shared-model",
      name: "Shared Model",
      fetchedAt: "now",
      metadata: { contextWindow: 128000 },
    },
    { accountId: "b", providerId: "p", model: "shared-model", name: "", fetchedAt: "now" },
    { accountId: "c", providerId: "q", model: "shared-model", name: "", fetchedAt: "now" },
    { accountId: "c", providerId: "q", model: "other-model", name: "", fetchedAt: "now" },
  ])
  const user = userEvent.setup()
  renderCatalog({
    providers: [...catalogProps.providers, { ...catalogProps.providers[0], id: "q", name: "备用" }],
    accounts: [
      ...catalogProps.accounts,
      { ...catalogProps.accounts[0], id: "b", name: "Second account" },
      { ...catalogProps.accounts[0], id: "c", providerId: "q", name: "Backup account" },
    ],
  })
  const shared = await screen.findByRole("region", { name: "shared-model" })
  const openai = within(shared).getByRole("region", { name: "OpenAI" })
  expect(within(openai).getByText("Account")).toBeInTheDocument()
  expect(within(openai).getByText("Second account")).toBeInTheDocument()
  expect(within(openai).getByText("上下文 128000 Token")).toBeInTheDocument()
  expect(within(shared).getByRole("region", { name: "备用" })).toBeInTheDocument()
  expect(screen.queryByText(/按账号同步 ChatGPT \/ Codex 与 DeepSeek/)).not.toBeInTheDocument()
  await user.click(screen.getByRole("button", { name: "按服务商" }))
  const primary = screen.getByRole("region", { name: "OpenAI" })
  expect(within(primary).getAllByRole("heading", { name: "shared-model" })).toHaveLength(1)
  expect(within(primary).queryByRole("heading", { name: "other-model" })).not.toBeInTheDocument()
  const backup = screen.getByRole("region", { name: "备用" })
  expect(within(backup).getByRole("heading", { name: "shared-model" })).toBeInTheDocument()
  expect(within(backup).getByRole("heading", { name: "other-model" })).toBeInTheDocument()
  act(() => useUIStore.setState({ language: "en" }))
  expect(screen.getByRole("button", { name: "By provider" })).toHaveAttribute(
    "aria-pressed",
    "true",
  )
  expect(
    screen.queryByText(/Sync ChatGPT \/ Codex and DeepSeek models per account/),
  ).not.toBeInTheDocument()
})

it("同名服务商按配置 ID 保持独立，删除的服务商和账号使用原 ID 显示", async () => {
  vi.mocked(listModelCatalog).mockResolvedValue([
    { accountId: "a", providerId: "p", model: "first", name: "", fetchedAt: "now" },
    { accountId: "b", providerId: "q", model: "second", name: "", fetchedAt: "now" },
    {
      accountId: "gone-account",
      providerId: "gone-provider",
      model: "third",
      name: "",
      fetchedAt: "now",
    },
  ])
  const user = userEvent.setup()
  renderCatalog({
    ...catalogProps,
    providers: [...catalogProps.providers, { ...catalogProps.providers[0], id: "q" }],
  })
  await screen.findByRole("heading", { name: "first" })
  await user.click(screen.getByRole("button", { name: "按服务商" }))
  const groups = screen.getAllByRole("region", { name: "OpenAI" })
  expect(groups).toHaveLength(2)
  expect(within(groups[0]).getByRole("heading", { name: "first" })).toBeInTheDocument()
  expect(within(groups[1]).getByRole("heading", { name: "second" })).toBeInTheDocument()
  expect(
    within(screen.getByRole("region", { name: "gone-provider" })).getByText("gone-account"),
  ).toBeInTheDocument()
})
