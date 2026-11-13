import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import type { AccountItem, DuMatePoints as DuMatePointsData } from "../../services/omniproxy"
import { checkinDuMate, getDuMatePoints } from "../../services/omniproxy"
import { DuMatePoints } from "./dumate-points"

vi.mock("../../services/omniproxy", async (original) => ({
  ...(await original<typeof import("../../services/omniproxy")>()),
  getDuMatePoints: vi.fn(),
  checkinDuMate: vi.fn(),
}))

const account: AccountItem = {
  id: "acct_dumate",
  providerId: "builtin_dumate",
  name: "百度搭子",
  authMethod: "dumate_cookie",
  authState: "authenticated",
  enabled: true,
  createdAt: "2026-11-13T00:00:00Z",
  updatedAt: "2026-11-13T00:00:00Z",
}

const live: DuMatePointsData = {
  state: "live",
  fetchedAt: "2026-11-13T08:00:00Z",
  points: {
    totalPoints: 100,
    usedPoints: 40,
    remainingPoints: 60,
    checkinIssued: false,
    checkinDays: 2,
    checkinKnown: true,
    expiringAt: "2026-12-01T00:00:00Z",
    expiringPoints: 30,
  },
}

function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <DuMatePoints account={account} />
      </QueryClientProvider>
    </I18nProvider>,
  )
}

describe("DuMate points", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useUIStore.setState({ language: "zh" })
    vi.mocked(getDuMatePoints).mockResolvedValue(live)
    vi.mocked(checkinDuMate).mockResolvedValue({
      outcome: "claimed",
      credit: 500,
      totalTimes: 3,
      checkedAt: "2026-11-13T08:01:00Z",
    })
  })

  it("shows points, expiry and claims the daily check-in", async () => {
    const user = userEvent.setup()
    setup()
    expect(await screen.findByText("60 积分")).toBeInTheDocument()
    expect(screen.getByText("40 / 100")).toBeInTheDocument()
    expect(screen.getByText(/30 积分将于/)).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "签到" }))
    expect(checkinDuMate).toHaveBeenCalledWith("acct_dumate")
    expect(await screen.findByText(/签到成功 \+500/)).toBeInTheDocument()
  })

  it("hides the button when today is already checked in", async () => {
    vi.mocked(getDuMatePoints).mockResolvedValue({
      ...live,
      points: { ...live.points!, checkinIssued: true, checkinDays: 9 },
    })
    setup()
    expect(await screen.findByText("今日已签到（累计 9 天）")).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "签到" })).not.toBeInTheDocument()
  })

  it("reports a failed query without showing zero points", async () => {
    vi.mocked(getDuMatePoints).mockResolvedValue({
      state: "unavailable",
      errorCode: "console_unavailable",
    })
    setup()
    expect(await screen.findByRole("alert")).toHaveTextContent("控制台不可用")
    expect(screen.queryByText("0 积分")).not.toBeInTheDocument()
  })

  it("labels historical values as stale when the query fails", async () => {
    vi.mocked(getDuMatePoints).mockResolvedValue({
      ...live,
      state: "stale",
      errorCode: "rate_limited",
    })
    setup()
    expect(await screen.findByText("60 积分")).toBeInTheDocument()
    expect(screen.getByText(/显示最近一次成功结果/)).toHaveTextContent("控制台限流")
  })

  it("keeps unknown check-in state distinguishable from an unchecked day", async () => {
    vi.mocked(getDuMatePoints).mockResolvedValue({
      ...live,
      points: { ...live.points!, checkinKnown: false, checkinIssued: false },
    })
    setup()
    expect(await screen.findByText("签到状态未知")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "签到" })).toBeEnabled()
  })
})
