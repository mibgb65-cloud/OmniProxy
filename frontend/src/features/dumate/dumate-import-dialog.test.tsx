import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import type { AccountItem } from "../../services/omniproxy"
import { importDuMateCookie, importDuMateDesktop } from "../../services/omniproxy"
import { DuMateImportDialog } from "./dumate-import-dialog"

vi.mock("../../services/omniproxy", async (original) => ({
  ...(await original<typeof import("../../services/omniproxy")>()),
  importDuMateCookie: vi.fn(),
  importDuMateDesktop: vi.fn(),
}))

const imported: AccountItem = {
  id: "acct_dumate",
  providerId: "builtin_dumate",
  name: "Tester",
  authMethod: "dumate_cookie",
  authState: "authenticated",
  enabled: true,
  createdAt: "2026-11-13T00:00:00Z",
  updatedAt: "2026-11-13T00:00:00Z",
}

function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <DuMateImportDialog providerId="builtin_dumate" />
      </QueryClientProvider>
    </I18nProvider>,
  )
}

describe("DuMate import dialog", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useUIStore.setState({ language: "zh" })
    vi.mocked(importDuMateDesktop).mockResolvedValue({
      account: imported,
      mode: "created",
      source: "desktop",
    })
    vi.mocked(importDuMateCookie).mockResolvedValue({
      account: { ...imported, name: "Cookie 账号" },
      mode: "updated",
      source: "manual",
    })
  })

  it("imports the desktop session and reports the new account", async () => {
    const user = userEvent.setup()
    setup()
    await user.click(screen.getByRole("button", { name: "导入 DuMate 账号" }))
    await user.click(screen.getByRole("button", { name: "导入本机桌面端登录态" }))
    expect(importDuMateDesktop).toHaveBeenCalledWith("builtin_dumate")
    expect(await screen.findByText(/已添加新账号：Tester/)).toBeInTheDocument()
  })

  it("submits pasted cookies only when present", async () => {
    const user = userEvent.setup()
    setup()
    await user.click(screen.getByRole("button", { name: "导入 DuMate 账号" }))
    const submit = screen.getByRole("button", { name: /^导入$/ })
    expect(submit).toBeDisabled()
    await user.type(
      screen.getByLabelText("粘贴浏览器 Cookie"),
      "bce-sessionid=abc; bce-user-info=csrf",
    )
    expect(submit).toBeEnabled()
    await user.click(submit)
    expect(importDuMateCookie).toHaveBeenCalledWith(
      "builtin_dumate",
      "bce-sessionid=abc; bce-user-info=csrf",
    )
    expect(await screen.findByText(/已更新同一账号的登录态/)).toBeInTheDocument()
  })

  it("localizes fixed import failures without echoing raw errors", async () => {
    vi.mocked(importDuMateDesktop).mockRejectedValue(new Error("dumate: desktop_login_missing"))
    const user = userEvent.setup()
    setup()
    await user.click(screen.getByRole("button", { name: "导入 DuMate 账号" }))
    await user.click(screen.getByRole("button", { name: "导入本机桌面端登录态" }))
    expect(await screen.findByRole("alert")).toHaveTextContent("未检测到本机百度搭子登录态")
    expect(screen.queryByText(/desktop_login_missing/)).not.toBeInTheDocument()
  })

  it("explains an identity change instead of overwriting another account", async () => {
    vi.mocked(importDuMateCookie).mockRejectedValue(new Error("dumate: identity_changed"))
    const user = userEvent.setup()
    setup()
    await user.click(screen.getByRole("button", { name: "导入 DuMate 账号" }))
    await user.type(screen.getByLabelText("粘贴浏览器 Cookie"), "bce-sessionid=other")
    await user.click(screen.getByRole("button", { name: /^导入$/ }))
    expect(await screen.findByRole("alert")).toHaveTextContent("该登录态属于另一个百度账号")
  })
})
