import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, within } from "@testing-library/react"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import type { RequestItem } from "../../services/activity"
import { useUIStore } from "../../stores/ui-store"
import { RequestTable } from "./request-table"

vi.mock("@wailsio/runtime", () => ({ Clipboard: { SetText: vi.fn() } }))

function requestItem(id: string, state: RequestItem["state"], startedAt: string): RequestItem {
  return {
    id,
    startedAt,
    finishedAt: state === "pending" ? "" : startedAt,
    protocol: "chat",
    transport: "http",
    inputModel: "gemini-4-hydrogen-xhigh",
    model: "gemini-4-hydrogen-xhigh",
    effort: "xhigh",
    stream: true,
    providerId: "builtin_google",
    providerName: "Google",
    accountId: "acct",
    accountName: "member",
    email: "member@example.invalid",
    workspaceId: "",
    accountDeleted: false,
    state,
    statusCode: state === "pending" ? 0 : 200,
    errorCode: "",
    latencyMs: 1000,
    firstTokenMs: 0,
    inputTokens: "10",
    outputTokens: "2",
    inputKnown: true,
    outputKnown: true,
    usageComplete: state !== "pending",
    cachedInputTokens: "0",
    cachedInputKnown: false,
    costMin: 0,
    costMax: 0,
    costKnown: false,
  }
}

function mount(items: RequestItem[]) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <RequestTable items={items} onSelect={() => {}} />
      </QueryClientProvider>
    </I18nProvider>,
  )
  return screen.getAllByRole("row")
}

function rowIds(rows: HTMLElement[]) {
  return rows
    .map((row) => row.textContent ?? "")
    .map((text) => (text.includes("进行中") ? "pending" : text.includes("成功") ? "success" : ""))
}

beforeEach(() => {
  useUIStore.setState({ language: "zh" })
})

it("pins in-progress requests above finished ones with a divider", () => {
  const rows = mount([
    requestItem("a".repeat(32), "success", "2026-11-13T03:42:55Z"),
    requestItem("b".repeat(32), "pending", "2026-11-13T03:43:05Z"),
    requestItem("c".repeat(32), "success", "2026-11-13T03:42:20Z"),
  ])
  // 第 0 行是表头，第 1 行必须是进行中的记录，即使它不是最新的一条。
  expect(rowIds(rows)[1]).toBe("pending")
  const divider = screen.getByTestId("finished-divider")
  expect(divider).toHaveTextContent("已结束")
  const dividerIndex = rows.indexOf(divider)
  expect(dividerIndex).toBe(2)
  expect(rowIds(rows)[3]).toBe("success")
  expect(rowIds(rows)[4]).toBe("success")
  // 组内仍按接口给出的顺序，不重排（用记录 ID 前缀断言，避免依赖时区格式）。
  expect(within(rows[3]).getByText("aaaaaaaaaaaa", { exact: false })).toBeInTheDocument()
  expect(within(rows[4]).getByText("cccccccccccc", { exact: false })).toBeInTheDocument()
})

it("omits the divider when only finished requests are listed", () => {
  mount([requestItem("a".repeat(32), "success", "2026-11-13T03:42:55Z")])
  expect(screen.queryByTestId("finished-divider")).not.toBeInTheDocument()
})

it("omits the divider when everything is still running", () => {
  const rows = mount([requestItem("b".repeat(32), "pending", "2026-11-13T03:43:05Z")])
  expect(screen.queryByTestId("finished-divider")).not.toBeInTheDocument()
  expect(rowIds(rows)[1]).toBe("pending")
})
