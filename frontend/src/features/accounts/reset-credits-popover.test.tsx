import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import type { AccountItem } from "../../services/omniproxy"
import {
  consumeSubscriptionReset,
  readSubscriptionQuota,
  readSubscriptionResetStatus,
} from "../../services/subscription"
import { useUIStore } from "../../stores/ui-store"
import { ResetCreditsPopover } from "./reset-credits-popover"

vi.mock("../../services/subscription", async (original) => ({
  ...(await original<typeof import("../../services/subscription")>()),
  readSubscriptionQuota: vi.fn(),
  readSubscriptionResetStatus: vi.fn(),
  consumeSubscriptionReset: vi.fn(),
}))

const account: AccountItem = {
  id: "popover-test",
  providerId: "builtin_openai",
  name: "重置卡面板测试账号",
  authMethod: "subscription_login",
  authState: "authenticated",
  enabled: true,
  createdAt: "2026-10-13T00:00:00Z",
  updatedAt: "2026-10-13T00:00:00Z",
}

function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <ResetCreditsPopover account={account} disabled={false} />
        <button onClick={outsideAction}>面板外操作</button>
      </QueryClientProvider>
    </I18nProvider>,
  )
  return userEvent.setup()
}

const outsideAction = vi.fn()

describe("重置卡局部面板", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useUIStore.setState({ language: "zh" })
    vi.mocked(readSubscriptionQuota).mockResolvedValue({
      plan: "plus",
      fiveHourEnabled: true,
      fetchedAt: "2026-10-13T00:00:00Z",
      resetCredits: { availableCount: 2, detailsKnown: false, credits: [] },
    })
    vi.mocked(readSubscriptionResetStatus).mockResolvedValue({
      operationId: "",
      creditId: "",
      state: "",
      outcome: "",
    })
  })

  it.each(["Escape", "入口按钮", "关闭按钮"])("通过%s关闭并恢复入口焦点", async (action) => {
    const user = setup()
    const trigger = screen.getByRole("button", { name: "重置卡" })
    await user.click(trigger)
    const panel = screen.getByRole("dialog", { name: "重置卡" })
    expect(panel).not.toHaveAttribute("aria-modal", "true")
    expect(panel).toHaveAccessibleDescription(account.name)
    expect(trigger).toHaveAttribute("aria-expanded", "true")
    if (action === "Escape") await user.keyboard("{Escape}")
    else if (action === "入口按钮") await user.click(trigger)
    else await user.click(within(panel).getByRole("button", { name: "关闭重置卡窗口" }))
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
    expect(trigger).toHaveAttribute("aria-expanded", "false")
    expect(trigger).toHaveFocus()
    expect(consumeSubscriptionReset).not.toHaveBeenCalled()
  })

  it("面板外的操作可直接响应并收起面板", async () => {
    const user = setup()
    await user.click(screen.getByRole("button", { name: "重置卡" }))
    const outside = screen.getByRole("button", { name: "面板外操作" })
    await user.click(outside)
    expect(outsideAction).toHaveBeenCalledTimes(1)
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    expect(outside).toHaveFocus()
  })

  it("使用中阻止关闭，完成后可以关闭", async () => {
    let finish!: () => void
    vi.mocked(consumeSubscriptionReset).mockImplementationOnce(
      (input) =>
        new Promise((resolve) => {
          finish = () =>
            resolve({
              operation: { ...input, state: "completed", outcome: "reset" },
              refreshFailed: false,
            })
        }),
    )
    const user = setup()
    const trigger = screen.getByRole("button", { name: "重置卡" })
    await user.click(trigger)
    const panel = screen.getByRole("dialog", { name: "重置卡" })
    const next = within(panel).getByRole("button", { name: "使用下一张可用卡" })
    await waitFor(() => expect(next).toBeEnabled())
    await user.click(next)
    await user.click(within(panel).getByRole("button", { name: "确认使用" }))
    expect(within(panel).getByRole("button", { name: "关闭重置卡窗口" })).toBeDisabled()
    await user.keyboard("{Escape}")
    await user.click(trigger)
    await user.click(screen.getByRole("button", { name: "面板外操作" }))
    expect(panel).toBeInTheDocument()
    expect(consumeSubscriptionReset).toHaveBeenCalledTimes(1)
    finish()
    await within(panel).findByText("重置卡使用成功。")
    await user.keyboard("{Escape}")
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
  })
})
