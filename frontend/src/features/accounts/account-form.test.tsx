import { render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { AddAccountDialog } from "./add-account-dialog"

const provider = {
  id: "builtin_deepseek",
  kind: "deepseek",
  source: "builtin",
  name: "DeepSeek",
  baseUrl: "",
  enabled: true,
  createdAt: "",
  updatedAt: "",
}
const create = vi.fn(async () => {})
const createBatch = vi.fn(async () => {})

async function openForm() {
  const user = userEvent.setup()
  render(
    <I18nProvider>
      <AddAccountDialog provider={provider} onCreate={create} onBatchCreate={createBatch} />
    </I18nProvider>,
  )
  await user.click(screen.getByRole("button", { name: "添加账号" }))
  return user
}

describe("AccountForm", () => {
  beforeEach(() => {
    vi.resetAllMocks()
    create.mockResolvedValue(undefined)
    createBatch.mockResolvedValue(undefined)
    useUIStore.setState({ language: "zh" })
  })

  it("allows an empty name and leaves default naming to the backend", async () => {
    const user = await openForm()
    expect(screen.getByLabelText("账号名称")).toHaveFocus()
    await user.type(screen.getByLabelText("API Key"), "  12345678-test-key  ")
    await user.click(screen.getByRole("button", { name: "保存账号" }))
    expect(create).toHaveBeenCalledWith({
      providerId: provider.id,
      name: "",
      apiKey: "12345678-test-key",
    })
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
  })

  it("hides names in batch mode and submits all lines in one request", async () => {
    const user = await openForm()
    await user.type(screen.getByLabelText("账号名称"), "单个账号名称")
    await user.click(screen.getByRole("tab", { name: "批量添加" }))
    expect(screen.queryByLabelText("账号名称")).not.toBeInTheDocument()
    await user.click(screen.getByLabelText("API Keys"))
    await user.paste("  first-test-key\r\n\r\nsecond-test-key  \n")
    expect(screen.getByRole("status")).toHaveTextContent("将添加 2 个账号")
    await user.click(screen.getByRole("button", { name: "保存全部账号" }))
    expect(createBatch).toHaveBeenCalledOnce()
    expect(createBatch).toHaveBeenCalledWith({
      providerId: provider.id,
      apiKeys: ["first-test-key", "second-test-key"],
    })
    expect(create).not.toHaveBeenCalled()
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
  })

  it.each([
    "",
    "valid-test-key\nbad\tkey",
    Array.from({ length: 101 }, () => "test-only-key").join("\n"),
  ])("rejects an invalid batch before saving", async (text) => {
    const user = await openForm()
    await user.click(screen.getByRole("tab", { name: "批量添加" }))
    if (text) {
      await user.click(screen.getByLabelText("API Keys"))
      await user.paste(text)
    }
    await user.click(screen.getByRole("button", { name: "保存全部账号" }))
    expect(screen.getByText(/请输入 1–100 个有效 API Key/)).toBeInTheDocument()
    expect(createBatch).not.toHaveBeenCalled()
  })

  it("retains a failed batch so it can be retried without a partial client save", async () => {
    createBatch.mockRejectedValueOnce(new Error("storage unavailable"))
    const user = await openForm()
    await user.click(screen.getByRole("tab", { name: "批量添加" }))
    await user.click(screen.getByLabelText("API Keys"))
    await user.paste("first-test-key\nsecond-test-key")
    await user.click(screen.getByRole("button", { name: "保存全部账号" }))
    expect(await screen.findByRole("alert")).toHaveTextContent("保存失败")
    expect(screen.getByLabelText("API Keys")).toHaveValue("first-test-key\nsecond-test-key")
    await user.click(screen.getByRole("button", { name: "保存全部账号" }))
    expect(createBatch).toHaveBeenCalledTimes(2)
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
  })

  it("locks mode changes and cancellation while a batch is being saved", async () => {
    createBatch.mockImplementationOnce(() => new Promise(() => {}))
    const user = await openForm()
    await user.click(screen.getByRole("tab", { name: "批量添加" }))
    await user.click(screen.getByLabelText("API Keys"))
    await user.paste("first-test-key\nsecond-test-key")
    await user.click(screen.getByRole("button", { name: "保存全部账号" }))
    const dialog = screen.getByRole("dialog")
    expect(within(dialog).getByRole("button", { name: "正在保存…" })).toBeDisabled()
    expect(screen.getByRole("tab", { name: "单个添加" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "取消" })).toBeDisabled()
    await user.keyboard("{Escape}")
    expect(dialog).toBeInTheDocument()
    expect(createBatch).toHaveBeenCalledOnce()
  })
})
