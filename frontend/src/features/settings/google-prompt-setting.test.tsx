import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { ReactNode } from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import type { DesktopSettings } from "../../services/omniproxy"
import { getDesktopSettings, setGooglePromptAsWritten } from "../../services/omniproxy"
import { useUIStore } from "../../stores/ui-store"
import { GooglePromptSetting } from "./google-prompt-setting"

vi.mock("../../services/omniproxy", () => ({
  getDesktopSettings: vi.fn(),
  setGooglePromptAsWritten: vi.fn(),
}))

let current: DesktopSettings

function renderSetting() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <I18nProvider>
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      </I18nProvider>
    )
  }
  return render(<GooglePromptSetting />, { wrapper: Wrapper })
}

describe("GooglePromptSetting", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    useUIStore.setState({ language: "zh" })
    current = {
      autoStart: false,
      autoStartAvailable: false,
      closeBehavior: "exit",
      gatewayAutoStart: false,
      googlePromptAsWritten: false,
    }
    vi.mocked(getDesktopSettings).mockImplementation(async () => ({ ...current }))
    vi.mocked(setGooglePromptAsWritten).mockImplementation(async (asWritten) => {
      current.googlePromptAsWritten = asWritten
      return asWritten
    })
  })

  it("defaults to cloaking the prompt and saves the as-written choice", async () => {
    const user = userEvent.setup()
    renderSetting()

    const toggle = await screen.findByRole("switch", { name: /按原文发送 system prompt/ })
    expect(toggle).not.toBeChecked()

    await user.click(toggle)
    await waitFor(() => expect(setGooglePromptAsWritten).toHaveBeenCalledWith(true))
    await waitFor(() => expect(toggle).toBeChecked())

    await user.click(toggle)
    await waitFor(() => expect(setGooglePromptAsWritten).toHaveBeenLastCalledWith(false))
    await waitFor(() => expect(toggle).not.toBeChecked())
  })

  it("keeps the previous choice and explains a failed save", async () => {
    vi.mocked(setGooglePromptAsWritten).mockRejectedValue(new Error("config unavailable"))
    const user = userEvent.setup()
    renderSetting()

    const toggle = await screen.findByRole("switch", { name: /按原文发送 system prompt/ })
    await user.click(toggle)

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "更改 system prompt 发送方式失败：config unavailable",
    )
    expect(toggle).not.toBeChecked()
  })
})
