import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, within } from "@testing-library/react"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import type { RequestItem } from "../../services/activity"
import { useUIStore } from "../../stores/ui-store"
import { formatCachedHit, formatCost, formatFirstToken, formatSpeed } from "./activity-format"
import { RequestTable } from "./request-table"

vi.mock("@wailsio/runtime", () => ({ Clipboard: { SetText: vi.fn() } }))

const item: RequestItem = {
  id: "d".repeat(32),
  startedAt: "2026-10-23T01:02:03Z",
  finishedAt: "2026-10-23T01:02:09Z",
  protocol: "responses",
  transport: "http",
  inputModel: "gpt-6-luna",
  model: "gpt-6-luna",
  stream: true,
  providerId: "builtin_openai",
  providerName: "OpenAI",
  accountId: "acct",
  accountName: "member",
  email: "member@example.invalid",
  workspaceId: "",
  accountDeleted: false,
  state: "success",
  statusCode: 200,
  errorCode: "",
  latencyMs: 4200,
  firstTokenMs: 200,
  inputTokens: "1000000",
  outputTokens: "300",
  inputKnown: true,
  outputKnown: true,
  usageComplete: true,
  cachedInputTokens: "400000",
  cachedInputKnown: true,
  costMin: 0.114,
  costMax: 0.114,
  costKnown: true,
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
}

beforeEach(() => {
  useUIStore.setState({ language: "zh" })
})

it("缓存命中按输入 Token 折算，未报告与真实零命中区分", () => {
  expect(formatCachedHit("400000", true, "1000000", true)).toBe("40%")
  expect(formatCachedHit("0", true, "1000000", true)).toBe("0%")
  expect(formatCachedHit("2500", true, "1000000", true)).toBe("0.3%")
  expect(formatCachedHit("", false, "1000000", true)).toBe("—")
  expect(formatCachedHit("500", true, "", false)).toBe("—")
})

it("成本按参考价给出金额，缓存未知时保留区间，未知价格保持破折号", () => {
  expect(formatCost(0.114, 0.114, true)).toBe("$0.114")
  expect(formatCost(0.06, 0.15, true)).toBe("$0.06–$0.15")
  expect(formatCost(1.5, 1.5, true)).toBe("$1.50")
  expect(formatCost(0.0000004, 0.0000004, true)).toBe("<$0.000001")
  expect(formatCost(0, 0, false)).toBe("—")
  expect(formatCost(Number.NaN, Number.NaN, true)).toBe("—")
})

it("速度扣除首字等待，未知输出与进行中的请求保持破折号", () => {
  const base = { outputTokens: "300", outputKnown: true, latencyMs: 4200, firstTokenMs: 200 }
  expect(formatSpeed({ ...base, state: "success" })).toBe("≈75 tok/s")
  expect(formatSpeed({ ...base, firstTokenMs: null, state: "success" })).toBe("≈71.4 tok/s")
  expect(formatSpeed({ ...base, outputKnown: false, state: "failed" })).toBe("—")
  expect(formatSpeed({ ...base, outputTokens: "0", state: "success" })).toBe("—")
  expect(formatSpeed({ ...base, state: "pending" })).toBe("—")
  expect(formatSpeed({ ...base, latencyMs: 0, state: "success" })).toBe("—")
})

it("首字只显示已采集的流式耗时", () => {
  expect(formatFirstToken(160)).toBe("160 ms")
  expect(formatFirstToken(0)).toBe("0 ms")
  expect(formatFirstToken(undefined)).toBe("—")
  expect(formatFirstToken(null)).toBe("—")
  expect(formatFirstToken(120, "pending")).toBe("—")
})

it("请求列表新增缓存命中、成本、速度与首字列并带说明提示", () => {
  mount([item])
  for (const column of ["缓存命中", "成本", "速度", "首字耗时"]) {
    expect(screen.getByText(column)).toBeInTheDocument()
    expect(screen.getByRole("button", { name: `${column}说明` })).toBeInTheDocument()
  }
  const row = screen.getAllByRole("row")[1]!
  expect(within(row).getByText("40%")).toBeInTheDocument()
  expect(within(row).getByText("$0.114")).toBeInTheDocument()
  expect(within(row).getByText("≈75 tok/s")).toBeInTheDocument()
  expect(within(row).getByText("200 ms")).toBeInTheDocument()
})

it("请求表横向滚动沿用统一细滑块样式，不再露出原生滚动条", () => {
  mount([item])
  const wrapper = screen.getByRole("table").parentElement
  expect(wrapper).toHaveClass("app-scroll-x")
  expect(wrapper).toHaveClass("overflow-x-auto")
})

it("缺少缓存、成本或输出明细的请求显示破折号而不是零", () => {
  mount([
    {
      ...item,
      state: "failed",
      outputKnown: false,
      outputTokens: "0",
      firstTokenMs: null,
      cachedInputTokens: "",
      cachedInputKnown: false,
      costMin: 0,
      costMax: 0,
      costKnown: false,
    },
  ])
  const row = screen.getAllByRole("row")[1]!
  expect(within(row).queryByText("0%")).not.toBeInTheDocument()
  expect(within(row).queryByText("$0.00")).not.toBeInTheDocument()
  expect(within(row).queryByText(/tok\/s/)).not.toBeInTheDocument()
  expect(within(row).getAllByText("—").length).toBeGreaterThanOrEqual(3)
})
