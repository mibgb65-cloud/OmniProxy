import { render, screen } from "@testing-library/react"
import { expect, it } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import type { RequestItem } from "../../services/activity"
import { RequestAttempts } from "./request-attempts"

it("分别展示拒绝与成功账号，未报告用量保持未知", () => {
  const base = {
    providerId: "deepseek",
    accountId: "account",
    providerRegion: "",
    billingMode: "",
    finishedAt: "2026-10-18T00:00:02Z",
    usageComplete: false,
  }
  const item = {
    attempts: [
      {
        ...base,
        index: 1,
        startedAt: "2026-10-18T00:00:00Z",
        accountName: "账号 A",
        providerName: "DeepSeek",
        model: "deepseek-flash",
        state: "failed",
        statusCode: 429,
        errorCode: "upstream_rate_limited",
        latencyMs: 10,
        inputTokens: "0",
        outputTokens: "0",
        inputKnown: false,
        outputKnown: false,
      },
      {
        ...base,
        index: 2,
        startedAt: "2026-10-18T00:00:01Z",
        accountName: "账号 B",
        providerName: "DeepSeek",
        model: "deepseek-flash",
        state: "success",
        statusCode: 200,
        errorCode: "",
        latencyMs: 30,
        inputTokens: "12",
        outputTokens: "4",
        inputKnown: true,
        outputKnown: true,
      },
    ],
  } satisfies Pick<RequestItem, "attempts">
  render(
    <I18nProvider>
      <RequestAttempts item={item} />
    </I18nProvider>,
  )
  expect(screen.getByRole("region", { name: "账号尝试记录" })).toBeInTheDocument()
  expect(screen.getByText("第 1 次 · 账号 A")).toBeInTheDocument()
  expect(screen.getByText("第 2 次 · 账号 B")).toBeInTheDocument()
  expect(screen.getByText("已切换 1 次")).toBeInTheDocument()
  expect(screen.getByText("输入 Token：未报告 · 输出 Token：未报告")).toBeInTheDocument()
  expect(screen.getByText("输入 Token：12 · 输出 Token：4")).toBeInTheDocument()
})
