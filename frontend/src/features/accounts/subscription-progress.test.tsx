import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { Clipboard } from "@wailsio/runtime"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import {
  beginSubscription,
  cancelSubscription,
  currentSubscription,
  openSubscription,
  readSubscription,
  type SubscriptionSession,
} from "../../services/subscription"
import { SubscriptionLoginDialog } from "./subscription-login-dialog"

vi.mock("@wailsio/runtime", () => ({
  Events: { On: vi.fn(() => vi.fn()) },
  Clipboard: { SetText: vi.fn() },
}))
vi.mock("../../services/subscription", async (original) => ({
  ...(await original<typeof import("../../services/subscription")>()),
  beginSubscription: vi.fn(),
  readSubscription: vi.fn(),
  currentSubscription: vi.fn(),
  cancelSubscription: vi.fn(),
  openSubscription: vi.fn(),
}))

let state: SubscriptionSession
function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <SubscriptionLoginDialog providerId="builtin_openai" />
      </QueryClientProvider>
    </I18nProvider>,
  )
  return { user: userEvent.setup(), client }
}

describe("订阅授权进度和登录链接", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useUIStore.setState({ language: "zh" })
    state = {
      id: "progress-session",
      state: "waiting",
      url: "https://auth.openai.com/authorize?state=test-only",
      userCode: "",
      errorCode: "",
      expiresAt: "",
      cleanupWarning: false,
    }
    vi.mocked(beginSubscription).mockImplementation(async () => ({ ...state }))
    vi.mocked(currentSubscription).mockResolvedValue({ ...state, id: "", state: "" })
    vi.mocked(readSubscription).mockImplementation(async () => ({ ...state }))
    vi.mocked(openSubscription).mockResolvedValue(undefined)
    vi.mocked(Clipboard.SetText).mockResolvedValue()
  })

  it("复制当前会话链接且打开浏览器失败后仍可复制，错误不会泄露授权链接", async () => {
    vi.mocked(openSubscription).mockRejectedValue(new Error("private-browser-error"))
    const { user } = setup()
    await user.click(screen.getByRole("button", { name: "登录 ChatGPT" }))
    await user.click(screen.getByRole("button", { name: "开始授权" }))
    const button = await screen.findByRole("button", { name: "复制登录链接" })
    vi.mocked(Clipboard.SetText).mockRejectedValueOnce(new Error(state.url))
    await user.click(button)
    expect(await screen.findByText("复制失败，请重试。")).toBeInTheDocument()
    expect(screen.queryByText(state.url)).not.toBeInTheDocument()
    await user.click(button)
    await waitFor(() => expect(Clipboard.SetText).toHaveBeenLastCalledWith(state.url))
    expect(await screen.findByRole("button", { name: "登录链接已复制" })).toBeInTheDocument()
    expect(beginSubscription).toHaveBeenCalledTimes(1)
  })

  it("无原生事件也轮询到获取账号及保存阶段，最终停止轮询", async () => {
    const { user, client } = setup()
    await user.click(screen.getByRole("button", { name: "登录 ChatGPT" }))
    await user.click(screen.getByRole("button", { name: "开始授权" }))
    await screen.findByRole("button", { name: "复制登录链接" })
    state = { ...state, state: "verifying", url: "" }
    expect(
      await screen.findByText("官方授权已完成，正在获取账号信息…", {}, { timeout: 2500 }),
    ).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "开始授权" })).not.toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "复制登录链接" })).not.toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "打开官方授权页面" })).not.toBeInTheDocument()
    state = { ...state, state: "saving" }
    await client.invalidateQueries({ queryKey: ["subscription-session"] })
    expect(await screen.findByText("正在保存本地授权凭证与账号…")).toBeInTheDocument()
    state = { ...state, state: "completed" }
    await client.invalidateQueries({ queryKey: ["subscription-session"] })
    expect(await screen.findByText("授权完成，账号已保存。")).toBeInTheDocument()
    const calls = vi.mocked(readSubscription).mock.calls.length
    await new Promise((resolve) => setTimeout(resolve, 1200))
    expect(readSubscription).toHaveBeenCalledTimes(calls)
    expect(beginSubscription).toHaveBeenCalledTimes(1)
  })

  it.each(["exchanging", "verifying", "saving"])("重开弹窗恢复 %s 阶段且可取消", async (phase) => {
    state = { ...state, state: phase, url: "" }
    vi.mocked(currentSubscription).mockImplementation(async () => ({ ...state }))
    vi.mocked(cancelSubscription).mockImplementation(async () => {
      state = { ...state, state: "cancelled" }
      return state
    })
    const { user } = setup()
    await user.click(screen.getByRole("button", { name: "登录 ChatGPT" }))
    expect(await screen.findByText(/可以关闭网页或此窗口/)).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "开始授权" })).not.toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "关闭订阅登录窗口" }))
    expect(cancelSubscription).not.toHaveBeenCalled()
    await user.click(screen.getByRole("button", { name: "登录 ChatGPT" }))
    await user.click(await screen.findByRole("button", { name: "取消" }))
    expect(await screen.findByText("授权已取消。")).toBeInTheDocument()
    expect(cancelSubscription).toHaveBeenCalledWith(state.id)
    expect(beginSubscription).not.toHaveBeenCalled()
  })
})
