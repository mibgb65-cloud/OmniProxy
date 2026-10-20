import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import type { AccountItem } from "../../services/omniproxy"
import { logoutSubscription, refreshSubscription } from "../../services/subscription"
import { useUIStore } from "../../stores/ui-store"
import { SubscriptionAccountStatus } from "./subscription-account-status"

vi.mock("../../services/subscription", async (original) => ({
  ...(await original<typeof import("../../services/subscription")>()),
  refreshSubscription: vi.fn(),
  logoutSubscription: vi.fn(),
}))
vi.mock("./subscription-login-dialog", () => ({ SubscriptionLoginDialog: () => null }))
vi.mock("./reset-credits-popover", () => ({ ResetCreditsPopover: () => null }))
vi.mock("@wailsio/runtime", () => ({}))

const account: AccountItem = {
  id: "subscription-test",
  providerId: "builtin_openai",
  name: "Test",
  enabled: true,
  createdAt: "2026-10-20T00:00:00Z",
  updatedAt: "2026-10-20T00:00:00Z",
  authMethod: "subscription_login",
  authState: "authenticated",
  authPlan: "plus",
  authEmail: "example@example.test",
  authAccountId: "workspace-test",
}

function mount() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const view = (item: AccountItem) => (
    <I18nProvider>
      <QueryClientProvider client={client}>
        <SubscriptionAccountStatus account={item} />
      </QueryClientProvider>
    </I18nProvider>
  )
  const result = render(view(account))
  return { ...result, update: (item: AccountItem) => result.rerender(view(item)) }
}

beforeEach(() => {
  vi.resetAllMocks()
  useUIStore.setState({ language: "zh" })
})
afterEach(() => vi.useRealTimers())

it.each([
  ["operation_cancelled", "本次刷新已停止，可稍后重试。"],
  ["operation_timeout", "账号状态刷新超时，已保留原状态，可稍后重试。"],
  ["refresh_busy", "其他账号操作尚未完成，请稍后刷新。"],
  ["unknown-private-detail", "暂时无法刷新账号状态，已保留原状态，请稍后重试。"],
])("%s 使用非阻断提示，不再误报授权取消或持续卡片红字", async (code, message) => {
  vi.mocked(refreshSubscription).mockRejectedValue(new Error(`subscription: ${code}`))
  const user = userEvent.setup()
  mount()
  await user.click(screen.getByRole("button", { name: "刷新状态" }))
  expect(await screen.findByText(message)).toBeInTheDocument()
  expect(screen.queryByRole("alert")).not.toBeInTheDocument()
  expect(screen.queryByText("授权操作已取消，请重新开始。")).not.toBeInTheDocument()
  expect(screen.queryByText(/unknown-private-detail/)).not.toBeInTheDocument()
  await user.click(screen.getByRole("button", { name: "关闭刷新提示" }))
  expect(screen.queryByText(message)).not.toBeInTheDocument()
})

it("临时刷新提示五秒后自动消失", async () => {
  vi.useFakeTimers()
  vi.mocked(refreshSubscription).mockRejectedValue(new Error("subscription: refresh_timeout"))
  mount()
  fireEvent.click(screen.getByRole("button", { name: "刷新状态" }))
  await act(async () => {
    await vi.advanceTimersByTimeAsync(100)
  })
  expect(screen.getByText("账号状态刷新超时，已保留原状态，可稍后重试。")).toBeInTheDocument()
  await act(async () => {
    await vi.advanceTimersByTimeAsync(5100)
  })
  expect(screen.queryByRole("button", { name: "关闭刷新提示" })).not.toBeInTheDocument()
})

it("重试成功清除临时失败提示，且不启动授权", async () => {
  vi.mocked(refreshSubscription)
    .mockRejectedValueOnce(new Error("subscription: refresh_unavailable"))
    .mockResolvedValueOnce(account)
  const user = userEvent.setup()
  mount()
  await user.click(screen.getByRole("button", { name: "刷新状态" }))
  await screen.findByRole("button", { name: "关闭刷新提示" })
  await user.click(screen.getByRole("button", { name: "刷新状态" }))
  await waitFor(() => expect(refreshSubscription).toHaveBeenCalledTimes(2))
  expect(screen.queryByRole("button", { name: "关闭刷新提示" })).not.toBeInTheDocument()
  expect(logoutSubscription).not.toHaveBeenCalled()
})

it("真实身份错误继续内联显示，后续重新登录的账号版本不继承旧错误", async () => {
  vi.mocked(refreshSubscription).mockRejectedValue(new Error("subscription: identity_changed"))
  const user = userEvent.setup()
  const { update } = mount()
  await user.click(screen.getByRole("button", { name: "刷新状态" }))
  expect(await screen.findByRole("alert")).toHaveTextContent("登录身份与原账号不同")
  expect(screen.queryByRole("button", { name: "关闭刷新提示" })).not.toBeInTheDocument()
  update({ ...account, updatedAt: "2026-10-20T00:05:00Z" })
  expect(screen.queryByRole("alert")).not.toBeInTheDocument()
})

it("确切缺失授权继续显示需要重新登录，不伪装成正常状态", () => {
  const { update } = mount()
  update({ ...account, authState: "reauth_required" })
  expect(screen.getByText("需要重新登录")).toBeInTheDocument()
})

it("英文错误反馈也明确区分刷新超时和授权取消", async () => {
  useUIStore.setState({ language: "en" })
  vi.mocked(refreshSubscription).mockRejectedValue(new Error("subscription: refresh_timeout"))
  const user = userEvent.setup()
  mount()
  await user.click(screen.getByRole("button", { name: "Refresh status" }))
  expect(
    await screen.findByText(
      "Account status refresh timed out. The previous state was kept; try again later.",
    ),
  ).toBeInTheDocument()
})
