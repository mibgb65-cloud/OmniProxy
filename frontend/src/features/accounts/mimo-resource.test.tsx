import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { openMiMoConsole } from "../../services/mimo"
import { useUIStore } from "../../stores/ui-store"
import { describeQuotaAccounts, makeQuotaEntry, quotaSummary } from "../quota/quota-model"
import { QuotaResource } from "../quota/quota-resource"

vi.mock("../../services/mimo", async (original) => ({
  ...(await original<typeof import("../../services/mimo")>()),
  openMiMoConsole: vi.fn(),
}))
const provider = {
  id: "builtin_mimo",
  kind: "mimo",
  source: "builtin",
  name: "MiMo",
  baseUrl: "",
  enabled: true,
  createdAt: "",
  updatedAt: "",
}
const account = {
  id: "mimo-plan",
  providerId: provider.id,
  name: "MiMo",
  enabled: true,
  authMethod: "api_key",
  createdAt: "",
  updatedAt: "v1",
  mimo: { region: "ams", billingMode: "token_plan", version: 1 },
}
beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(openMiMoConsole).mockResolvedValue(undefined)
})

it("shows unknown official plan credits and counts them as uncertain without issuing a query", async () => {
  const item = describeQuotaAccounts([account], [provider])[0]!
  expect(item).toMatchObject({ kind: "mimo_plan", canQuery: false })
  const entry = makeQuotaEntry(item, { pending: false, fetching: false, error: null }, Date.now())
  expect(entry.status).toBe("unknown")
  expect(quotaSummary([entry])).toMatchObject({ monitored: 1, exhausted: null, uncertain: 1 })
  render(
    <I18nProvider>
      <QuotaResource entry={entry} now={Date.now()} />
    </I18nProvider>,
  )
  expect(screen.getByText("Token Plan 套餐额度")).toBeInTheDocument()
  expect(screen.getByText(/官方剩余 Credits 暂未查询/)).toBeInTheDocument()
  expect(screen.queryByText(/100%|0%|¥0/)).not.toBeInTheDocument()
  expect(openMiMoConsole).not.toHaveBeenCalled()
  await userEvent.setup().click(screen.getByRole("button", { name: "查看官方套餐" }))
  expect(openMiMoConsole).toHaveBeenCalledExactlyOnceWith(account.id)
})
it("keeps an invalid or ordinary channel out of automatic plan monitoring", () => {
  for (const mimo of [
    undefined,
    { ...account.mimo, region: "unknown" },
    { ...account.mimo, billingMode: "payg", region: "" },
  ]) {
    expect(describeQuotaAccounts([{ ...account, mimo }], [provider])[0]).toMatchObject({
      kind: "unsupported",
      canQuery: false,
    })
  }
})
it("offers a safe retry when opening the official plan fails", async () => {
  vi.mocked(openMiMoConsole).mockRejectedValue(new Error("private-browser-error"))
  const item = describeQuotaAccounts([account], [provider])[0]!
  render(
    <I18nProvider>
      <QuotaResource
        entry={makeQuotaEntry(item, { pending: false, fetching: false, error: null }, Date.now())}
        now={Date.now()}
      />
    </I18nProvider>,
  )
  await userEvent.setup().click(screen.getByRole("button", { name: "查看官方套餐" }))
  expect(await screen.findByRole("alert")).toHaveTextContent("未能打开官方平台")
  expect(screen.queryByText(/private-browser-error/)).not.toBeInTheDocument()
})
