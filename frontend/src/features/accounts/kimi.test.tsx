import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { Dialog } from "radix-ui"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { AddAccountDialog } from "./add-account-dialog"
import { AccountEditForm } from "./account-edit-form"
import { KimiPlatformBadge } from "./kimi-platform"

vi.mock("../../services/omniproxy", () => ({ getAccountKeyPreview: vi.fn(async () => "••••••••") }))
const provider = {
  id: "builtin_kimi",
  kind: "kimi",
  source: "builtin",
  name: "Moonshot AI",
  baseUrl: "",
  enabled: true,
  createdAt: "",
  updatedAt: "",
}
const account = {
  id: "kimi-account",
  providerId: provider.id,
  name: "Kimi",
  enabled: true,
  authMethod: "api_key",
  createdAt: "",
  updatedAt: "now",
  kimi: { region: "global", version: 3 },
}
beforeEach(() => {
  useUIStore.setState({ language: "zh" })
})

it("defaults new Kimi keys to China and submits the selected platform", async () => {
  const user = userEvent.setup(),
    save = vi.fn(async () => {}),
    batch = vi.fn(async () => {})
  render(
    <I18nProvider>
      <AddAccountDialog provider={provider} onCreate={save} onBatchCreate={batch} />
    </I18nProvider>,
  )
  await user.click(screen.getByRole("button", { name: "添加账号" }))
  expect(screen.getByLabelText("Kimi 平台")).toHaveTextContent("国内")
  await user.type(screen.getByLabelText("API Key"), "placeholder-kimi-key")
  await user.click(screen.getByRole("button", { name: "保存账号" }))
  expect(save).toHaveBeenCalledWith({
    providerId: provider.id,
    name: "",
    apiKey: "placeholder-kimi-key",
    kimi: { region: "cn", version: 0 },
  })
})

it("uses one platform for a batch of keys", async () => {
  const user = userEvent.setup(),
    save = vi.fn(async () => {}),
    batch = vi.fn(async () => {})
  render(
    <I18nProvider>
      <AddAccountDialog provider={provider} onCreate={save} onBatchCreate={batch} />
    </I18nProvider>,
  )
  await user.click(screen.getByRole("button", { name: "添加账号" }))
  await user.click(screen.getByLabelText("Kimi 平台"))
  await user.click(screen.getByRole("option", { name: "海外" }))
  await user.click(screen.getByRole("tab", { name: "批量添加" }))
  await user.type(screen.getByLabelText("API Keys"), "placeholder-one\nplaceholder-two")
  await user.click(screen.getByRole("button", { name: "保存全部账号" }))
  expect(batch).toHaveBeenCalledWith({
    providerId: provider.id,
    apiKeys: ["placeholder-one", "placeholder-two"],
    kimi: { region: "global", version: 0 },
  })
})

it("preserves an existing key and platform version when editing", async () => {
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
  expect(screen.getByLabelText("Kimi 平台")).toHaveTextContent("海外")
  await user.click(screen.getByLabelText("Kimi 平台"))
  await user.click(screen.getByRole("option", { name: "国内" }))
  await user.click(screen.getByRole("button", { name: "保存修改" }))
  expect(save).toHaveBeenCalledWith({
    id: account.id,
    name: "Kimi",
    apiKey: "",
    kimi: { region: "cn", version: 3 },
  })
})

it("labels a legacy unconfirmed platform without guessing the key", () => {
  render(
    <I18nProvider>
      <KimiPlatformBadge account={{ ...account, kimi: undefined }} />
    </I18nProvider>,
  )
  expect(screen.getByText("需确认平台")).toBeInTheDocument()
})
