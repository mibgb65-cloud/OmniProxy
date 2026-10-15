import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { Dialog } from "radix-ui"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { AddAccountDialog } from "./add-account-dialog"
import { AccountEditForm } from "./account-edit-form"
import { MiMoBadge } from "./mimo-options"

vi.mock("../../services/omniproxy", () => ({ getAccountKeyPreview: vi.fn(async () => "••••••••") }))
const provider = {
  id: "builtin_mimo",
  kind: "mimo",
  source: "builtin",
  name: "Xiaomi MiMo",
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
  mimo: { region: "sgp", billingMode: "token_plan", version: 3 },
}
beforeEach(() => {
  useUIStore.setState({ language: "zh" })
})

async function setup() {
  const save = vi.fn(async () => {}),
    batch = vi.fn(async () => {}),
    user = userEvent.setup()
  render(
    <I18nProvider>
      <AddAccountDialog provider={provider} onCreate={save} onBatchCreate={batch} />
    </I18nProvider>,
  )
  await user.click(screen.getByRole("button", { name: "添加账号" }))
  return { save, batch, user }
}
async function choosePlan(user: ReturnType<typeof userEvent.setup>, label: string) {
  await user.click(screen.getByLabelText("计费方式"))
  await user.click(screen.getByRole("option", { name: "Token Plan" }))
  await user.click(screen.getByLabelText("套餐集群"))
  await user.click(screen.getByRole("option", { name: label }))
}

it("preserves pay-as-you-go as the default without adding a plan cluster", async () => {
  const { user, save } = await setup()
  expect(screen.getByLabelText("计费方式")).toHaveTextContent("普通按量 API")
  expect(screen.queryByLabelText("套餐集群")).not.toBeInTheDocument()
  await user.type(screen.getByLabelText("API Key"), "sk-placeholder-key")
  await user.click(screen.getByRole("button", { name: "保存账号" }))
  expect(save).toHaveBeenCalledWith({
    providerId: provider.id,
    name: "",
    apiKey: "sk-placeholder-key",
    mimo: { region: "", billingMode: "payg", version: 0 },
  })
})
it.each([
  ["中国", "cn"],
  ["新加坡", "sgp"],
  ["欧洲", "ams"],
])("submits the selected %s Token Plan endpoint", async (label, region) => {
  const { user, save } = await setup()
  await choosePlan(user, label!)
  expect(screen.getByText(`https://token-plan-${region}.xiaomimimo.com/v1`)).toBeInTheDocument()
  await user.type(screen.getByLabelText("API Key"), "tp-placeholder-key")
  await user.click(screen.getByRole("button", { name: "保存账号" }))
  expect(save).toHaveBeenCalledWith({
    providerId: provider.id,
    name: "",
    apiKey: "tp-placeholder-key",
    mimo: { region, billingMode: "token_plan", version: 0 },
  })
})
it("shares one selected cluster across personal and team keys in a batch", async () => {
  const { user, batch } = await setup()
  await choosePlan(user, "欧洲")
  await user.click(screen.getByRole("tab", { name: "批量添加" }))
  await user.type(screen.getByLabelText("API Keys"), "tp-placeholder-one\nttp-placeholder-two")
  await user.click(screen.getByRole("button", { name: "保存全部账号" }))
  expect(batch).toHaveBeenCalledWith({
    providerId: provider.id,
    apiKeys: ["tp-placeholder-one", "ttp-placeholder-two"],
    mimo: { region: "ams", billingMode: "token_plan", version: 0 },
  })
})
it("rejects ordinary keys for a Token Plan batch before calling the backend", async () => {
  const { user, batch, save } = await setup()
  await choosePlan(user, "中国")
  await user.click(screen.getByRole("tab", { name: "批量添加" }))
  await user.type(screen.getByLabelText("API Keys"), "tp-placeholder-one\nsk-placeholder-wrong")
  await user.click(screen.getByRole("button", { name: "保存全部账号" }))
  expect(batch).not.toHaveBeenCalled()
  expect(save).not.toHaveBeenCalled()
  expect(screen.getByLabelText("API Keys")).toHaveAttribute("aria-invalid", "true")
})
it("edits the cluster while retaining the protected key and optimistic version", async () => {
  const user = userEvent.setup(),
    save = vi.fn(async () => {}),
    client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <Dialog.Root open>
          <AccountEditForm account={account} onSubmit={save} onRestoreFocus={() => {}} />
        </Dialog.Root>
      </QueryClientProvider>
    </I18nProvider>,
  )
  expect(screen.getByLabelText("套餐集群")).toHaveTextContent("新加坡")
  expect(screen.getByLabelText("API Key")).toHaveValue("")
  await user.click(screen.getByLabelText("套餐集群"))
  await user.click(screen.getByRole("option", { name: "欧洲" }))
  await user.click(screen.getByRole("button", { name: "保存修改" }))
  expect(save).toHaveBeenCalledWith({
    id: account.id,
    name: account.name,
    apiKey: "",
    mimo: { region: "ams", billingMode: "token_plan", version: 3 },
  })
})
it("marks a legacy unconfirmed channel without guessing a cluster", () => {
  render(
    <I18nProvider>
      <MiMoBadge account={{ ...account, mimo: undefined }} />
    </I18nProvider>,
  )
  expect(screen.getByText("需确认 MiMo 通道")).toBeInTheDocument()
})
