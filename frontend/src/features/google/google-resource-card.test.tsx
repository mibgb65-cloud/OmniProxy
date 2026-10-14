import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { setGoogleProject, type GoogleResource } from "../../services/google"
import type { AccountItem } from "../../services/omniproxy"
import { GoogleResourceCard } from "./google-resource-card"

vi.mock("../../services/google", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../services/google")>()),
  setGoogleProject: vi.fn(),
}))
const accounts: AccountItem[] = ["a", "b"].map((id) => ({
  id,
  name: id,
  providerId: "builtin_google",
  enabled: true,
  authMethod: "api_key",
  createdAt: "now",
  updatedAt: "version-1",
}))
const resource: GoogleResource = {
  id: "project:shared-project",
  projectId: "shared-project",
  accountIds: ["a", "b"],
  verified: false,
  officialRemaining: null,
  observations: [
    {
      model: "gemini-test",
      category: "daily_exhausted",
      metric: "generativelanguage.googleapis.com/request_count",
      limit: 10,
      retryAt: "2020-01-01T00:00:00Z",
      observedAt: "2020-01-01T00:00:00Z",
      stale: true,
    },
  ],
}
function show() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <GoogleResourceCard resource={resource} accounts={accounts} />
      </QueryClientProvider>
    </I18nProvider>,
  )
}

describe("Gemini 项目资源", () => {
  it("同项目多个 Key 显示一个未知资源池，过期记录不能推算恢复", () => {
    show()
    expect(screen.getAllByText("未知（未连接官方额度查询）")).toHaveLength(1)
    expect(
      screen.getByText("用户声明的项目，尚未验证；组内 Key 共享项目额度。"),
    ).toBeInTheDocument()
    expect(screen.getByText("此记录已过期，仅供参考，未确认当前额度恢复。")).toBeInTheDocument()
    expect(screen.getAllByRole("button", { name: /绑定项目：/ })).toHaveLength(2)
  })
  it("绑定时带上账号版本，错误保留弹窗并允许重试", async () => {
    const user = userEvent.setup()
    vi.mocked(setGoogleProject)
      .mockRejectedValueOnce(new Error("google: account_changed"))
      .mockResolvedValueOnce(undefined)
    show()
    await user.click(screen.getByRole("button", { name: "绑定项目：a" }))
    const input = screen.getByRole("textbox", { name: "项目 ID" })
    await user.clear(input)
    await user.type(input, "new-project")
    await user.click(screen.getByRole("button", { name: "保存" }))
    expect(await screen.findByRole("alert")).toHaveTextContent("账号已发生变化")
    expect(setGoogleProject).toHaveBeenLastCalledWith({
      accountId: "a",
      accountUpdatedAt: "version-1",
      projectId: "new-project",
    })
    await user.click(screen.getByRole("button", { name: "保存" }))
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
  })
})
