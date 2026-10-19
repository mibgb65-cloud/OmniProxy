import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { CancellablePromise } from "@wailsio/runtime"
import { act, render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import {
  getCodexSettings,
  getCodexQuotaActivationStatus,
  setCodexAccountRotationMode,
} from "../../services/codex-settings"
import { OpenAIAccountSettings } from "./openai-account-settings"

vi.mock("../../services/codex-settings", () => ({
  getCodexSettings: vi.fn(),
  setCodexWebSocketEnabled: vi.fn(),
  setCodexQuotaAutoActivation: vi.fn(),
  setCodexAccountRotationMode: vi.fn(),
  getCodexQuotaActivationStatus: vi.fn(),
}))
const settings = {
  webSocketEnabled: true,
  autoQuotaActivation: false,
  accountRotationMode: "queue",
}
const status = {
  running: false,
  checked: 2,
  sent: 0,
  activated: 2,
  failed: 0,
  pending: 0,
  updatedAt: "2026-10-20T00:00:00Z",
  errorCode: "",
}

beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(getCodexSettings).mockResolvedValue({ ...settings })
  vi.mocked(setCodexAccountRotationMode).mockResolvedValue({
    ...settings,
    accountRotationMode: "balanced",
  })
  vi.mocked(getCodexQuotaActivationStatus).mockResolvedValue({ ...status })
})

async function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <OpenAIAccountSettings />
      </QueryClientProvider>
    </I18nProvider>,
  )
  const user = userEvent.setup()
  await user.click(screen.getByRole("button", { name: "OpenAI 账号设置" }))
  await waitFor(() => expect(screen.getByRole("radio", { name: "队列轮换" })).toBeEnabled())
  return { user, client }
}

it("默认队列模式，保存平衡模式且重开保持选择，中英文及精简提示正确", async () => {
  const { user, client } = await setup()
  expect(screen.getByRole("radio", { name: "队列轮换" })).toBeChecked()
  expect(screen.queryByText(/此设置对所有 OpenAI 账号共用/)).not.toBeInTheDocument()
  expect(screen.queryByText(/仅检测启用且已授权/)).not.toBeInTheDocument()
  expect(screen.queryByText(/切换开关自动保存并生效/)).not.toBeInTheDocument()
  await user.click(screen.getByRole("radio", { name: "平衡切换" }))
  await waitFor(() =>
    expect(setCodexAccountRotationMode).toHaveBeenCalledWith("balanced", expect.anything()),
  )
  await waitFor(() => expect(screen.getByRole("radio", { name: "平衡切换" })).toBeChecked())
  expect(client.getQueryData(["codex-settings"])).toEqual({
    ...settings,
    accountRotationMode: "balanced",
  })
  await user.click(screen.getByRole("button", { name: "关闭账号设置" }))
  vi.mocked(getCodexSettings).mockResolvedValue({ ...settings, accountRotationMode: "balanced" })
  await user.click(screen.getByRole("button", { name: "OpenAI 账号设置" }))
  expect(await screen.findByRole("radio", { name: "平衡切换" })).toBeChecked()
  act(() => useUIStore.setState({ language: "en" }))
  expect(await screen.findByRole("radio", { name: "Balanced" })).toBeChecked()
})

it("模式保存失败保留旧值，不显示后端原始错误", async () => {
  vi.mocked(setCodexAccountRotationMode).mockRejectedValueOnce(new Error("private-database-detail"))
  const { user } = await setup()
  await user.click(screen.getByRole("radio", { name: "平衡切换" }))
  expect(await screen.findByRole("alert")).toHaveTextContent("原有开关状态已保留")
  expect(screen.getByRole("radio", { name: "队列轮换" })).toBeChecked()
  expect(screen.queryByText(/private-database-detail/)).not.toBeInTheDocument()
})

it("轮换模式保存期间锁住所有设置，重开弹窗仍不能重复提交", async () => {
  let finish!: () => void
  vi.mocked(setCodexAccountRotationMode).mockImplementationOnce(
    () =>
      new CancellablePromise((resolve) => {
        finish = () => resolve({ ...settings, accountRotationMode: "balanced" })
      }),
  )
  const { user } = await setup()
  await user.click(screen.getByRole("radio", { name: "平衡切换" }))
  screen.getAllByRole("switch").forEach((toggle) => expect(toggle).toBeDisabled())
  screen.getAllByRole("radio").forEach((radio) => expect(radio).toBeDisabled())
  await user.click(screen.getByRole("button", { name: "关闭账号设置" }))
  await user.click(screen.getByRole("button", { name: "OpenAI 账号设置" }))
  screen.getAllByRole("switch").forEach((toggle) => expect(toggle).toBeDisabled())
  screen.getAllByRole("radio").forEach((radio) => expect(radio).toBeDisabled())
  expect(setCodexAccountRotationMode).toHaveBeenCalledTimes(1)
  finish()
  await waitFor(() => expect(screen.getByRole("radio", { name: "平衡切换" })).toBeChecked())
})

it("检测摘要使用全局悬浮提示，关闭设置后保留，同一结果不重复弹出", async () => {
  vi.mocked(getCodexSettings).mockResolvedValue({ ...settings, autoQuotaActivation: true })
  const { user, client } = await setup()
  const summary = await screen.findByText(
    "最近检查 2 个账号，发送 0 条消息，确认 2 个账号窗口已开始计时。",
  )
  expect(summary.closest("[data-slot='global-status-toast']")).toBeInTheDocument()
  expect(screen.getByRole("dialog", { name: "OpenAI 账号设置" })).not.toContainElement(summary)
  await user.click(screen.getByRole("button", { name: "关闭账号设置" }))
  expect(summary).toBeInTheDocument()
  await user.click(screen.getByRole("button", { name: "关闭额度检测提示" }))
  await user.click(screen.getByRole("button", { name: "OpenAI 账号设置" }))
  await client.invalidateQueries({ queryKey: ["codex-quota-activation"] })
  expect(screen.queryByRole("button", { name: "关闭额度检测提示" })).not.toBeInTheDocument()
  vi.mocked(getCodexQuotaActivationStatus).mockResolvedValue({
    ...status,
    updatedAt: "2026-10-20T00:05:00Z",
    activated: 1,
  })
  await client.invalidateQueries({ queryKey: ["codex-quota-activation"] })
  expect(
    await screen.findByText("最近检查 2 个账号，发送 0 条消息，确认 1 个账号窗口已开始计时。"),
  ).toBeInTheDocument()
})
