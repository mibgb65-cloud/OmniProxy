import { act, isInaccessible, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it } from "vitest"
import { renderPreferences, resetPreferences } from "./preferences-test-helpers"
import {
  checkForUpdate,
  getRuntimeInfo,
  getStorageInfo,
  setDataDir,
} from "../../services/omniproxy"
import { setRetention } from "../../services/activity"
import { useUIStore } from "../../stores/ui-store"
import { useAppPreferences } from "./app-preferences-state"

beforeEach(resetPreferences)

it("侧栏主导航移除两页入口，合并按钮位于折叠按钮之后，关闭时不加载设置", async () => {
  renderPreferences()
  const trigger = await screen.findByRole("button", { name: "设置与关于" })
  const collapse = screen.getByRole("button", { name: "折叠" })
  expect(collapse.compareDocumentPosition(trigger) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  expect(screen.queryByRole("link", { name: "设置" })).not.toBeInTheDocument()
  expect(screen.queryByRole("link", { name: "关于" })).not.toBeInTheDocument()
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
  expect(getStorageInfo).not.toHaveBeenCalled()
  expect(getRuntimeInfo).not.toHaveBeenCalled()
})

it("打开窗口保留工作页面，限制焦点，Escape 关闭后返回原入口", async () => {
  const user = userEvent.setup()
  const { router } = renderPreferences()
  const draft = await screen.findByRole("textbox", { name: "工作草稿" })
  await user.type(draft, "继续编辑")
  const href = router.history.location.href
  const preferences = localStorage.getItem("omniproxy-ui-state")
  const trigger = screen.getByRole("button", { name: "设置与关于" })
  await user.click(trigger)
  const dialog = await screen.findByRole("dialog", { name: "设置与关于" })
  await waitFor(() => expect(within(dialog).getByRole("tab", { name: "通用设置" })).toHaveFocus())
  expect(isInaccessible(draft)).toBe(true)
  expect(router.history.location.href).toBe(href)
  for (let index = 0; index < 12; index++) {
    await user.tab()
    expect(dialog).toContainElement(document.activeElement as HTMLElement)
  }
  await user.keyboard("{Escape}")
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
  expect(trigger).toHaveFocus()
  expect(draft).toHaveValue("原工作继续编辑")
  expect(router.history.location.href).toBe(href)
  expect(localStorage.getItem("omniproxy-ui-state")).toBe(preferences)
  expect(useAppPreferences.getState().open).toBe(false)
})

it("页签切换保留填写内容，关于信息按需读取且不自动检查更新", async () => {
  const user = userEvent.setup()
  renderPreferences()
  await user.click(await screen.findByRole("button", { name: "设置与关于" }))
  expect(screen.queryByRole("textbox", { name: "新数据目录" })).not.toBeInTheDocument()
  expect(getStorageInfo).not.toHaveBeenCalled()
  expect(getRuntimeInfo).not.toHaveBeenCalled()
  await user.click(screen.getByRole("tab", { name: "数据存储" }))
  const path = await screen.findByRole("textbox", { name: "新数据目录" })
  await user.clear(path)
  await user.type(path, "D:/draft-data")
  await user.click(screen.getByRole("tab", { name: "请求与日志" }))
  const retention = await screen.findByRole("spinbutton", { name: "请求明细上限" })
  await waitFor(() => expect(retention).toBeEnabled())
  await user.clear(retention)
  await user.type(retention, "20000")
  await user.click(screen.getByRole("tab", { name: "关于" }))
  expect(await screen.findByText("go1.26.2")).toBeInTheDocument()
  expect(checkForUpdate).not.toHaveBeenCalled()
  await user.click(screen.getByRole("tab", { name: "请求与日志" }))
  expect(retention).toHaveValue(20000)
  expect(screen.getAllByRole("tabpanel")).toHaveLength(1)
  expect(screen.queryByRole("textbox", { name: "新数据目录" })).not.toBeInTheDocument()
  await user.click(screen.getByRole("tab", { name: "数据存储" }))
  expect(path).toHaveValue("D:/draft-data")
  expect(setDataDir).not.toHaveBeenCalled()
  expect(setRetention).not.toHaveBeenCalled()
})

it("设置仍通过原服务保存，关闭按钮恢复焦点", async () => {
  const user = userEvent.setup()
  renderPreferences()
  const trigger = await screen.findByRole("button", { name: "设置与关于" })
  await user.click(trigger)
  await user.click(screen.getByRole("tab", { name: "请求与日志" }))
  const field = await screen.findByRole("spinbutton", { name: "请求明细上限" })
  await waitFor(() => expect(field).toBeEnabled())
  await user.clear(field)
  await user.type(field, "20000")
  await user.click(screen.getByRole("button", { name: "保存设置" }))
  await waitFor(() => expect(setRetention).toHaveBeenCalledExactlyOnceWith(20000))
  expect(await screen.findByText("设置已保存")).toHaveAttribute("role", "status")
  await user.click(screen.getByRole("button", { name: "关闭设置窗口" }))
  await waitFor(() => expect(trigger).toHaveFocus())
})

it("收起侧栏仍可打开，窗口与页签随语言切换", async () => {
  useUIStore.setState({ sidebarCollapsed: true })
  const user = userEvent.setup()
  renderPreferences()
  await user.click(await screen.findByRole("button", { name: "设置与关于" }))
  expect(await screen.findByRole("dialog", { name: "设置与关于" })).toBeInTheDocument()
  act(() => useUIStore.getState().setLanguage("en"))
  expect(await screen.findByRole("dialog", { name: "Settings & About" })).toBeInTheDocument()
  expect(screen.getByRole("tab", { name: "General" })).toBeInTheDocument()
  expect(screen.getByRole("tab", { name: "Requests & logs" })).toBeInTheDocument()
  expect(screen.getByRole("tab", { name: "Data storage" })).toBeInTheDocument()
  expect(screen.getByRole("tab", { name: "About" })).toBeInTheDocument()
  await user.click(screen.getByRole("button", { name: "Close settings window" }))
  await waitFor(() =>
    expect(screen.getByRole("button", { name: "Settings & About" })).toHaveFocus(),
  )
})

it.each([
  ["/settings", "通用设置"],
  ["/about", "关于"],
])("旧链接 %s 打开对应页签并兼容关闭", async (entry, label) => {
  const user = userEvent.setup()
  const { router } = renderPreferences(entry)
  expect(await screen.findByRole("dialog", { name: "设置与关于" })).toBeInTheDocument()
  expect(screen.getByRole("tab", { name: label })).toHaveAttribute("aria-selected", "true")
  await waitFor(() => expect(router.history.location.pathname).toBe("/"))
  await user.keyboard("{Escape}")
  await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
  expect(screen.getByRole("button", { name: "设置与关于" })).toHaveFocus()
})

it("分类支持上下键导航，仅显示当前面板，切换后关闭可重新打开当前分类", async () => {
  const user = userEvent.setup()
  renderPreferences()
  const trigger = await screen.findByRole("button", { name: "设置与关于" })
  await user.click(trigger)
  const general = screen.getByRole("tab", { name: "通用设置" })
  await waitFor(() => expect(general).toHaveFocus())
  expect(screen.getByRole("tablist")).toHaveAttribute("aria-orientation", "vertical")
  expect(screen.getByRole("slider", { name: "全局文字大小" })).toBeInTheDocument()
  expect(screen.getByRole("switch", { name: /开机自启动/ })).toBeInTheDocument()
  await user.keyboard("{ArrowDown}")
  expect(screen.getByRole("tab", { name: "请求与日志" })).toHaveFocus()
  expect(screen.getAllByRole("tabpanel")).toHaveLength(1)
  expect(screen.queryByRole("slider")).not.toBeInTheDocument()
  expect(await screen.findByRole("spinbutton", { name: "请求明细上限" })).toBeInTheDocument()
  await user.keyboard("{End}")
  expect(screen.getByRole("tab", { name: "关于" })).toHaveAttribute("aria-selected", "true")
  await user.keyboard("{Escape}")
  await user.click(trigger)
  await waitFor(() => expect(screen.getByRole("tab", { name: "关于" })).toHaveFocus())
})
