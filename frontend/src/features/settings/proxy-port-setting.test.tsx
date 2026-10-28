import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { ReactNode } from "react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import {
  getProxyPort,
  setProxyPort,
  type ProxyPortState,
  type ProxyPortUpdate,
} from "../../services/omniproxy"
import { useUIStore } from "../../stores/ui-store"
import { ProxyPortSetting } from "./proxy-port-setting"

vi.mock("../../services/omniproxy", () => ({
  getProxyPort: vi.fn(),
  setProxyPort: vi.fn(),
}))

let current: ProxyPortState

function update(overrides: Partial<ProxyPortUpdate> = {}): ProxyPortUpdate {
  return {
    state: { ...current },
    requested: current.port,
    adjusted: false,
    updated: [],
    manual: [],
    ...overrides,
  }
}

function renderSetting() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <I18nProvider>
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      </I18nProvider>
    )
  }
  return render(<ProxyPortSetting />, { wrapper: Wrapper })
}

describe("ProxyPortSetting", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    useUIStore.setState({ language: "zh" })
    current = { port: 3000, defaultPort: 3000, available: true, running: true, envLocked: false }
    vi.mocked(getProxyPort).mockImplementation(async () => ({ ...current }))
    vi.mocked(setProxyPort).mockImplementation(async (port) => {
      current = { ...current, port }
      return update({ requested: port })
    })
  })

  it("shows the effective and default ports", async () => {
    renderSetting()

    expect(await screen.findByText("当前端口：3000")).toBeInTheDocument()
    expect(screen.getByText("默认端口：3000")).toBeInTheDocument()
    expect(screen.getByText("端口可用")).toBeInTheDocument()
    expect(screen.getByText("网关正在监听")).toBeInTheDocument()
  })

  it("saves a new port through the backend", async () => {
    const user = userEvent.setup()
    renderSetting()

    const input = await screen.findByLabelText("端口")
    await user.clear(input)
    await user.type(input, "4321")
    await user.click(screen.getByRole("button", { name: "保存端口" }))

    await waitFor(() => expect(setProxyPort).toHaveBeenCalledWith(4321))
    expect(await screen.findByText("端口已更新为 4321。")).toBeInTheDocument()
  })

  it("restores the default port with zero", async () => {
    const user = userEvent.setup()
    renderSetting()

    await user.click(await screen.findByRole("button", { name: "恢复默认端口" }))

    await waitFor(() => expect(setProxyPort).toHaveBeenCalledWith(0))
  })

  it("explains automatic avoidance when the requested port was taken", async () => {
    const user = userEvent.setup()
    vi.mocked(setProxyPort).mockImplementation(async () => {
      current = { ...current, port: 3001 }
      return update({ requested: 3000, adjusted: true, updated: ["codex", "claude"] })
    })
    renderSetting()

    await user.click(await screen.findByRole("button", { name: "恢复默认端口" }))

    expect(await screen.findByText("端口 3000 已被占用，已自动改用 3001。")).toBeInTheDocument()
    expect(screen.getByText(/已同步更新受管客户端地址：codex \/ claude。/)).toBeInTheDocument()
  })

  it("lists clients that need to be reconfigured manually", async () => {
    const user = userEvent.setup()
    vi.mocked(setProxyPort).mockResolvedValue(update({ manual: ["codex"] }))
    renderSetting()

    await user.click(await screen.findByRole("button", { name: "恢复默认端口" }))

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "以下客户端的配置被外部改动过，未自动改写地址，请重新读取并一键配置：codex",
    )
  })

  it("rejects out of range input before calling the backend", async () => {
    const user = userEvent.setup()
    renderSetting()

    const input = await screen.findByLabelText("端口")
    await user.clear(input)
    await user.type(input, "70000")

    expect(screen.getByText("请输入 1–65535 之间的端口号。")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "保存端口" })).toBeDisabled()
    expect(setProxyPort).not.toHaveBeenCalled()
  })

  it("is read only when the environment variable pins the port", async () => {
    current = { ...current, envLocked: true }
    renderSetting()

    expect(await screen.findByText(/端口由环境变量 OMNIPROXY_PROXY_PORT 固定/)).toBeInTheDocument()
    expect(screen.queryByLabelText("端口")).not.toBeInTheDocument()
  })

  it("surfaces backend failures", async () => {
    const user = userEvent.setup()
    vi.mocked(setProxyPort).mockRejectedValue(new Error("gateway: port is fixed"))
    renderSetting()

    await user.click(await screen.findByRole("button", { name: "恢复默认端口" }))

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "更改端口失败：gateway: port is fixed",
    )
  })
})
