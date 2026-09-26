import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { ReactNode } from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import {
  getDesktopSettings,
  setAutoStart,
  setCloseBehavior,
  type DesktopSettings,
} from "../../services/omniproxy"
import { useUIStore } from "../../stores/ui-store"
import { DesktopBehavior } from "./desktop-behavior"

vi.mock("../../services/omniproxy", () => ({
  getDesktopSettings: vi.fn(),
  setAutoStart: vi.fn(),
  setCloseBehavior: vi.fn(),
}))

let current: DesktopSettings

function renderBehavior() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <I18nProvider>
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      </I18nProvider>
    )
  }
  return render(<DesktopBehavior />, { wrapper: Wrapper })
}

describe("DesktopBehavior", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    useUIStore.setState({ language: "zh" })
    current = { autoStart: false, autoStartAvailable: true, closeBehavior: "exit" }
    vi.mocked(getDesktopSettings).mockImplementation(async () => ({ ...current }))
    vi.mocked(setAutoStart).mockImplementation(async (enabled) => {
      current.autoStart = enabled
      return enabled
    })
    vi.mocked(setCloseBehavior).mockImplementation(async (mode) => {
      current.closeBehavior = mode
      return mode
    })
  })

  it("shows current OS startup status and defaults to direct exit", async () => {
    renderBehavior()

    expect(await screen.findByRole("switch", { name: /开机自启动/ })).not.toBeChecked()
    expect(screen.getByRole("radio", { name: /直接退出/ })).toBeChecked()
    expect(screen.getByRole("radio", { name: /保留在系统托盘/ })).not.toBeChecked()
  })

  it("explains why startup is unavailable in development", async () => {
    current.autoStartAvailable = false
    renderBehavior()

    expect(await screen.findByRole("switch", { name: /开机自启动/ })).toBeDisabled()
    expect(screen.getByText(/开发环境不会注册 Windows 启动项/)).toBeInTheDocument()
  })

  it("saves autostart and close behavior choices", async () => {
    const user = userEvent.setup()
    renderBehavior()

    await user.click(await screen.findByRole("switch", { name: /开机自启动/ }))
    await waitFor(() => expect(setAutoStart).toHaveBeenCalledWith(true))
    await waitFor(() => expect(screen.getByRole("switch", { name: /开机自启动/ })).toBeChecked())

    await user.click(screen.getByRole("radio", { name: /保留在系统托盘/ }))
    await waitFor(() => expect(setCloseBehavior).toHaveBeenCalledWith("tray"))
    await waitFor(() => expect(screen.getByRole("radio", { name: /保留在系统托盘/ })).toBeChecked())
  })

  it("keeps the original setting visible when registration fails", async () => {
    const user = userEvent.setup()
    vi.mocked(setAutoStart).mockRejectedValue(new Error("registry denied"))
    renderBehavior()

    await user.click(await screen.findByRole("switch", { name: /开机自启动/ }))

    expect(await screen.findByRole("alert")).toHaveTextContent("更改开机启动失败：registry denied")
    expect(screen.getByRole("switch", { name: /开机自启动/ })).not.toBeChecked()
  })
})
