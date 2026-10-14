import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { Dialog } from "radix-ui"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { AddAccountDialog } from "./add-account-dialog"
import { AccountEditForm } from "./account-edit-form"
import { StepFunBalanceValues, StepFunPlanResource } from "./stepfun-resource"
import { openStepConsole } from "../../services/stepfun"
import { describeQuotaAccounts, quotaEntryStatus } from "../quota/quota-model"

vi.mock("../../services/omniproxy", () => ({ getAccountKeyPreview: vi.fn(async () => "••••••••") }))
vi.mock("../../services/stepfun", () => ({ openStepConsole: vi.fn(async () => {}) }))
const provider = {
  id: "builtin_stepfun",
  kind: "stepfun",
  name: "StepFun",
  source: "builtin",
  baseUrl: "",
  enabled: true,
  createdAt: "",
  updatedAt: "",
}
const account = {
  id: "step-account",
  providerId: provider.id,
  name: "Step",
  enabled: true,
  authMethod: "api_key",
  createdAt: "",
  updatedAt: "now",
  stepfun: { region: "global", billingMode: "step_plan", version: 2 },
}

beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
})

it("uses one platform and billing selection for the entire batch", async () => {
  const user = userEvent.setup(),
    single = vi.fn(),
    batch = vi.fn(async () => {})
  render(
    <I18nProvider>
      <AddAccountDialog provider={provider} onCreate={single} onBatchCreate={batch} />
    </I18nProvider>,
  )
  await user.click(screen.getByRole("button", { name: "添加账号" }))
  await user.click(screen.getByLabelText("平台"))
  await user.click(screen.getByRole("option", { name: "海外" }))
  await user.click(screen.getByLabelText("计费方式"))
  await user.click(screen.getByRole("option", { name: "Step Plan" }))
  await user.click(screen.getByRole("tab", { name: "批量添加" }))
  await user.type(screen.getByLabelText("API Keys"), "placeholder-key-one\nplaceholder-key-two")
  await user.click(screen.getByRole("button", { name: "保存全部账号" }))
  expect(batch).toHaveBeenCalledWith({
    providerId: provider.id,
    apiKeys: ["placeholder-key-one", "placeholder-key-two"],
    stepfun: { region: "global", billingMode: "step_plan", version: 0 },
  })
  expect(single).not.toHaveBeenCalled()
})

it("edits billing with the saved version and leaves a blank key unchanged", async () => {
  const user = userEvent.setup(),
    save = vi.fn(async () => {})
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <Dialog.Root open>
          <AccountEditForm account={account} onSubmit={save} onRestoreFocus={() => {}} />
        </Dialog.Root>
      </QueryClientProvider>
    </I18nProvider>,
  )
  await user.click(screen.getByLabelText("计费方式"))
  await user.click(screen.getByRole("option", { name: "按量 API" }))
  await user.click(screen.getByRole("button", { name: "保存修改" }))
  expect(save).toHaveBeenCalledWith({
    id: account.id,
    name: "Step",
    apiKey: "",
    stepfun: { region: "global", billingMode: "payg", version: 2 },
  })
})

it("keeps plan credits unknown and opens the regional console through the service", async () => {
  const user = userEvent.setup()
  render(
    <I18nProvider>
      <StepFunPlanResource account={account} />
    </I18nProvider>,
  )
  expect(screen.getByText(/剩余 Credits 未知/)).toBeInTheDocument()
  await user.click(screen.getByRole("button", { name: "打开官方平台" }))
  await waitFor(() => expect(openStepConsole).toHaveBeenCalledWith(account.id))
  const item = describeQuotaAccounts([account], [provider])[0]
  expect(item.kind).toBe("step_plan")
  expect(item.canQuery).toBe(false)
  expect(quotaEntryStatus(item, { pending: false, fetching: false, error: null }, Date.now())).toBe(
    "unknown",
  )
})

it("shows official cumulative amounts without inventing currency or remaining splits", () => {
  render(
    <I18nProvider>
      <StepFunBalanceValues
        value={{ type: "postpaid", amount: "-2.3400", totalCash: "100", totalVoucher: "5" }}
      />
    </I18nProvider>,
  )
  expect(screen.getByText("-2.3400")).toBeInTheDocument()
  expect(screen.getByText("总充值金额")).toBeInTheDocument()
  expect(screen.getByText(/后付费.*币种待确认/)).toBeInTheDocument()
  expect(screen.queryByText("余额不足")).not.toBeInTheDocument()
})

it("shows domestic balances in Chinese yuan without changing decimal precision", () => {
  render(
    <I18nProvider>
      <StepFunBalanceValues
        value={{
          type: "prepaid",
          amount: "123.4500",
          totalCash: "1000",
          totalVoucher: "5",
          currency: "CNY",
        }}
      />
    </I18nProvider>,
  )
  expect(screen.getByText("¥123.4500")).toBeInTheDocument()
  expect(screen.getByText("¥1000")).toBeInTheDocument()
  expect(screen.getByText("¥5")).toBeInTheDocument()
  expect(screen.getByText(/人民币（CNY）/)).toBeInTheDocument()
  expect(screen.queryByText(/币种待确认/)).not.toBeInTheDocument()
})
