import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import {
  readCachedGoogleSubscriptionQuota,
  readGoogleSubscriptionQuota,
} from "../../services/google-subscription"
import { GoogleSubscriptionQuota } from "./google-subscription-quota"
import { googleQuotaAccount, googleQuotaFixture } from "./google-quota-test-helpers"

vi.mock("../../services/google-subscription", () => ({
  readCachedGoogleSubscriptionQuota: vi.fn(),
  readGoogleSubscriptionQuota: vi.fn(),
  readGoogleSubscriptionModels: vi.fn(async () => []),
}))

beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(readCachedGoogleSubscriptionQuota).mockResolvedValue({
    ...googleQuotaFixture,
    fromCache: true,
  })
  vi.mocked(readGoogleSubscriptionQuota).mockRejectedValue(
    new Error("google_subscription: upstream_unavailable"),
  )
})

function setup(account = googleQuotaAccount) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <GoogleSubscriptionQuota account={account} />
      </QueryClientProvider>
    </I18nProvider>,
  )
}

it("restores persisted percentages before a failed read and across a cold query client", async () => {
  const first = setup()
  expect(await screen.findByText("80%")).toBeInTheDocument()
  expect(await screen.findByRole("alert")).toBeInTheDocument()
  expect(screen.getByText("已保留上次获取的信息")).toBeInTheDocument()
  expect(screen.queryByText("已达到官方重置时间，请刷新确认最新额度。")).not.toBeInTheDocument()
  first.unmount()
  setup()
  expect(await screen.findByText("80%")).toBeInTheDocument()
  await waitFor(() => expect(readCachedGoogleSubscriptionQuota).toHaveBeenCalledTimes(2))
})

it("keeps local values visible while waiting for the upstream", async () => {
  vi.mocked(readGoogleSubscriptionQuota).mockImplementation(() => new Promise(() => {}))
  setup()
  expect(await screen.findByText("80%")).toBeInTheDocument()
  expect(screen.queryByRole("alert")).not.toBeInTheDocument()
})

it("does not restore or query quotas for a logged-out account", async () => {
  setup({ ...googleQuotaAccount, authState: "logged_out" })
  expect(readCachedGoogleSubscriptionQuota).not.toHaveBeenCalled()
  expect(readGoogleSubscriptionQuota).not.toHaveBeenCalled()
  expect(screen.queryByText("80%")).not.toBeInTheDocument()
})
