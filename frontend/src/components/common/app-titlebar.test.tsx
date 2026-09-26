import { act, render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { ThemeProvider } from "../../app/theme-provider"
import { I18nProvider } from "../../i18n/provider"
import { defaultLanguage } from "../../i18n/translation"
import { defaultTheme } from "../../lib/theme"
import { useUIStore } from "../../stores/ui-store"
import { AppTitlebar } from "./app-titlebar"

const runtime = vi.hoisted(() => ({
  handlers: new Map<string, () => void>(),
  minimise: vi.fn(async () => {}),
  toggleMaximise: vi.fn(async () => {}),
  isMaximised: vi.fn(async () => false),
  close: vi.fn(async () => {}),
}))

vi.mock("@wailsio/runtime", () => ({
  Window: {
    Minimise: runtime.minimise,
    ToggleMaximise: runtime.toggleMaximise,
    IsMaximised: runtime.isMaximised,
    Close: runtime.close,
  },
  Events: {
    Types: {
      Common: {
        WindowMaximise: "common:WindowMaximise",
        WindowUnMaximise: "common:WindowUnMaximise",
      },
    },
    On: (name: string, callback: () => void) => {
      runtime.handlers.set(name, callback)
      return () => runtime.handlers.delete(name)
    },
  },
}))

function renderTitlebar() {
  return render(
    <I18nProvider>
      <ThemeProvider>
        <AppTitlebar />
      </ThemeProvider>
    </I18nProvider>,
  )
}

describe("AppTitlebar", () => {
  beforeEach(() => {
    localStorage.clear()
    vi.clearAllMocks()
    runtime.handlers.clear()
    runtime.isMaximised.mockResolvedValue(false)
    useUIStore.setState({ language: defaultLanguage, theme: defaultTheme })
  })

  it("switches the full document theme and remembers the choice", async () => {
    const user = userEvent.setup()
    renderTitlebar()

    expect(document.documentElement).toHaveClass("dark")
    await user.click(screen.getByRole("button", { name: "切换到亮色模式" }))

    expect(document.documentElement).not.toHaveClass("dark")
    expect(useUIStore.getState().theme).toBe("light")
    expect(JSON.parse(localStorage.getItem("omniproxy-ui-state")!).state.theme).toBe("light")

    await user.click(screen.getByRole("button", { name: "切换到暗色模式" }))
    expect(document.documentElement).toHaveClass("dark")
  })

  it("calls the Wails window controls and follows native maximise events", async () => {
    const user = userEvent.setup()
    renderTitlebar()

    await user.click(screen.getByRole("button", { name: "最小化" }))
    expect(runtime.minimise).toHaveBeenCalledOnce()

    await user.click(screen.getByRole("button", { name: "最大化" }))
    expect(runtime.toggleMaximise).toHaveBeenCalledOnce()

    act(() => runtime.handlers.get("common:WindowMaximise")?.())
    expect(screen.getByRole("button", { name: "还原窗口" })).toBeInTheDocument()

    act(() => runtime.handlers.get("common:WindowUnMaximise")?.())
    expect(screen.getByRole("button", { name: "最大化" })).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "关闭窗口" }))
    expect(runtime.close).toHaveBeenCalledOnce()
  })

  it("shows a visible error if a window action fails", async () => {
    const user = userEvent.setup()
    runtime.minimise.mockRejectedValueOnce(new Error("failed"))
    renderTitlebar()

    await user.click(screen.getByRole("button", { name: "最小化" }))
    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("窗口操作失败"))
  })
})
