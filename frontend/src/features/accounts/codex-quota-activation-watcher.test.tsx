import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { Events } from "@wailsio/runtime"
import { act, render, screen, waitFor } from "@testing-library/react"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { getCodexQuotaActivationStatus, getCodexSettings } from "../../services/codex-settings"
import { CodexQuotaActivationWatcher } from "./codex-quota-activation-watcher"

vi.mock("../../services/codex-settings", () => ({
  getCodexSettings: vi.fn(),
  getCodexQuotaActivationStatus: vi.fn(),
  setCodexQuotaAutoActivation: vi.fn(),
}))
vi.mock("@wailsio/runtime", () => ({ Events: { On: vi.fn(() => vi.fn()) } }))

const enabled = {
  webSocketEnabled: true,
  accountRotationMode: "queue",
  upstreamBaseUrl: "https://chatgpt.com/backend-api/codex",
  autoQuotaActivation: true,
}
const quiet = {
  running: false,
  checked: 0,
  sent: 0,
  activated: 0,
  failed: 0,
  pending: 0,
  updatedAt: "first",
  errorCode: "",
}

beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(getCodexSettings).mockResolvedValue(enabled)
  vi.mocked(getCodexQuotaActivationStatus).mockResolvedValue(quiet)
})

function renderWatcher() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <CodexQuotaActivationWatcher />
      </QueryClientProvider>
    </I18nProvider>,
  )
}

it("空轮次不提示，后台结果推送后立即提示且不必打开设置面板", async () => {
  renderWatcher()
  await waitFor(() => expect(vi.mocked(getCodexQuotaActivationStatus)).toHaveBeenCalled())
  expect(screen.queryByRole("button", { name: "关闭额度检测提示" })).not.toBeInTheDocument()
  const callback = vi
    .mocked(Events.On)
    .mock.calls.find(([name]) => name === "subscription:activation-updated")?.[1]
  expect(callback).toBeTypeOf("function")
  await act(async () => {
    callback?.({
      name: "subscription:activation-updated",
      data: { ...quiet, checked: 2, sent: 1, activated: 1, updatedAt: "second" },
      sender: "",
    })
  })
  expect(
    await screen.findByText("最近检查 2 个账号，发送 1 条消息，确认 1 个账号窗口已开始计时。"),
  ).toBeInTheDocument()
})

it("未开启自动激活时既不查询状态也不提示", async () => {
  vi.mocked(getCodexSettings).mockResolvedValue({ ...enabled, autoQuotaActivation: false })
  renderWatcher()
  await waitFor(() => expect(vi.mocked(getCodexSettings)).toHaveBeenCalled())
  expect(vi.mocked(getCodexQuotaActivationStatus)).not.toHaveBeenCalled()
  expect(screen.queryByRole("button", { name: "关闭额度检测提示" })).not.toBeInTheDocument()
})
