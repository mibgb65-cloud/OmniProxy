import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import type { AccountItem } from "../../services/omniproxy"
import { workbuddyCachedQuota, workbuddyQuota, type WorkBuddyQuota } from "../../services/workbuddy"
import { useUIStore } from "../../stores/ui-store"
import { AccountDetailsCard } from "./account-details-card"
import { WorkBuddyCredits } from "./workbuddy-credits"

vi.mock("@wailsio/runtime", () => ({ Events: { On: vi.fn(() => vi.fn()) } }))

vi.mock("../../services/workbuddy", async (original) => ({
  ...(await original<typeof import("../../services/workbuddy")>()),
  workbuddyQuota: vi.fn(),
  workbuddyCachedQuota: vi.fn(async () => null),
}))

const account: AccountItem = {
  id: "workbuddy-one",
  providerId: "builtin_workbuddy_ai",
  name: "WorkBuddy",
  enabled: true,
  authMethod: "workbuddy_login",
  authState: "authenticated",
  authPlan: "Pro",
  createdAt: "2026-11-16T00:00:00Z",
  updatedAt: "v1",
}

const quota: WorkBuddyQuota = {
  plan: "Pro",
  fetchedAt: "2026-11-16T10:00:00Z",
  windows: [
    { name: "Credits", used: 40, amount: 400, limit: 1000 },
    { name: "Extra credits", used: 25, amount: 50, limit: 200, aside: true },
    {
      name: "Total credits",
      used: 37.5,
      amount: 450,
      limit: 1200,
      resetsAt: "2026-12-01T00:00:00Z",
      runsOutAt: "2026-11-20T00:00:00Z",
    },
    { name: "Prepaid credits", used: 0, amount: 300, prepaid: true },
  ],
}

function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return {
    user: userEvent.setup(),
    ...render(
      <I18nProvider>
        <QueryClientProvider client={client}>
          <WorkBuddyCredits account={account} />
        </QueryClientProvider>
      </I18nProvider>,
    ),
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(workbuddyCachedQuota).mockResolvedValue(null)
  vi.mocked(workbuddyQuota).mockResolvedValue(quota)
})

it("按三窗口语义展示 Credits、额外 Credits、合计与预付余额", async () => {
  setup()
  expect(await screen.findByText("WorkBuddy AI Credits")).toBeInTheDocument()
  expect(await screen.findByText("计划内 Credits")).toBeInTheDocument()
  expect(screen.getByText("额外 Credits")).toBeInTheDocument()
  expect(screen.getByText("Credits 合计")).toBeInTheDocument()
  expect(screen.getByText("预付余额")).toBeInTheDocument()
  expect(screen.getByText("400 / 1,000")).toBeInTheDocument()
  expect(screen.getByText("50 / 200")).toBeInTheDocument()
  expect(screen.getByText("450 / 1,200")).toBeInTheDocument()
  // 预付余额只有余额，不参与百分比。
  expect(screen.getByText("300")).toBeInTheDocument()
  // aside 窗口必须明确说明不计入账号可用性。
  expect(screen.getByText("不计入账号可用性")).toBeInTheDocument()
  expect(screen.getByText("重置时间")).toBeInTheDocument()
  expect(screen.getByText("最早到期")).toBeInTheDocument()
  expect(screen.getByText("已用 37.5%")).toBeInTheDocument()
})

it("使用英文标签时保持相同的窗口语义", async () => {
  useUIStore.setState({ language: "en" })
  setup()
  expect(await screen.findByText("Plan credits")).toBeInTheDocument()
  expect(screen.getByText("Extra credits")).toBeInTheDocument()
  expect(screen.getByText("Total credits")).toBeInTheDocument()
  expect(screen.getByText("Prepaid balance")).toBeInTheDocument()
  expect(screen.getByText("Not counted toward account availability")).toBeInTheDocument()
  expect(screen.getByText("Resets at")).toBeInTheDocument()
  expect(screen.getByText("Earliest expiry")).toBeInTheDocument()
})

it("未知数量显示破折号而不是 0", async () => {
  vi.mocked(workbuddyQuota).mockResolvedValue({
    plan: "Free plan",
    fetchedAt: "2026-11-16T10:00:00Z",
    windows: [{ name: "Credits", used: 40 }],
  })
  setup()
  expect(await screen.findByText("—")).toBeInTheDocument()
  expect(screen.queryByText("0")).not.toBeInTheDocument()
  expect(screen.queryByText("0%")).not.toBeInTheDocument()
  expect(screen.queryByRole("progressbar")).not.toBeInTheDocument()
})

it("刷新失败时显示稳定错误码文案并保留上次成功的数据", async () => {
  const { user } = setup()
  expect(await screen.findByText("400 / 1,000")).toBeInTheDocument()
  vi.mocked(workbuddyQuota).mockRejectedValueOnce(
    new Error("workbuddy_ai: sign_in_expired; private-upstream-body"),
  )
  await user.click(screen.getByRole("button", { name: "刷新 WorkBuddy AI Credits" }))
  expect(await screen.findByRole("alert")).toHaveTextContent("登录已失效，请重新登录。")
  expect(screen.queryByText(/private-upstream-body|private/)).not.toBeInTheDocument()
  expect(screen.getByText("400 / 1,000")).toBeInTheDocument()
})

it("未知错误码回落到通用文案，不原样回显上游文本", async () => {
  vi.mocked(workbuddyQuota).mockRejectedValue(new Error("private-upstream-body"))
  setup()
  expect(await screen.findByRole("alert")).toHaveTextContent(
    "操作未完成，请检查网络和登录状态后重试。",
  )
  expect(screen.queryByText(/private-upstream-body/)).not.toBeInTheDocument()
})

it("冷启动先展示本机保留的快照，再等待联网结果", async () => {
  vi.mocked(workbuddyCachedQuota).mockResolvedValue({ ...quota, fromCache: true })
  vi.mocked(workbuddyQuota).mockImplementation(() => new Promise(() => {}))
  setup()
  expect(await screen.findByText("400 / 1,000")).toBeInTheDocument()
  expect(workbuddyCachedQuota).toHaveBeenCalledWith(account.id)
  expect(workbuddyQuota).toHaveBeenCalledWith(account.id)
  expect(screen.getByText("已保留上次获取的信息")).toBeInTheDocument()
})

it("账号卡用 Credits 面板展示 WorkBuddy 订阅，不落到余额未接入或 Codex 窗口", async () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <AccountDetailsCard
          account={account}
          busy={false}
          onToggle={vi.fn()}
          onDelete={vi.fn()}
          onEdit={vi.fn()}
        />
      </QueryClientProvider>
    </I18nProvider>,
  )
  expect(await screen.findByText("400 / 1,000")).toBeInTheDocument()
  expect(screen.getByText("WorkBuddy AI 订阅")).toBeInTheDocument()
  expect(screen.queryByText("余额未接入")).not.toBeInTheDocument()
  expect(screen.queryByText("5h 额度")).not.toBeInTheDocument()
  expect(screen.queryByText("一周额度")).not.toBeInTheDocument()
  // 浏览器登录账号只有授权状态，没有可复制的 API Key。
  expect(screen.queryByText("API Key")).not.toBeInTheDocument()
})
