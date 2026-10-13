import { act, render, screen, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { CancellablePromise } from "@wailsio/runtime"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { OpenAIAccountSettings } from "./openai-account-settings"
import {
  getCodexSettings,
  getCodexQuotaActivationStatus,
  setCodexQuotaAutoActivation,
  setCodexWebSocketEnabled,
} from "../../services/codex-settings"

vi.mock("../../services/codex-settings", () => ({
  getCodexSettings: vi.fn(),
  setCodexWebSocketEnabled: vi.fn(),
  setCodexQuotaAutoActivation: vi.fn(),
  getCodexQuotaActivationStatus: vi.fn(),
}))

function renderSettings() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <OpenAIAccountSettings />
        <button>面板外操作</button>
      </QueryClientProvider>
    </I18nProvider>,
  )
}

describe("OpenAI account settings", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useUIStore.setState({ language: "zh" })
    vi.mocked(getCodexSettings).mockResolvedValue({
      webSocketEnabled: true,
      autoQuotaActivation: false,
    })
    vi.mocked(setCodexWebSocketEnabled).mockResolvedValue({
      webSocketEnabled: false,
      autoQuotaActivation: false,
    })
    vi.mocked(getCodexQuotaActivationStatus).mockResolvedValue({
      running: false,
      checked: 0,
      sent: 0,
      activated: 0,
      failed: 0,
      pending: 0,
      updatedAt: "",
      errorCode: "",
    })
  })
  it("opens local gateway settings on demand and restores focus when closed", async () => {
    const user = userEvent.setup()
    renderSettings()
    expect(getCodexSettings).not.toHaveBeenCalled()
    const trigger = screen.getByRole("button", { name: "OpenAI 账号设置" })
    await user.click(trigger)
    const panel = screen.getByRole("dialog", { name: "OpenAI 账号设置" })
    expect(panel).not.toHaveAttribute("aria-modal", "true")
    expect(await screen.findByRole("switch", { name: "Codex WebSocket" })).toBeChecked()
    expect(panel).toHaveTextContent("OpenAI 账号设置")
    await user.keyboard("{Escape}")
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
    expect(trigger).toHaveFocus()
    await user.click(trigger)
    await user.click(trigger)
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    await user.click(trigger)
    const outside = screen.getByRole("button", { name: "面板外操作" })
    await user.click(outside)
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    expect(outside).toHaveFocus()
    await user.click(trigger)
    await user.click(screen.getByRole("button", { name: "关闭账号设置" }))
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
  })
  it("saves the switch and reads its persisted state when reopened", async () => {
    const user = userEvent.setup()
    renderSettings()
    await user.click(screen.getByRole("button", { name: "OpenAI 账号设置" }))
    const toggle = await screen.findByRole("switch", { name: "Codex WebSocket" })
    await waitFor(() => expect(toggle).toBeEnabled())
    toggle.focus()
    await user.keyboard(" ")
    await waitFor(() =>
      expect(setCodexWebSocketEnabled).toHaveBeenCalledWith(false, expect.anything()),
    )
    await waitFor(() => expect(toggle).not.toBeChecked())
    expect(await screen.findByRole("status")).toHaveTextContent("设置已保存并生效")
    await user.keyboard("{Escape}")
    vi.mocked(getCodexSettings).mockResolvedValue({
      webSocketEnabled: false,
      autoQuotaActivation: false,
    })
    await user.click(screen.getByRole("button", { name: "OpenAI 账号设置" }))
    expect(await screen.findByRole("switch", { name: "Codex WebSocket" })).not.toBeChecked()
  })
  it("keeps the previous state on save failure and hides private errors", async () => {
    vi.mocked(setCodexWebSocketEnabled).mockRejectedValue(new Error("secret token"))
    const user = userEvent.setup()
    renderSettings()
    await user.click(screen.getByRole("button", { name: "OpenAI 账号设置" }))
    const toggle = await screen.findByRole("switch", { name: "Codex WebSocket" })
    await waitFor(() => expect(toggle).toBeEnabled())
    await user.click(toggle)
    expect(await screen.findByRole("alert")).toHaveTextContent("原有开关状态已保留")
    expect(toggle).toBeChecked()
    expect(screen.queryByText("secret token")).not.toBeInTheDocument()
  })
  it("disables changes after a read error and provides retry", async () => {
    vi.mocked(getCodexSettings).mockRejectedValueOnce(new Error("private database error"))
    const user = userEvent.setup()
    renderSettings()
    await user.click(screen.getByRole("button", { name: "OpenAI 账号设置" }))
    expect(await screen.findByRole("alert")).toHaveTextContent("无法读取接入设置")
    expect(screen.getByRole("switch", { name: "Codex WebSocket" })).toBeDisabled()
    expect(screen.queryByText("private database error")).not.toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "重新读取" }))
    await waitFor(() =>
      expect(screen.getByRole("switch", { name: "Codex WebSocket" })).toBeChecked(),
    )
    expect(setCodexWebSocketEnabled).not.toHaveBeenCalled()
  })
  it.each(["websocket", "activation"])(
    "locks both switches during %s save across reopen and ignores an old read",
    async (setting) => {
      const activation = setting === "activation"
      const next = { webSocketEnabled: activation, autoQuotaActivation: activation }
      let finishSave!: () => void
      let finishRead!: () => void
      const setter = activation ? setCodexQuotaAutoActivation : setCodexWebSocketEnabled
      vi.mocked(setter).mockImplementationOnce(
        () =>
          new CancellablePromise((resolve) => {
            finishSave = () => resolve(next)
          }),
      )
      const user = userEvent.setup()
      renderSettings()
      const trigger = screen.getByRole("button", { name: "OpenAI 账号设置" })
      await user.click(trigger)
      const name = activation ? "自动激活 Codex 额度窗口" : "Codex WebSocket"
      const toggle = await screen.findByRole("switch", { name })
      await waitFor(() => expect(toggle).toBeEnabled())
      await user.click(toggle)
      expect(screen.getAllByRole("switch")).toHaveLength(2)
      screen.getAllByRole("switch").forEach((item) => expect(item).toBeDisabled())
      await user.click(screen.getByRole("button", { name: "关闭账号设置" }))
      vi.mocked(getCodexSettings).mockImplementationOnce(
        () =>
          new CancellablePromise((resolve) => {
            finishRead = () => resolve({ webSocketEnabled: true, autoQuotaActivation: false })
          }),
      )
      await user.click(trigger)
      screen.getAllByRole("switch").forEach((item) => expect(item).toBeDisabled())
      await user.click(screen.getByRole("switch", { name }))
      expect(setter).toHaveBeenCalledTimes(1)
      finishSave()
      await waitFor(() =>
        screen.getAllByRole("switch").forEach((item) => expect(item).toBeEnabled()),
      )
      await act(async () => finishRead())
      const websocket = screen.getByRole("switch", { name: "Codex WebSocket" })
      if (activation) expect(websocket).toBeChecked()
      else expect(websocket).not.toBeChecked()
      if (activation)
        expect(screen.getByRole("switch", { name: "自动激活 Codex 额度窗口" })).toBeChecked()
      else expect(screen.getByRole("switch", { name: "自动激活 Codex 额度窗口" })).not.toBeChecked()
    },
  )
})
