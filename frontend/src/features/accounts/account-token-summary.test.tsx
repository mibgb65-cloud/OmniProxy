import { render, screen } from "@testing-library/react"
import { expect, it } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { AccountTokenSummary } from "./account-token-summary"

const usage = {
  inputTokens: "1000000",
  outputTokens: "100",
  cachedInputTokens: "",
  requests: 2,
  unknownCacheRequests: 2,
  incompleteRequests: 0,
}

it("distinguishes unknown cache from zero and keeps partial lower bounds without notices", () => {
  const view = render(
    <I18nProvider>
      <AccountTokenSummary usage={usage} error={false} />
    </I18nProvider>,
  )
  expect(screen.getByText("—")).toBeInTheDocument()
  expect(
    screen.queryByText(/部分请求无缓存明细|缓存已包含在输入中|本机留存/),
  ).not.toBeInTheDocument()
  view.rerender(
    <I18nProvider>
      <AccountTokenSummary
        usage={{ ...usage, cachedInputTokens: "800000", incompleteRequests: 1 }}
        error={false}
      />
    </I18nProvider>,
  )
  expect(screen.getByText("≥ 800K")).toBeInTheDocument()
  expect(screen.queryByText(/部分请求用量不完整|部分请求无缓存明细/)).not.toBeInTheDocument()
  view.rerender(
    <I18nProvider>
      <AccountTokenSummary
        usage={{ ...usage, cachedInputTokens: "0", unknownCacheRequests: 0 }}
        error={false}
      />
    </I18nProvider>,
  )
  expect(screen.getByText("0")).toBeInTheDocument()
})

it("keeps exact large token counts and hides stale values on errors", () => {
  const view = render(
    <I18nProvider>
      <AccountTokenSummary usage={{ ...usage, inputTokens: "9007199254740993" }} error={false} />
    </I18nProvider>,
  )
  expect(screen.getByLabelText("9,007,199,254,740,993")).toBeInTheDocument()
  view.rerender(
    <I18nProvider>
      <AccountTokenSummary usage={usage} error />
    </I18nProvider>,
  )
  expect(screen.getAllByText("—")).toHaveLength(3)
  expect(screen.queryByText("用量暂不可用")).not.toBeInTheDocument()
  expect(screen.queryByText("1M")).not.toBeInTheDocument()
})
