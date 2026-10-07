import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import {
  getAccountKeyPreview,
  listAccounts,
  updateAccount,
  type AccountItem,
} from "../../services/omniproxy"
import { useUIStore } from "../../stores/ui-store"
import { EditAccountDialog } from "./edit-account-dialog"
import { AccountKeyPreview } from "./account-key-preview"
import { useAccountActions } from "./use-account-actions"
import { useAccounts } from "./use-accounts"

vi.mock("../../services/omniproxy", () => ({
  listAccounts: vi.fn(),
  createAccount: vi.fn(),
  createAccounts: vi.fn(),
  updateAccount: vi.fn(),
  setAccountEnabled: vi.fn(),
  deleteAccount: vi.fn(),
  getAccountKeyPreview: vi.fn(),
  copyAccountKey: vi.fn(),
}))
const original: AccountItem = {
  id: "acct_edit_test",
  providerId: "builtin_deepseek",
  name: "原名称",
  enabled: true,
  createdAt: "2026-10-07T00:00:00Z",
  updatedAt: "2026-10-07T00:00:00Z",
}
let saved: AccountItem
let preview: string

function Editor() {
  const actions = useAccountActions()
  const accounts = useAccounts()
  const account = accounts.data?.[0]
  return (
    <>
      {account ? (
        <>
          <button onClick={() => actions.editItem(account)}>编辑测试账号</button>
          <span>{account.name}</span>
          <AccountKeyPreview account={account} />
        </>
      ) : null}
      {actions.updateWarning ? <p role="alert">已保存但清理未完成</p> : null}
      <EditAccountDialog actions={actions} />
    </>
  )
}
async function openEditor() {
  const user = userEvent.setup()
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <Editor />
      </QueryClientProvider>
    </I18nProvider>,
  )
  const trigger = await screen.findByRole("button", { name: "编辑测试账号" })
  await user.click(trigger)
  return { user, client, trigger, dialog: screen.getByRole("dialog", { name: "编辑账号" }) }
}
describe("Account editor", () => {
  beforeEach(() => {
    vi.resetAllMocks()
    useUIStore.setState({ language: "zh" })
    saved = { ...original }
    preview = "sk-o••••••••1234"
    vi.mocked(listAccounts).mockImplementation(async () => [saved])
    vi.mocked(getAccountKeyPreview).mockImplementation(async () => preview)
    vi.mocked(updateAccount).mockImplementation(async (input) => {
      saved = { ...saved, name: input.name || "auto-key" }
      if (input.apiKey) preview = "sk-n••••••••5678"
      return { account: saved, cleanupWarning: false }
    })
  })
  it("prefills only the name and preserves the original key when renaming", async () => {
    const { user, dialog } = await openEditor()
    expect(within(dialog).getByLabelText("账号名称")).toHaveValue("原名称")
    expect(within(dialog).getByLabelText("API Key")).toHaveValue("")
    await user.clear(within(dialog).getByLabelText("账号名称"))
    await user.type(within(dialog).getByLabelText("账号名称"), "新名称")
    await user.click(within(dialog).getByRole("button", { name: "保存修改" }))
    expect(updateAccount).toHaveBeenCalledWith({ id: original.id, name: "新名称", apiKey: "" })
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
    expect(screen.getByText("新名称")).toBeInTheDocument()
  })
  it("submits a replacement key, refreshes its preview and clears mutation input", async () => {
    const { user, dialog, client } = await openEditor()
    await user.type(within(dialog).getByLabelText("API Key"), "new-test-protected-key")
    await user.click(within(dialog).getByRole("button", { name: "保存修改" }))
    expect(updateAccount).toHaveBeenCalledWith({
      id: original.id,
      name: "原名称",
      apiKey: "new-test-protected-key",
    })
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
    expect(await screen.findByText("sk-n••••••••5678")).toBeInTheDocument()
    expect(client.getQueryData(["account-key-preview", original.id])).toBe("sk-n••••••••5678")
    await waitFor(() =>
      expect(
        JSON.stringify(
          client
            .getMutationCache()
            .getAll()
            .map((m) => m.state.variables),
        ),
      ).not.toContain("new-test-protected-key"),
    )
  })
  it("cancels without updating and restores focus", async () => {
    const { user, dialog, trigger } = await openEditor()
    await user.type(within(dialog).getByLabelText("API Key"), "unsaved-test-key")
    await user.click(within(dialog).getByRole("button", { name: "取消" }))
    expect(updateAccount).not.toHaveBeenCalled()
    await waitFor(() => expect(trigger).toHaveFocus())
  })
  it("keeps failures open for retry and shows committed cleanup warnings", async () => {
    vi.mocked(updateAccount)
      .mockRejectedValueOnce(new Error("write failed"))
      .mockResolvedValueOnce({ account: original, cleanupWarning: true })
    const { user, dialog } = await openEditor()
    await user.click(within(dialog).getByRole("button", { name: "保存修改" }))
    expect(await within(dialog).findByRole("alert")).toHaveTextContent("保存失败")
    await user.click(within(dialog).getByRole("button", { name: "保存修改" }))
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
    expect(screen.getByRole("alert")).toHaveTextContent("已保存但清理未完成")
  })
  it("blocks repeated submit and cancellation while saving", async () => {
    vi.mocked(updateAccount).mockImplementationOnce(() => new Promise(() => {}))
    const { user, dialog } = await openEditor()
    await user.click(within(dialog).getByRole("button", { name: "保存修改" }))
    expect(within(dialog).getByLabelText("账号名称")).toBeDisabled()
    expect(within(dialog).getByLabelText("API Key")).toBeDisabled()
    expect(within(dialog).getByRole("button", { name: "取消" })).toBeDisabled()
    await user.keyboard("{Escape}")
    expect(dialog).toBeInTheDocument()
    expect(updateAccount).toHaveBeenCalledOnce()
  })
})
