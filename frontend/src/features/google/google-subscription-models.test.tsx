import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import {
  readGoogleSubscriptionModels,
  readGoogleSubscriptionQuota,
} from "../../services/google-subscription"
import { GoogleSubscriptionModels } from "./google-subscription-models"
import { GoogleSubscriptionQuota } from "./google-subscription-quota"
import {
  googleQuotaAccount,
  googleModelsFixture,
  googleQuotaFixture,
} from "./google-quota-test-helpers"

vi.mock("../../services/google-subscription", () => ({
  readGoogleSubscriptionModels: vi.fn(),
  readGoogleSubscriptionQuota: vi.fn(),
}))

function setup(quota = false) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        {quota ? (
          <GoogleSubscriptionQuota account={googleQuotaAccount} />
        ) : (
          <GoogleSubscriptionModels account={googleQuotaAccount} />
        )}
      </QueryClientProvider>
    </I18nProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(readGoogleSubscriptionModels).mockResolvedValue(googleModelsFixture)
  vi.mocked(readGoogleSubscriptionQuota).mockResolvedValue(googleQuotaFixture)
})

it("shows current account models and separates gateway adapters from upstream choices", async () => {
  const user = userEvent.setup()
  const view = setup()
  const trigger = await screen.findByRole("button", { name: "查看全部 3 个账号模型" })
  expect(screen.getByText("已接入 2 个 · 上游可选 3 个")).toBeInTheDocument()
  await user.click(trigger)
  const dialog = screen.getByRole("dialog", { name: "可选模型" })
  expect(view.container).not.toContainElement(dialog)
  expect(within(dialog).getByText("当前网关已接入 · Chat API")).toBeInTheDocument()
  expect(within(dialog).getByText("上游可选 · 当前网关待接入")).toBeInTheDocument()
  expect(within(dialog).getByText("gemini-current-high")).toBeInTheDocument()
  expect(within(dialog).getByText("claude-current")).toBeInTheDocument()
  await user.keyboard("{Escape}")
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
  expect(trigger).toHaveFocus()
})

it("keeps successful quota visible when model discovery fails", async () => {
  vi.mocked(readGoogleSubscriptionModels).mockRejectedValueOnce(new Error("private-model-token"))
  setup(true)
  expect(await screen.findByRole("alert")).toHaveTextContent("模型列表暂时不可读")
  expect(screen.getAllByRole("progressbar")).toHaveLength(4)
  expect(screen.queryByText(/private-model-token/)).not.toBeInTheDocument()
  expect(readGoogleSubscriptionQuota).toHaveBeenCalledTimes(1)
})

it("retains the last model list while refresh fails", async () => {
  const user = userEvent.setup()
  setup()
  await screen.findByRole("button", { name: "查看全部 3 个账号模型" })
  vi.mocked(readGoogleSubscriptionModels).mockRejectedValueOnce(new Error("upstream changed"))
  await user.click(screen.getByRole("button", { name: "刷新 Google 模型列表" }))
  await screen.findByRole("alert")
  expect(screen.getByRole("button", { name: "查看全部 3 个账号模型" })).toBeInTheDocument()
})
