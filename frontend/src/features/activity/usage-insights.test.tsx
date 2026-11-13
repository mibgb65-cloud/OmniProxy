import { render, screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import type {
  Breakdown,
  CacheUsage,
  HourUsage,
  ModelDay,
  Reliability,
  Summary,
} from "../../services/activity"
import { usageDateKey } from "./usage-model"
import { UsageKpis } from "./usage-kpis"
import { UsageDailyChart } from "./usage-daily"
import { UsageComposition } from "./usage-composition"
import { UsageHours } from "./usage-hours"
import { UsageModelTrend } from "./usage-model-trend"
import { UsageQuality } from "./usage-quality"
import { AccountShareCard, ModelShareCard, ProviderShareCard } from "./usage-share"

function total(input: number, output: number, requests: number, complete = requests): Summary {
  return {
    inputTokens: String(input),
    outputTokens: String(output),
    totalTokens: String(input + output),
    requests,
    completeUsage: complete,
  }
}

function breakdown(key: string, name: string, summary: Summary): Breakdown {
  return { key, name, email: "", workspaceId: "", summary }
}

function today(offset = 0) {
  const date = new Date()
  date.setDate(date.getDate() + offset)
  return usageDateKey(date)
}

const summary = total(1000, 250, 10, 8)
const previous = total(500, 100, 5, 4)
const cache: CacheUsage = { cachedInputTokens: "400", knownRequests: 6 }
const reliability: Reliability = {
  requests: 10,
  success: 8,
  failed: 1,
  cancelled: 1,
  successRate: 88.888888,
  averageLatencyMs: 1500,
  averageFirstTokenMs: 400,
  firstTokenSamples: 7,
}

beforeEach(() => {
  useUIStore.setState({ language: "zh" })
})

describe("usage KPIs", () => {
  it("shows lifetime, recent comparison, average, cache lower bound and quality", () => {
    render(
      <I18nProvider>
        <UsageKpis
          allTime={summary}
          previous={previous}
          recent={summary}
          cache={cache}
          reliability={reliability}
        />
      </I18nProvider>,
    )
    const lifetime = screen.getByRole("group", { name: "累计代理 Token" })
    expect(within(lifetime).getByText("1,250")).toBeInTheDocument()
    expect(within(lifetime).getByText("输入 1,000 · 输出 250")).toBeInTheDocument()
    expect(screen.getByText("完整用量 8 / 10 请求")).toBeInTheDocument()
    expect(screen.getByText("108.3%")).toBeInTheDocument()
    expect(screen.getByText("125")).toBeInTheDocument()
    // 只有 6 / 10 个请求报告了缓存，命中率必须显示为下界。
    expect(screen.getByText("≥ 40%")).toBeInTheDocument()
    expect(screen.getByText("6 个请求报告了缓存明细", { exact: false })).toBeInTheDocument()
    expect(screen.getByText("88.9%")).toBeInTheDocument()
  })

  it("does not fake a cache rate or success rate without samples", () => {
    render(
      <I18nProvider>
        <UsageKpis
          allTime={total(100, 20, 2)}
          previous={total(0, 0, 0)}
          recent={total(100, 20, 2)}
          cache={{ cachedInputTokens: "0", knownRequests: 0 }}
          reliability={{
            ...reliability,
            requests: 0,
            success: 0,
            failed: 0,
            cancelled: 0,
            successRate: null,
          }}
        />
      </I18nProvider>,
    )
    expect(screen.getAllByText("—").length).toBeGreaterThan(0)
    expect(screen.getByText("还没有请求报告缓存明细")).toBeInTheDocument()
  })
})

describe("usage daily chart", () => {
  it("switches metric and range without losing recorded days", async () => {
    const user = userEvent.setup()
    const days = [breakdown(today(), today(), total(30, 10, 3))]
    render(
      <I18nProvider>
        <UsageDailyChart days={days} />
      </I18nProvider>,
    )
    const region = screen.getByRole("region", { name: "每日用量" })
    expect(within(region).getAllByLabelText(/^\d{4}-\d{2}-\d{2} ·/)).toHaveLength(30)
    expect(
      within(region).getByLabelText(new RegExp(`^${today()} · 合计 Token 40`)),
    ).toBeInTheDocument()
    await user.click(within(region).getByRole("button", { name: "请求数" }))
    await user.click(within(region).getByRole("button", { name: "7 天" }))
    expect(within(region).getAllByLabelText(/^\d{4}-\d{2}-\d{2} ·/)).toHaveLength(7)
    expect(
      within(region).getByLabelText(new RegExp(`^${today()} · .*请求数 3`)),
    ).toBeInTheDocument()
  })
})

describe("usage structure cards", () => {
  it("shows token composition percentages", () => {
    render(
      <I18nProvider>
        <UsageComposition summary={summary} />
      </I18nProvider>,
    )
    const region = screen.getByRole("region", { name: "Token 构成" })
    expect(within(region).getByText("80%")).toBeInTheDocument()
    expect(within(region).getByText("20%")).toBeInTheDocument()
    expect(within(region).getByText("1,000")).toBeInTheDocument()
    expect(within(region).getByText("250")).toBeInTheDocument()
  })

  it("folds models beyond the top six into other", () => {
    const models = Array.from({ length: 9 }, (_, index) =>
      breakdown(`model-${index}`, `model-${index}`, total(100 - index, 0, 1)),
    )
    render(
      <I18nProvider>
        <ModelShareCard items={models} total={total(1000, 0, 9)} />
      </I18nProvider>,
    )
    const region = screen.getByRole("region", { name: "模型用量占比" })
    expect(within(region).getByText("model-0")).toBeInTheDocument()
    expect(within(region).getByText("model-5")).toBeInTheDocument()
    expect(within(region).queryByText("model-6")).not.toBeInTheDocument()
    expect(within(region).getByText("其他 3 个模型")).toBeInTheDocument()
  })

  it("marks providers with unconfirmed historical attribution", () => {
    render(
      <I18nProvider>
        <ProviderShareCard
          items={[breakdown("legacy:Old", "", total(50, 0, 1))]}
          total={total(50, 0, 1)}
        />
      </I18nProvider>,
    )
    const region = screen.getByRole("region", { name: "提供商用量占比" })
    expect(within(region).getByText("Old")).toBeInTheDocument()
    expect(within(region).getByText("历史归属")).toBeInTheDocument()
    expect(within(region).getByText("100%")).toBeInTheDocument()
  })

  it("ranks account members with input/output split", () => {
    const accounts = [
      breakdown("a", "a@example.invalid", total(80, 20, 4)),
      breakdown("b", "b@example.invalid", total(10, 10, 2)),
    ]
    render(
      <I18nProvider>
        <AccountShareCard items={accounts} total={total(120, 0, 6)} />
      </I18nProvider>,
    )
    const region = screen.getByRole("region", { name: "按账号成员" })
    expect(within(region).getByText("a@example.invalid")).toBeInTheDocument()
    expect(within(region).getByText("83.3%", { exact: false })).toBeInTheDocument()
    expect(within(region).getByText("100")).toBeInTheDocument()
  })
})

describe("usage time and quality", () => {
  it("stacks daily usage by model and keeps the remainder as other", () => {
    const key = today()
    const days = [breakdown(key, key, total(170, 30, 4))]
    const modelDays: ModelDay[] = [
      { day: key, model: "model-a", summary: total(120, 0, 2) },
      { day: key, model: "model-b", summary: total(50, 0, 1) },
    ]
    render(
      <I18nProvider>
        <UsageModelTrend modelDays={modelDays} days={days} />
      </I18nProvider>,
    )
    const region = screen.getByRole("region", { name: "模型每日用量" })
    expect(within(region).getByText("model-a")).toBeInTheDocument()
    expect(within(region).getByText("model-b")).toBeInTheDocument()
    expect(within(region).getByText("其他")).toBeInTheDocument()
    expect(
      within(region).getByLabelText(new RegExp(`^${key} · 合计 Token 200`)),
    ).toBeInTheDocument()
  })

  it("shows the peak local hour and all 24 slots", () => {
    const hours: HourUsage[] = Array.from({ length: 24 }, (_, hour) => ({
      hour,
      summary: total(hour === 9 ? 40 : 0, hour === 9 ? 10 : 0, hour === 9 ? 4 : 0),
    }))
    render(
      <I18nProvider>
        <UsageHours hours={hours} />
      </I18nProvider>,
    )
    const region = screen.getByRole("region", { name: "请求时段分布" })
    expect(within(region).getByText("高峰 09 时 · 4 请求")).toBeInTheDocument()
    expect(within(region).getAllByLabelText(/^\d{2} 时 · 请求数/)).toHaveLength(24)
  })

  it("shows success rate and duration averages only when samples exist", () => {
    render(
      <I18nProvider>
        <UsageQuality reliability={reliability} />
      </I18nProvider>,
    )
    const region = screen.getByRole("region", { name: "近 30 天请求质量" })
    expect(within(region).getByText("88.9%")).toBeInTheDocument()
    expect(within(region).getByText("1.5 s")).toBeInTheDocument()
    expect(within(region).getByText("400 ms")).toBeInTheDocument()
    expect(within(region).getByText("已取消")).toBeInTheDocument()
  })

  it("keeps empty states explicit", () => {
    const empty: Reliability = {
      ...reliability,
      requests: 0,
      success: 0,
      failed: 0,
      cancelled: 0,
      successRate: null,
      averageLatencyMs: null,
      averageFirstTokenMs: null,
    }
    render(
      <I18nProvider>
        <UsageQuality reliability={empty} />
        <UsageHours hours={[]} />
        <UsageModelTrend modelDays={[]} days={[]} />
      </I18nProvider>,
    )
    expect(screen.getByText("近 30 天还没有已结束的请求。")).toBeInTheDocument()
    expect(screen.getAllByText("还没有用量记录。通过本地网关发送请求后会自动统计。")).toHaveLength(
      1,
    )
    expect(screen.getByText("还没有模型用量。")).toBeInTheDocument()
  })
})
