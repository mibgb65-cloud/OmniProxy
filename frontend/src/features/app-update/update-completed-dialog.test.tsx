import { render, screen } from "@testing-library/react"
import { Dialog } from "radix-ui"
import { expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { UpdateCompletedDialog } from "./update-completed-dialog"

function renderDialog(
  from: string,
  version: string,
  handlers: { confirm?: () => void; notes?: () => void } = {},
) {
  return render(
    <I18nProvider>
      <Dialog.Root open>
        <UpdateCompletedDialog
          installed={{ version, from }}
          onConfirm={handlers.confirm ?? vi.fn()}
          onOpenNotes={handlers.notes ?? vi.fn()}
        />
      </Dialog.Root>
    </I18nProvider>,
  )
}

it("说明升级来源，并渲染内置的当前版本说明", async () => {
  const { container } = renderDialog("2.1.1-beta.1", "2.1.1-beta.2")
  expect(await screen.findByRole("dialog", { name: "更新完成" })).toBeInTheDocument()
  expect(
    screen.getByText("已从 2.1.1-beta.1 升级到 2.1.1-beta.2，以下为本版新功能。"),
  ).toBeInTheDocument()
  const summary = screen.getByRole("region", { name: "版本信息" })
  expect(summary).toHaveTextContent("2.1.1-beta.1")
  expect(summary).toHaveTextContent("2.1.1-beta.2")
  expect(screen.getByRole("heading", { name: "新特性 / Features" })).toBeInTheDocument()
  expect(screen.getByText("升级浮窗三段式布局").tagName).toBe("STRONG")
  // 顶部版本标题与 Markdown 记号都不应原样出现。
  const text = container.textContent ?? ""
  expect(text).not.toContain("# OmniProxy v2.1.1-beta.2")
  expect(text).not.toContain("## 新特性")
  expect(text).not.toContain("**升级浮窗三段式布局**")
})

it("不知道升级前版本时只说已更新到当前版本", async () => {
  renderDialog("", "2.1.1-beta.2")
  await screen.findByRole("dialog", { name: "更新完成" })
  expect(screen.getByText("已更新到 2.1.1-beta.2，以下为本版新功能。")).toBeInTheDocument()
  expect(screen.queryByRole("region", { name: "版本信息" })).not.toBeInTheDocument()
})

it("没有内置说明时给出明确提示而不是空白区域", async () => {
  renderDialog("2.1.1-beta.1", "9.9.9-local")
  await screen.findByRole("dialog", { name: "更新完成" })
  expect(
    screen.getByText("此版本未附带更新说明，可在「设置 → 关于」查看当前版本信息。"),
  ).toBeInTheDocument()
})
