import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import type { AccountItem } from "../../services/omniproxy"
import {
  consumeSubscriptionReset,
  readSubscriptionQuota,
  readSubscriptionResetStatus,
  type SubscriptionQuota,
  type SubscriptionResetOperation,
} from "../../services/subscription"
import { ResetCreditsPopover } from "./reset-credits-popover"

vi.mock("../../services/subscription", async (original) => ({
  ...(await original<typeof import("../../services/subscription")>()),
  readSubscriptionQuota: vi.fn(),
  readCachedSubscriptionQuota: vi.fn(async () => null),
  readSubscriptionResetStatus: vi.fn(),
  consumeSubscriptionReset: vi.fn(),
}))
const account: AccountItem = {
  id: "acct_reset",
  providerId: "builtin_openai",
  name: "重置测试账号",
  authMethod: "subscription_login",
  authState: "authenticated",
  authPlan: "plus",
  enabled: true,
  createdAt: "2026-10-09T00:00:00Z",
  updatedAt: "2026-10-09T00:00:00Z",
}
let quota: SubscriptionQuota
let status: SubscriptionResetOperation
function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <ResetCreditsPopover account={account} disabled={false} />
      </QueryClientProvider>
    </I18nProvider>,
  )
  return userEvent.setup()
}
async function open(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole("button", { name: "重置卡" }))
  const dialog = screen.getByRole("dialog", { name: "重置卡" })
  await within(dialog).findByText("测试重置卡")
  await waitFor(() =>
    expect(within(dialog).getByRole("button", { name: "使用这张卡" })).toBeEnabled(),
  )
  return dialog
}
describe("manual reset credits", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useUIStore.setState({ language: "zh" })
    status = { operationId: "", creditId: "", state: "", outcome: "" }
    quota = {
      plan: "plus",
      fiveHourEnabled: true,
      fetchedAt: "2026-10-09T00:00:00Z",
      weekly: { remainingPercent: 50 },
      resetCredits: {
        availableCount: 3,
        detailsKnown: true,
        credits: [
          {
            id: "credit-test",
            title: "测试重置卡",
            description: "模拟重置权益",
            resetType: "codexRateLimits",
            status: "available",
            grantedAt: "2026-10-08T00:00:00Z",
            expiresAt: "2099-10-09T00:00:00Z",
          },
        ],
      },
    }
    vi.mocked(readSubscriptionQuota).mockImplementation(async () => quota)
    vi.mocked(readSubscriptionResetStatus).mockImplementation(async () => status)
    vi.mocked(consumeSubscriptionReset).mockImplementation(async (input) => {
      status = {
        operationId: input.operationId,
        creditId: input.creditId,
        state: "completed",
        outcome: "reset",
      }
      quota = { ...quota, resetCredits: { ...quota.resetCredits!, availableCount: 2 } }
      return { operation: status, quota, refreshFailed: false }
    })
  })
  it("uses the authoritative count and requires confirmation before a single consume", async () => {
    const user = setup()
    const dialog = await open(user)
    expect(within(dialog).getByText("3")).toBeInTheDocument()
    expect(within(dialog).getByText(/部分卡片详情/)).toBeInTheDocument()
    await user.click(within(dialog).getByRole("button", { name: "使用这张卡" }))
    expect(consumeSubscriptionReset).not.toHaveBeenCalled()
    await user.dblClick(within(dialog).getByRole("button", { name: "确认使用" }))
    expect(await within(dialog).findByText("重置卡使用成功。")).toBeInTheDocument()
    expect(consumeSubscriptionReset).toHaveBeenCalledTimes(1)
    expect(consumeSubscriptionReset).toHaveBeenCalledWith({
      accountId: account.id,
      creditId: "credit-test",
      operationId: expect.stringMatching(
        /^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$/,
      ),
    })
    expect(within(dialog).getByText("2")).toBeInTheDocument()
  })
  it("restores an uncertain attempt on reopen and verifies with the original key", async () => {
    vi.mocked(consumeSubscriptionReset).mockImplementationOnce(async (input) => {
      status = {
        operationId: input.operationId,
        creditId: input.creditId,
        state: "pending",
        outcome: "",
      }
      return { operation: status, refreshFailed: false }
    })
    const user = setup()
    let dialog = await open(user)
    await user.click(within(dialog).getByRole("button", { name: "使用这张卡" }))
    await user.click(within(dialog).getByRole("button", { name: "确认使用" }))
    await within(dialog).findByText(/本次使用结果尚未确认/)
    const first = vi.mocked(consumeSubscriptionReset).mock.calls[0][0]
    await user.click(within(dialog).getByRole("button", { name: "关闭重置卡窗口" }))
    await user.click(screen.getByRole("button", { name: "重置卡" }))
    dialog = screen.getByRole("dialog", { name: "重置卡" })
    expect(await within(dialog).findByText(/本次使用结果尚未确认/)).toBeInTheDocument()
    expect(within(dialog).getByRole("button", { name: "使用下一张可用卡" })).toBeDisabled()
    await user.click(within(dialog).getByRole("button", { name: "核对本次使用" }))
    await waitFor(() => expect(consumeSubscriptionReset).toHaveBeenCalledTimes(2))
    expect(vi.mocked(consumeSubscriptionReset).mock.calls[1][0]).toEqual(first)
  })
  it("keeps unavailable information distinct from zero inventory", async () => {
    quota = { ...quota, resetCredits: null }
    const user = setup()
    await user.click(screen.getByRole("button", { name: "重置卡" }))
    const dialog = screen.getByRole("dialog", { name: "重置卡" })
    expect(await within(dialog).findByText("官方暂未返回重置卡信息。")).toBeInTheDocument()
    expect(within(dialog).getByRole("button", { name: "使用下一张可用卡" })).toBeDisabled()
    quota = { ...quota, resetCredits: { availableCount: 0, detailsKnown: true, credits: [] } }
    await user.click(within(dialog).getByRole("button", { name: "刷新重置卡" }))
    expect(await within(dialog).findByText("当前没有可用重置卡。")).toBeInTheDocument()
    expect(consumeSubscriptionReset).not.toHaveBeenCalled()
  })
  it("supports count-only inventory using the official next-credit choice", async () => {
    quota = { ...quota, resetCredits: { availableCount: 2, detailsKnown: false, credits: [] } }
    const user = setup()
    await user.click(screen.getByRole("button", { name: "重置卡" }))
    const dialog = screen.getByRole("dialog", { name: "重置卡" })
    await waitFor(() =>
      expect(within(dialog).getByRole("button", { name: "使用下一张可用卡" })).toBeEnabled(),
    )
    await user.click(within(dialog).getByRole("button", { name: "使用下一张可用卡" }))
    await user.click(within(dialog).getByRole("button", { name: "确认使用" }))
    await waitFor(() =>
      expect(consumeSubscriptionReset).toHaveBeenCalledWith(
        expect.objectContaining({ creditId: "" }),
      ),
    )
  })
})
