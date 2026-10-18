import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { getFailoverSettings, getFailoverHealth, setFailoverEnabled } from "../../services/failover"
import { GatewayFailoverCard } from "./gateway-failover-card"

vi.mock("../../services/failover", () => ({
  getFailoverSettings: vi.fn(),
  getFailoverHealth: vi.fn(),
  setFailoverEnabled: vi.fn(),
}))

function mount() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <GatewayFailoverCard active running={false} />
      </QueryClientProvider>
    </I18nProvider>,
  )
}
beforeEach(() => {
  vi.resetAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(getFailoverSettings).mockResolvedValue({ enabled: true, maxAttempts: 3 })
  vi.mocked(getFailoverHealth).mockResolvedValue([])
})

it("开关等实际保存后更新，保存失败保留原值并支持重试", async () => {
  const user = userEvent.setup()
  let finish!: (value: { enabled: boolean; maxAttempts: number }) => void
  vi.mocked(setFailoverEnabled).mockImplementationOnce(
    () =>
      new Promise((resolve) => {
        finish = resolve
      }),
  )
  mount()
  const toggle = await screen.findByRole("switch", { name: "账号故障切换" })
  await waitFor(() => expect(toggle).toBeEnabled())
  expect(toggle).toBeChecked()
  await user.click(toggle)
  expect(setFailoverEnabled).toHaveBeenCalledWith(false, expect.anything())
  expect(toggle).toBeDisabled()
  finish({ enabled: false, maxAttempts: 3 })
  await waitFor(() => expect(toggle).not.toBeChecked())
  vi.mocked(setFailoverEnabled).mockRejectedValueOnce(new Error("private upstream detail"))
  await user.click(toggle)
  expect(await screen.findByRole("alert")).toHaveTextContent("读取或保存失败，请重试。")
  expect(toggle).not.toBeChecked()
  expect(screen.queryByText("private upstream detail")).not.toBeInTheDocument()
  await user.click(screen.getByRole("button", { name: "重试读取切换设置" }))
  await waitFor(() => expect(toggle).toBeChecked())
})

it("显示冷却原因和恢复时间，不把它呈现成账号停用", async () => {
  vi.mocked(getFailoverHealth).mockResolvedValue([
    {
      accountId: "a",
      accountName: "测试账号",
      model: "deepseek-flash",
      code: "upstream_rate_limited",
      retryAt: "2026-10-18T09:00:00Z",
      recovering: false,
    },
  ])
  mount()
  expect(await screen.findByText("1 项账号资源暂时不可用")).toBeInTheDocument()
  expect(screen.getByText("测试账号 · deepseek-flash")).toBeInTheDocument()
  expect(screen.getByText(/可恢复时间：/)).toBeInTheDocument()
  expect(screen.getByText(/最多尝试 3 个账号/)).toBeInTheDocument()
})
