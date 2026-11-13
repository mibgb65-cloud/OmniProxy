import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import type { AccountItem } from "../../services/omniproxy"
import { getDuMatePoints } from "../../services/omniproxy"
import { AccountDetailsCard } from "../accounts/account-details-card"
import { AccountDetailsList } from "../accounts/account-details-list"

vi.mock("../../services/omniproxy", async (original) => ({
  ...(await original<typeof import("../../services/omniproxy")>()),
  getDuMatePoints: vi.fn(),
  checkinDuMate: vi.fn(),
}))
vi.mock("@wailsio/runtime", () => ({ Events: { On: vi.fn(() => () => {}) } }))

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
const actions = { busy: false, onToggle: vi.fn(), onDelete: vi.fn(), onEdit: vi.fn() }

function setup(node: React.ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <I18nProvider>
      <QueryClientProvider client={client}>{node}</QueryClientProvider>
    </I18nProvider>,
  )
}

describe("DuMate account rendering", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useUIStore.setState({ language: "zh" })
    vi.mocked(getDuMatePoints).mockResolvedValue({
      state: "live",
      fetchedAt: "2026-11-13T08:00:00Z",
      points: {
        totalPoints: 100,
        usedPoints: 40,
        remainingPoints: 60,
        checkinIssued: true,
        checkinDays: 3,
        checkinKnown: true,
      },
    })
  })

  it("shows point panel and imported status instead of an API key", async () => {
    setup(<AccountDetailsCard account={account} {...actions} />)
    expect(await screen.findByText("积分与签到")).toBeInTheDocument()
    expect(await screen.findByText("60 积分")).toBeInTheDocument()
    expect(screen.getByText("登录态已导入")).toBeInTheDocument()
    expect(screen.queryByText("API Key")).not.toBeInTheDocument()
    expect(screen.getAllByText("百度账号登录态").length).toBeGreaterThan(0)
  })

  it("uses a point column when the list contains only DuMate accounts", async () => {
    setup(<AccountDetailsList accounts={[account]} {...actions} />)
    expect(screen.getByRole("columnheader", { name: "积分与签到" })).toBeInTheDocument()
    expect(await screen.findByText("60 积分")).toBeInTheDocument()
  })
})
