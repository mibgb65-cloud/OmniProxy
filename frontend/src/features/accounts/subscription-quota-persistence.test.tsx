import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { act, render, screen, waitFor } from "@testing-library/react"
import { afterEach, beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import type { AccountItem } from "../../services/omniproxy"
import {
  readCachedSubscriptionQuota,
  readSubscriptionQuota,
  type SubscriptionQuota as Quota,
} from "../../services/subscription"
import { SubscriptionQuota } from "./subscription-quota"

vi.mock("../../services/subscription", async (original) => ({
  ...(await original<typeof import("../../services/subscription")>()),
  readSubscriptionQuota: vi.fn(),
  readCachedSubscriptionQuota: vi.fn(),
  readSubscriptionQuotaEstimate: vi.fn(async () => ({ fetchedAt: "" })),
}))
const account: AccountItem = {
  id: "acct_persistent",
  providerId: "builtin_openai",
  name: "测试账号",
  authMethod: "subscription_login",
  authState: "authenticated",
  authEmail: "member@example.test",
  authAccountId: "shared-workspace",
  authPlan: "plus",
  enabled: true,
  createdAt: "",
  updatedAt: "",
}
const cached: Quota = {
  plan: "plus",
  fiveHourEnabled: true,
  fiveHour: { remainingPercent: 75, resetsAt: "2099-10-20T05:00:00Z" },
  weekly: { remainingPercent: 40, resetsAt: "2099-10-27T00:00:00Z" },
  fetchedAt: "2020-01-01T00:00:00Z",
  fromCache: true,
}
const fresh: Quota = {
  ...cached,
  fiveHour: { ...cached.fiveHour!, remainingPercent: 65 },
  fetchedAt: "2026-10-20T00:00:00Z",
  fromCache: false,
}
let pending: Array<() => void> = []
function holdLive() {
  let finish!: (value: Quota) => void
  vi.mocked(readSubscriptionQuota).mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve
      }),
  )
  pending.push(() => finish?.(fresh))
  return async (value = fresh) => act(async () => finish?.(value))
}
function mount(
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } }),
  twice = false,
) {
  const view = (item = account) => (
    <I18nProvider>
      <QueryClientProvider client={client}>
        <SubscriptionQuota account={item} />
        {twice ? <SubscriptionQuota account={item} /> : null}
      </QueryClientProvider>
    </I18nProvider>
  )
  const result = render(view())
  return { ...result, client, changeAccount: (item: AccountItem) => result.rerender(view(item)) }
}
beforeEach(() => {
  vi.clearAllMocks()
  pending = []
  useUIStore.setState({ language: "zh" })
  vi.mocked(readCachedSubscriptionQuota).mockResolvedValue(cached)
  vi.mocked(readSubscriptionQuota).mockResolvedValue(fresh)
})
afterEach(async () => {
  await act(async () => pending.forEach((finish) => finish()))
})

it("冷启动先显示持久快照，等待网络时不回到占位，成功后原位更新", async () => {
  const finish = holdLive()
  mount()
  const bar = await screen.findByRole("progressbar", { name: "5h 额度剩余额度" })
  expect(bar).toHaveAttribute("aria-valuenow", "75")
  expect(screen.getByText("已保留上次获取的信息")).toBeInTheDocument()
  expect(screen.getAllByText(/后重置/)).toHaveLength(2)
  await waitFor(() => expect(readSubscriptionQuota).toHaveBeenCalledTimes(1))
  await finish()
  expect(await screen.findByText("65%")).toBeInTheDocument()
  expect(screen.getByRole("progressbar", { name: "5h 额度剩余额度" })).toBe(bar)
  expect(screen.queryByText("已保留上次获取的信息")).not.toBeInTheDocument()
})

it("刷新失败保留历史百分比和过期重置时间，不虚构为满额", async () => {
  vi.mocked(readCachedSubscriptionQuota).mockResolvedValue({
    ...cached,
    fiveHour: { remainingPercent: 0, resetsAt: "2020-01-01T01:00:00Z" },
  })
  vi.mocked(readSubscriptionQuota).mockRejectedValue(new Error("subscription: quota_unavailable"))
  mount()
  expect(await screen.findByRole("alert")).toHaveTextContent("额度查询失败")
  expect(screen.getByText("0%")).toBeInTheDocument()
  expect(screen.getByText("待刷新")).toBeInTheDocument()
  expect(readSubscriptionQuota).toHaveBeenCalledTimes(1)
})

it("页面重进使用内存快照，丢弃 QueryClient 后仍从持久来源恢复", async () => {
  const first = mount()
  await screen.findByText("65%")
  first.unmount()
  const second = mount(first.client)
  expect(screen.getByText("65%")).toBeInTheDocument()
  expect(readCachedSubscriptionQuota).toHaveBeenCalledTimes(1)
  second.unmount()
  first.client.clear()
  vi.mocked(readCachedSubscriptionQuota).mockResolvedValue({ ...fresh, fromCache: true })
  const finish = holdLive()
  mount()
  expect(await screen.findByText("65%")).toBeInTheDocument()
  await waitFor(() => expect(readSubscriptionQuota).toHaveBeenCalledTimes(2))
  expect(readCachedSubscriptionQuota).toHaveBeenCalledTimes(2)
  await finish()
})

it("相同工作区下切换成员也不复用旧快照，退出登录后不读取额度", async () => {
  const view = mount()
  await screen.findByText("65%")
  vi.mocked(readCachedSubscriptionQuota).mockResolvedValue(null)
  const finish = holdLive()
  view.changeAccount({ ...account, authEmail: "other@example.test" })
  await waitFor(() => expect(readCachedSubscriptionQuota).toHaveBeenCalledTimes(2))
  expect(screen.queryByText("65%")).not.toBeInTheDocument()
  view.changeAccount({ ...account, authState: "logged_out" })
  expect(screen.getByText("登录后可查询额度。")).toBeInTheDocument()
  expect(screen.queryByRole("progressbar")).not.toBeInTheDocument()
  await finish()
})

it("多个额度组件共享一次本地恢复及一次实时读取", async () => {
  const finish = holdLive()
  mount(undefined, true)
  await waitFor(() => expect(screen.getAllByText("75%")).toHaveLength(2))
  await waitFor(() => expect(readSubscriptionQuota).toHaveBeenCalledTimes(1))
  expect(readCachedSubscriptionQuota).toHaveBeenCalledTimes(1)
  await finish()
  await waitFor(() => expect(screen.getAllByText("65%")).toHaveLength(2))
})

it("本地恢复暂时失败仍能查询新额度，不泄露存储错误", async () => {
  vi.mocked(readCachedSubscriptionQuota).mockRejectedValue(new Error("private-storage-detail"))
  mount()
  expect(await screen.findByText("65%")).toBeInTheDocument()
  expect(screen.queryByText(/private-storage-detail/)).not.toBeInTheDocument()
  expect(readSubscriptionQuota).toHaveBeenCalledTimes(1)
})
