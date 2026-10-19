import { act, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { info, preferences, renderPage, resetUpdatePage } from "./update-page-test-helpers"
import { useUIStore } from "../../stores/ui-store"
import { checkForUpdate, getRuntimeInfo } from "../../services/omniproxy"
import {
  getUpdatePreferences,
  setAutomaticUpdateCheck,
  setAutomaticUpdateDownload,
} from "../../services/updates"

vi.mock("../../services/omniproxy", () => ({
  getRuntimeInfo: vi.fn(),
  checkForUpdate: vi.fn(),
}))
vi.mock("../../services/updates", () => ({
  getApplicationUpdate: vi.fn(),
  getUpdatePreferences: vi.fn(),
  setAutomaticUpdateCheck: vi.fn(),
  setAutomaticUpdateDownload: vi.fn(),
  setUpdateChannel: vi.fn(),
  startApplicationUpdate: vi.fn(),
  cancelApplicationUpdate: vi.fn(),
  installApplicationUpdate: vi.fn(),
}))

beforeEach(resetUpdatePage)

it("分组展示运行版本和本地日志，打开页面不联网检查或下载", async () => {
  renderPage()
  expect(await screen.findByText("Beta 渠道")).toBeInTheDocument()
  const notes = await screen.findByRole("region", { name: "当前版本更新日志" })
  expect(within(notes).getByText(info.version)).toBeInTheDocument()
  expect(within(notes).getByRole("heading", { name: "登录修复" })).toBeInTheDocument()
  expect(within(notes).getByRole("heading", { name: "日志修复" })).toBeInTheDocument()
  expect(within(notes).queryByText("5h／一周额度耗尽后轮换")).not.toBeInTheDocument()
  expect(screen.getAllByRole("button", { name: "检查更新" })).toHaveLength(1)
  expect(screen.getByRole("combobox", { name: "更新渠道" })).toBeEnabled()
  expect(checkForUpdate).not.toHaveBeenCalled()
})

it("运行版本徽标不跟随更新候选渠道误变", async () => {
  vi.mocked(getUpdatePreferences).mockResolvedValue({ ...preferences, channel: "stable" })
  renderPage()
  expect(await screen.findByText("Beta 渠道")).toBeInTheDocument()
  expect(screen.getByRole("combobox", { name: "更新渠道" })).toHaveTextContent("正式版")
})

it("未知版本明确提示没有日志，不展示其他版本说明", async () => {
  vi.mocked(getRuntimeInfo).mockResolvedValue({ ...info, version: "9.9.9" })
  renderPage()
  const notes = await screen.findByRole("region", { name: "当前版本更新日志" })
  expect(await within(notes).findByText("此版本尚未附带更新日志。")).toBeInTheDocument()
  expect(within(notes).queryByRole("heading", { name: "登录修复" })).not.toBeInTheDocument()
  expect(screen.getByText("正式渠道")).toBeInTheDocument()
})

it("空格键切换自动检查并持久保存，不触发手动检查", async () => {
  const user = userEvent.setup()
  renderPage()
  const toggle = await screen.findByRole("switch", { name: "启动后自动检查更新" })
  await waitFor(() => expect(toggle).toBeEnabled())
  expect(toggle).toBeChecked()
  toggle.focus()
  await user.keyboard(" ")
  await waitFor(() =>
    expect(setAutomaticUpdateCheck).toHaveBeenCalledWith(false, expect.anything()),
  )
  await waitFor(() => expect(toggle).not.toBeChecked())
  expect(checkForUpdate).not.toHaveBeenCalled()
})

it("自动检查保存失败时反馈错误并保留原值", async () => {
  const user = userEvent.setup()
  vi.mocked(setAutomaticUpdateCheck).mockRejectedValue(new Error("save failed"))
  renderPage()
  const toggle = await screen.findByRole("switch", { name: "启动后自动检查更新" })
  await waitFor(() => expect(toggle).toBeEnabled())
  await user.click(toggle)
  expect(await screen.findByRole("alert")).toHaveTextContent("无法读取或保存更新设置，请重试。")
  expect(toggle).toBeChecked()
})

it("设置加载失败时禁止修改及检查，日志仍可离线查看", async () => {
  vi.mocked(getUpdatePreferences).mockRejectedValue(new Error("read failed"))
  renderPage()
  expect(await screen.findByRole("alert")).toBeInTheDocument()
  expect(screen.getByRole("switch", { name: "启动后自动检查更新" })).toBeDisabled()
  expect(screen.getByRole("combobox", { name: "更新渠道" })).toBeDisabled()
  expect(screen.getByRole("button", { name: "检查更新" })).toBeDisabled()
  expect(screen.getByRole("heading", { name: "登录修复" })).toBeInTheDocument()
})

it("英文界面翻译控件及日志区域，保留真实发行说明原文", async () => {
  useUIStore.setState({ language: "en" })
  renderPage()
  const notes = await screen.findByRole("region", { name: "Current version changelog" })
  expect(await within(notes).findByRole("heading", { name: "登录修复" })).toBeInTheDocument()
  expect(
    screen.getByRole("switch", {
      name: "Check automatically after startup",
    }),
  ).toBeChecked()
  expect(screen.getByRole("button", { name: "Check for updates" })).toBeInTheDocument()
})

it("检查进行中显示忙状态并禁止并发修改", async () => {
  const user = userEvent.setup()
  let finish: ((value: Awaited<ReturnType<typeof checkForUpdate>>) => void) | undefined
  vi.mocked(checkForUpdate).mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve
      }),
  )
  renderPage()
  const button = await screen.findByRole("button", { name: "检查更新" })
  await waitFor(() => expect(button).toBeEnabled())
  await user.click(button)
  expect(screen.getByRole("button", { name: "检查中…" })).toBeDisabled()
  expect(screen.getByRole("switch", { name: "启动后自动检查更新" })).toBeDisabled()
  expect(screen.getByRole("combobox", { name: "更新渠道" })).toBeDisabled()
  await act(async () =>
    finish?.({
      state: "up-to-date",
      currentVersion: info.version,
      latestVersion: info.version,
      releaseUrl: "",
      publishedAt: "",
      message: "",
    }),
  )
  expect(await screen.findByText("已是最新版本。")).toBeInTheDocument()
})

it("版本信息读取失败明确说明原因，不混用检查快照中的旧版本日志", async () => {
  vi.mocked(getRuntimeInfo).mockRejectedValue(new Error("unavailable"))
  renderPage()
  const notes = await screen.findByRole("region", { name: "当前版本更新日志" })
  expect(await within(notes).findByRole("alert")).toHaveTextContent(
    "版本信息加载失败，暂时无法匹配更新日志。",
  )
  expect(
    within(notes).queryByRole("heading", { name: "5h／一周额度耗尽后轮换" }),
  ).not.toBeInTheDocument()
})

it("自动下载默认关闭，开启后只保存选择，不立即联网或安装", async () => {
  const user = userEvent.setup()
  renderPage()
  const toggle = await screen.findByRole("switch", { name: "发现新版本后自动下载更新" })
  await waitFor(() => expect(toggle).toBeEnabled())
  expect(toggle).not.toBeChecked()
  await user.click(toggle)
  await waitFor(() =>
    expect(setAutomaticUpdateDownload).toHaveBeenCalledWith(true, expect.anything()),
  )
  await waitFor(() => expect(toggle).toBeChecked())
  expect(screen.getByText(/安装与重启仍需你确认/)).toBeInTheDocument()
  expect(checkForUpdate).not.toHaveBeenCalled()
})

it("自动下载保存失败反馈错误并保持关闭", async () => {
  const user = userEvent.setup()
  vi.mocked(setAutomaticUpdateDownload).mockRejectedValue(new Error("save failed"))
  renderPage()
  const toggle = await screen.findByRole("switch", { name: "发现新版本后自动下载更新" })
  await waitFor(() => expect(toggle).toBeEnabled())
  await user.click(toggle)
  expect(await screen.findByRole("alert")).toHaveTextContent("无法读取或保存更新设置，请重试。")
  expect(toggle).not.toBeChecked()
})

it("自动下载保存中禁止并发修改，成功后恢复操作", async () => {
  const user = userEvent.setup()
  let finish: ((value: Awaited<ReturnType<typeof setAutomaticUpdateDownload>>) => void) | undefined
  vi.mocked(setAutomaticUpdateDownload).mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve
      }),
  )
  renderPage()
  const toggle = await screen.findByRole("switch", { name: "发现新版本后自动下载更新" })
  await waitFor(() => expect(toggle).toBeEnabled())
  await user.click(toggle)
  expect(toggle).toBeDisabled()
  expect(screen.getByRole("switch", { name: "启动后自动检查更新" })).toBeDisabled()
  expect(screen.getByRole("combobox", { name: "更新渠道" })).toBeDisabled()
  expect(screen.getByRole("button", { name: "检查更新" })).toBeDisabled()
  await act(async () => finish?.({ ...preferences, autoDownload: true }))
  await waitFor(() => expect(toggle).toBeEnabled())
})
