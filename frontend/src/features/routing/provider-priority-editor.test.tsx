import { render, screen, within, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { defaultMapping } from "./routing-test-helpers"
import { ProviderPriorityEditor } from "./provider-priority-editor"

it("保留失效服务商的位置，可移除并只提交剩余候选", async () => {
  useUIStore.setState({ language: "zh" })
  const onSave = vi.fn().mockResolvedValue(undefined)
  render(
    <I18nProvider>
      <ProviderPriorityEditor
        open
        item={{
          ...defaultMapping,
          providerOrder: [
            { id: "gone", name: "已删除服务商", available: false },
            { id: "backup_ds", name: "备用 DeepSeek", available: true },
          ],
        }}
        onSave={onSave}
        onClose={vi.fn()}
        onRestoreFocus={vi.fn()}
      />
    </I18nProvider>,
  )
  const user = userEvent.setup()
  const list = screen.getByRole("list", { name: "服务商使用顺序" })
  expect(within(list).getAllByRole("listitem")[0]).toHaveTextContent("当前不可用")
  await user.click(screen.getByRole("button", { name: "移除不可用服务商 已删除服务商" }))
  expect(within(list).getAllByRole("listitem")[0]).toHaveTextContent("备用 DeepSeek")
  await user.click(screen.getByRole("button", { name: "保存优先级" }))
  await waitFor(() => expect(onSave).toHaveBeenCalledWith(["backup_ds", "builtin_deepseek"]))
})

it("保存期间禁止重排、重复提交和 Escape 关闭，完成后正常关闭", async () => {
  useUIStore.setState({ language: "zh" })
  let finish!: () => void
  const onSave = vi.fn().mockImplementation(
    () =>
      new Promise<void>((resolve) => {
        finish = resolve
      }),
  )
  const onClose = vi.fn()
  render(
    <I18nProvider>
      <ProviderPriorityEditor
        open
        item={defaultMapping}
        onSave={onSave}
        onClose={onClose}
        onRestoreFocus={vi.fn()}
      />
    </I18nProvider>,
  )
  const user = userEvent.setup()
  await user.click(screen.getByRole("button", { name: "保存优先级" }))
  expect(screen.getByRole("button", { name: "正在保存…" })).toBeDisabled()
  expect(screen.getByRole("button", { name: "提高 备用 DeepSeek 的优先级" })).toBeDisabled()
  expect(screen.getByRole("button", { name: "取消" })).toBeDisabled()
  await user.keyboard("{Escape}")
  expect(onClose).not.toHaveBeenCalled()
  finish()
  await waitFor(() => expect(onClose).toHaveBeenCalledOnce())
  expect(onSave).toHaveBeenCalledOnce()
})
