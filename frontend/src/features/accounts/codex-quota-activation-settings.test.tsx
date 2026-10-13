import { act, render, screen, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { CancellablePromise } from "@wailsio/runtime"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import {
  getCodexSettings,
  getCodexQuotaActivationStatus,
  setCodexQuotaAutoActivation,
} from "../../services/codex-settings"
import { CodexQuotaActivationSettings } from "./codex-quota-activation-settings"

vi.mock("../../services/codex-settings", () => ({
  getCodexSettings: vi.fn(),
  getCodexQuotaActivationStatus: vi.fn(),
  setCodexQuotaAutoActivation: vi.fn(),
}))
beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(getCodexSettings).mockResolvedValue({
    webSocketEnabled: true,
    autoQuotaActivation: false,
  })
  vi.mocked(setCodexQuotaAutoActivation).mockResolvedValue({
    webSocketEnabled: true,
    autoQuotaActivation: true,
  })
  vi.mocked(getCodexQuotaActivationStatus).mockResolvedValue({
    running: false,
    checked: 2,
    sent: 1,
    activated: 1,
    failed: 0,
    pending: 0,
    updatedAt: "now",
    errorCode: "",
  })
})
function renderSettings() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return {
    client,
    ...render(
      <I18nProvider>
        <QueryClientProvider client={client}>
          <CodexQuotaActivationSettings />
        </QueryClientProvider>
      </I18nProvider>,
    ),
  }
}

it("默认关闭，切换开关直接保存并显示检测状态，无需二次确认", async () => {
  const user = userEvent.setup()
  renderSettings()
  const toggle = await screen.findByRole("switch", { name: "自动激活 Codex 额度窗口" })
  await waitFor(() => expect(toggle).toBeEnabled())
  expect(toggle).not.toBeChecked()
  expect(getCodexQuotaActivationStatus).not.toHaveBeenCalled()
  expect(screen.queryByRole("button")).not.toBeInTheDocument()
  toggle.focus()
  await user.keyboard(" ")
  await waitFor(() => expect(vi.mocked(setCodexQuotaAutoActivation).mock.calls[0]?.[0]).toBe(true))
  expect(setCodexQuotaAutoActivation).toHaveBeenCalledTimes(1)
  expect(
    await screen.findByText("最近检查 2 个账号，发送 1 条消息，确认激活 1 个账号。"),
  ).toBeInTheDocument()
  expect(toggle).toBeChecked()
  expect(screen.queryByRole("button")).not.toBeInTheDocument()
})

it("保存失败保留原开关状态，点击可重试且隐藏原始错误", async () => {
  vi.mocked(setCodexQuotaAutoActivation).mockRejectedValueOnce(new Error("private token error"))
  const user = userEvent.setup()
  const { client } = renderSettings()
  const toggle = await screen.findByRole("switch", { name: "自动激活 Codex 额度窗口" })
  await waitFor(() => expect(toggle).toBeEnabled())
  await user.click(toggle)
  expect(await screen.findByRole("alert")).toHaveTextContent("未保存，原设置仍生效")
  expect(client.getQueryData(["codex-settings"])).toEqual({
    webSocketEnabled: true,
    autoQuotaActivation: false,
  })
  expect(getCodexQuotaActivationStatus).not.toHaveBeenCalled()
  expect(toggle).not.toBeChecked()
  expect(screen.queryByText("private token error")).not.toBeInTheDocument()
  await user.click(toggle)
  expect(await screen.findByText("自动激活设置已保存。")).toBeInTheDocument()
})

it("读取已保存授权，明确显示待复查与失败，不宣称未知窗口已激活", async () => {
  vi.mocked(getCodexSettings).mockResolvedValue({
    webSocketEnabled: true,
    autoQuotaActivation: true,
  })
  vi.mocked(getCodexQuotaActivationStatus).mockResolvedValue({
    running: false,
    checked: 1,
    sent: 1,
    activated: 0,
    failed: 1,
    pending: 1,
    updatedAt: "now",
    errorCode: "activation_failed",
  })
  renderSettings()
  expect(
    await screen.findByText("部分窗口尚待官方确认，后续只复查，不重复发送。"),
  ).toBeInTheDocument()
  expect(screen.getByText("部分账号检测或激活未完成，请检查账号授权与网络。")).toBeInTheDocument()
  act(() => useUIStore.setState({ language: "en" }))
  expect(
    await screen.findByRole("heading", { name: "Automatically activate Codex quota windows" }),
  ).toBeInTheDocument()
})

it("关闭开关直接保存关闭并停止状态查询", async () => {
  vi.mocked(getCodexSettings).mockResolvedValue({
    webSocketEnabled: true,
    autoQuotaActivation: true,
  })
  vi.mocked(setCodexQuotaAutoActivation).mockResolvedValue({
    webSocketEnabled: true,
    autoQuotaActivation: false,
  })
  const user = userEvent.setup()
  const { client } = renderSettings()
  const toggle = await screen.findByRole("switch", { name: "自动激活 Codex 额度窗口" })
  await waitFor(() => expect(toggle).toBeEnabled())
  await user.click(toggle)
  await waitFor(() =>
    expect(client.getQueryData(["codex-settings"])).toEqual({
      webSocketEnabled: true,
      autoQuotaActivation: false,
    }),
  )
  expect(setCodexQuotaAutoActivation).toHaveBeenCalledWith(false, expect.anything())
  expect(toggle).not.toBeChecked()
})

it("保存期间显示进度，拒绝重复切换且保留已确认值", async () => {
  let finish!: () => void
  vi.mocked(setCodexQuotaAutoActivation).mockImplementationOnce(
    () =>
      new CancellablePromise((resolve) => {
        finish = () => resolve({ webSocketEnabled: true, autoQuotaActivation: true })
      }),
  )
  const user = userEvent.setup()
  renderSettings()
  const toggle = await screen.findByRole("switch", { name: "自动激活 Codex 额度窗口" })
  await waitFor(() => expect(toggle).toBeEnabled())
  await user.click(toggle)
  expect(toggle).toBeDisabled()
  expect(toggle).not.toBeChecked()
  expect(await screen.findByText("正在保存…")).toBeInTheDocument()
  await user.click(toggle)
  expect(setCodexQuotaAutoActivation).toHaveBeenCalledTimes(1)
  finish()
  await waitFor(() => expect(toggle).toBeChecked())
  await waitFor(() => expect(toggle).toBeEnabled())
})
