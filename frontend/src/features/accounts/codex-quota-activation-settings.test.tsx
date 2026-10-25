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
import { CodexQuotaActivationWatcher } from "./codex-quota-activation-watcher"

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
    accountRotationMode: "queue",
    upstreamBaseUrl: "https://chatgpt.com/backend-api/codex",
    autoQuotaActivation: false,
  })
  vi.mocked(setCodexQuotaAutoActivation).mockResolvedValue({
    webSocketEnabled: true,
    accountRotationMode: "queue",
    upstreamBaseUrl: "https://chatgpt.com/backend-api/codex",
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
          <ActivationHarness />
        </QueryClientProvider>
      </I18nProvider>,
    ),
  }
}

function ActivationHarness() {
  // 提示条挂在应用级监视器上，不再依赖设置面板是否打开。
  return (
    <>
      <CodexQuotaActivationSettings />
      <CodexQuotaActivationWatcher />
    </>
  )
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
    await screen.findByText("最近检查 2 个账号，发送 1 条消息，确认 1 个账号窗口已开始计时。"),
  ).toBeInTheDocument()
  expect(toggle).toBeChecked()
  expect(screen.getByRole("button", { name: "关闭额度检测提示" })).toBeInTheDocument()
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
    accountRotationMode: "queue",
    upstreamBaseUrl: "https://chatgpt.com/backend-api/codex",
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
    accountRotationMode: "queue",
    upstreamBaseUrl: "https://chatgpt.com/backend-api/codex",
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
    await screen.findByText("部分窗口仍在观测或等待官方确认；已经发送的请求只复查，不自动重发。"),
  ).toBeInTheDocument()
  expect(screen.getAllByText("部分账号检测或激活未完成，请检查账号授权与网络。")).toHaveLength(2)
  act(() => useUIStore.setState({ language: "en" }))
  expect(
    await screen.findByRole("heading", { name: "Automatically activate Codex quota windows" }),
  ).toBeInTheDocument()
})

it("检测失败时直接给出具体原因，不留在笼统提示上", async () => {
  vi.mocked(getCodexSettings).mockResolvedValue({
    webSocketEnabled: true,
    accountRotationMode: "queue",
    upstreamBaseUrl: "https://chatgpt.com/backend-api/codex",
    autoQuotaActivation: true,
  })
  vi.mocked(getCodexQuotaActivationStatus).mockResolvedValue({
    running: false,
    checked: 2,
    sent: 0,
    activated: 1,
    failed: 1,
    pending: 0,
    updatedAt: "now",
    errorCode: "models_unavailable",
  })
  renderSettings()
  const reason = "有账号的官方模型目录为空，激活消息无法选择模型；请先同步该账号模型。"
  expect(await screen.findByText(reason)).toBeInTheDocument()
  // 具体原因同时出现在提示条与设置面板，面板不会只显示检测中。
  await waitFor(() => expect(screen.getAllByText(reason)).toHaveLength(2))
  expect(screen.queryByText("正在检测并复查额度窗口…")).not.toBeInTheDocument()
})

it("关闭开关直接保存关闭并停止状态查询", async () => {
  vi.mocked(getCodexSettings).mockResolvedValue({
    webSocketEnabled: true,
    accountRotationMode: "queue",
    upstreamBaseUrl: "https://chatgpt.com/backend-api/codex",
    autoQuotaActivation: true,
  })
  vi.mocked(setCodexQuotaAutoActivation).mockResolvedValue({
    webSocketEnabled: true,
    accountRotationMode: "queue",
    upstreamBaseUrl: "https://chatgpt.com/backend-api/codex",
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
      accountRotationMode: "queue",
      upstreamBaseUrl: "https://chatgpt.com/backend-api/codex",
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
        finish = () =>
          resolve({
            webSocketEnabled: true,
            accountRotationMode: "queue",
            upstreamBaseUrl: "https://chatgpt.com/backend-api/codex",
            autoQuotaActivation: true,
          })
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

it("首次占位观测显示待确认，不把零发送显示成已激活", async () => {
  vi.mocked(getCodexSettings).mockResolvedValue({
    webSocketEnabled: true,
    accountRotationMode: "queue",
    upstreamBaseUrl: "https://chatgpt.com/backend-api/codex",
    autoQuotaActivation: true,
  })
  vi.mocked(getCodexQuotaActivationStatus).mockResolvedValue({
    running: false,
    checked: 1,
    sent: 0,
    activated: 0,
    failed: 0,
    pending: 1,
    updatedAt: "now",
    errorCode: "",
  })
  renderSettings()
  expect(
    await screen.findByText("最近检查 1 个账号，发送 0 条消息，确认 0 个账号窗口已开始计时。"),
  ).toBeInTheDocument()
  expect(
    screen.getByText("部分窗口仍在观测或等待官方确认；已经发送的请求只复查，不自动重发。"),
  ).toBeInTheDocument()
  expect(
    screen.queryByText("部分账号检测或激活未完成，请检查账号授权与网络。"),
  ).not.toBeInTheDocument()
})
