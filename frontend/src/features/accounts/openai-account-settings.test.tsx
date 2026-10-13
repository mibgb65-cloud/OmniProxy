import { render, screen, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { OpenAIAccountSettings } from "./openai-account-settings"
import { getCodexSettings, setCodexWebSocketEnabled } from "../../services/codex-settings"

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
  })
  it("opens global gateway settings on demand and restores focus when closed", async () => {
    const user = userEvent.setup()
    renderSettings()
    expect(getCodexSettings).not.toHaveBeenCalled()
    const trigger = screen.getByRole("button", { name: "OpenAI 账号设置" })
    await user.click(trigger)
    const panel = screen.getByRole("dialog", { name: "OpenAI 账号设置" })
    expect(panel).toHaveAttribute("aria-modal", "true")
    expect(await screen.findByRole("switch", { name: "Codex WebSocket" })).toBeChecked()
    expect(panel).toHaveTextContent("OpenAI 账号设置")
    await user.keyboard("{Escape}")
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
    expect(trigger).toHaveFocus()
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
})
