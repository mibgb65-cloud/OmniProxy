import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import {
  beginSubscription,
  cancelSubscription,
  openSubscription,
  readSubscription,
  currentSubscription,
  refreshSubscription,
  logoutSubscription,
  readSubscriptionQuota,
  type SubscriptionSession,
} from "../../services/subscription"
import { getAccountKeyPreview, type AccountItem } from "../../services/omniproxy"
import { SubscriptionLoginDialog } from "./subscription-login-dialog"
import { AccountKeyPreview } from "./account-key-preview"
import { AccountDetailsCard } from "./account-details-card"

vi.mock("@wailsio/runtime", () => ({ Events: { On: vi.fn(() => vi.fn()) } }))
vi.mock("../../services/subscription", async (original) => ({
  ...(await original<typeof import("../../services/subscription")>()),
  beginSubscription: vi.fn(),
  readSubscription: vi.fn(),
  currentSubscription: vi.fn(),
  cancelSubscription: vi.fn(),
  openSubscription: vi.fn(),
  refreshSubscription: vi.fn(),
  logoutSubscription: vi.fn(),
  readSubscriptionQuota: vi.fn(),
}))
vi.mock("../../services/omniproxy", () => ({
  getAccountKeyPreview: vi.fn(),
  copyAccountKey: vi.fn(),
}))
const item: AccountItem = {
  id: "acct_subscription",
  providerId: "builtin_openai",
  name: "订阅测试账号",
  enabled: true,
  createdAt: "2026-10-08T00:00:00Z",
  updatedAt: "2026-10-08T00:00:00Z",
  authMethod: "subscription_login",
  authEmail: "example@example.test",
  authPlan: "plus",
  authState: "authenticated",
  authAccountId: "11111111-2222-3333-4444-555555555555",
}
let state: SubscriptionSession
function setup(children = <SubscriptionLoginDialog providerId="builtin_openai" />) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    </I18nProvider>,
  )
  return { user: userEvent.setup(), client }
}
describe("ChatGPT subscription sign-in", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useUIStore.setState({ language: "zh" })
    state = {
      id: "test-session",
      state: "waiting",
      url: "https://auth.openai.com/authorize",
      userCode: "",
      errorCode: "",
      expiresAt: "2026-10-08T01:00:00Z",
      cleanupWarning: false,
    }
    vi.mocked(beginSubscription).mockImplementation(async () => state)
    vi.mocked(currentSubscription).mockImplementation(async () => ({ ...state, id: "", state: "" }))
    vi.mocked(readSubscription).mockImplementation(async () => state)
    vi.mocked(cancelSubscription).mockImplementation(async () => {
      state = { ...state, state: "cancelled", url: "", userCode: "" }
      return state
    })
    vi.mocked(openSubscription).mockResolvedValue(undefined)
    vi.mocked(refreshSubscription).mockResolvedValue(item)
    vi.mocked(logoutSubscription).mockResolvedValue({ ...item, authState: "logged_out" })
    vi.mocked(readSubscriptionQuota).mockResolvedValue({
      plan: "plus",
      fiveHourEnabled: true,
      fiveHour: { remainingPercent: 100 },
      weekly: { remainingPercent: 100 },
      fetchedAt: "2026-10-08T00:00:00Z",
    })
  })
  it("starts browser authorization and refreshes accounts after completion", async () => {
    const { user, client } = setup()
    client.setQueryData(["accounts"], [])
    await user.click(screen.getByRole("button", { name: "登录 ChatGPT" }))
    await user.click(screen.getByRole("button", { name: "开始授权" }))
    expect(await screen.findByText("等待官方页面完成授权")).toBeInTheDocument()
    expect(beginSubscription).toHaveBeenCalledWith({
      providerId: "builtin_openai",
      accountId: undefined,
      name: "",
      mode: "browser",
    })
    expect(openSubscription).toHaveBeenCalledWith("test-session")
    state = { ...state, state: "completed", url: "", account: item }
    await client.invalidateQueries({ queryKey: ["subscription-session"] })
    expect(await screen.findByText("授权完成，账号已保存。")).toBeInTheDocument()
    await waitFor(() => expect(client.getQueryState(["accounts"])?.isInvalidated).toBe(true))
    await user.click(screen.getByRole("button", { name: "完成" }))
    expect(cancelSubscription).not.toHaveBeenCalled()
  })
  it("keeps device authorization alive when closed and restores it on reopen", async () => {
    const { user } = setup()
    await user.click(screen.getByRole("button", { name: "登录 ChatGPT" }))
    await user.click(screen.getByRole("radio", { name: "设备码授权" }))
    state = { ...state, userCode: "TEST-CODE" }
    await user.click(screen.getByRole("button", { name: "开始授权" }))
    expect(await screen.findByText("TEST-CODE")).toBeInTheDocument()
    expect(beginSubscription).toHaveBeenCalledWith(expect.objectContaining({ mode: "device" }))
    await user.click(screen.getByRole("button", { name: "关闭订阅登录窗口" }))
    expect(cancelSubscription).not.toHaveBeenCalled()
    expect(screen.queryByText("TEST-CODE")).not.toBeInTheDocument()
    vi.mocked(currentSubscription).mockImplementation(async () => state)
    await user.click(screen.getByRole("button", { name: "登录 ChatGPT" }))
    expect(await screen.findByText("TEST-CODE")).toBeInTheDocument()
    expect(beginSubscription).toHaveBeenCalledTimes(1)
    await user.click(screen.getByRole("button", { name: "取消" }))
    await waitFor(() => expect(cancelSubscription).toHaveBeenCalledWith("test-session"))
    expect(await screen.findByText("授权已取消。")).toBeInTheDocument()
  })
  it("shows safe errors without exposing raw authorization messages", async () => {
    vi.mocked(beginSubscription).mockRejectedValue(new Error("private-authorization-value"))
    const { user } = setup()
    await user.click(screen.getByRole("button", { name: "登录 ChatGPT" }))
    await user.click(screen.getByRole("button", { name: "开始授权" }))
    expect(await screen.findByRole("alert")).toHaveTextContent("操作未完成")
    expect(screen.queryByText(/private-authorization-value/)).not.toBeInTheDocument()
  })
  it("renders subscription identity and lifecycle actions without reading an API key", async () => {
    const { user } = setup(
      <AccountDetailsCard
        account={item}
        busy={false}
        onToggle={vi.fn()}
        onDelete={vi.fn()}
        onEdit={vi.fn()}
      />,
    )
    expect(screen.getByRole("heading", { name: item.authEmail })).toBeInTheDocument()
    expect(screen.getAllByText("example@example.test")).toHaveLength(1)
    expect(screen.queryByText("已授权")).not.toBeInTheDocument()
    expect(screen.getByText("plus")).toBeInTheDocument()
    expect(screen.getByText("account_id: 11111111…555555")).toBeInTheDocument()
    for (const label of ["刷新状态", "重新登录", "退出登录"]) {
      const button = screen.getByRole("button", { name: label })
      expect(button.textContent).toBe("")
      expect(button).not.toHaveAttribute("title")
    }
    expect(getAccountKeyPreview).not.toHaveBeenCalled()
    await user.click(screen.getByRole("button", { name: "刷新状态" }))
    await waitFor(() => expect(refreshSubscription).toHaveBeenCalledWith(item.id))
    await user.click(screen.getByRole("button", { name: "退出登录" }))
    await waitFor(() => expect(logoutSubscription).toHaveBeenCalledWith(item.id))
    await user.click(screen.getByRole("button", { name: "重新登录" }))
    await user.click(screen.getByRole("button", { name: "开始授权" }))
    await waitFor(() =>
      expect(beginSubscription).toHaveBeenCalledWith(
        expect.objectContaining({ accountId: item.id }),
      ),
    )
  })
  it("keeps a readable title when identity is missing and shows authorization warnings", () => {
    setup(
      <AccountDetailsCard
        account={{ ...item, authEmail: "", authPlan: "", authState: "reauth_required" }}
        busy={false}
        onToggle={vi.fn()}
        onDelete={vi.fn()}
        onEdit={vi.fn()}
      />,
    )
    expect(screen.getByRole("heading", { name: item.name })).toBeInTheDocument()
    expect(screen.getByText("需要重新登录")).toBeInTheDocument()
    expect(screen.getByText("套餐未知")).toBeInTheDocument()
  })
  it("keeps logged-out status visible and removes the logout action", () => {
    setup(<AccountKeyPreview account={{ ...item, authState: "logged_out" }} />)
    expect(screen.getByText("已退出登录")).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "退出登录" })).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "重新登录" })).toBeInTheDocument()
  })
})
