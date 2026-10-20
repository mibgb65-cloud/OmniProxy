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

it("distinguishes unknown cache from zero and labels partial cache as a lower bound", () => {
  const view = render(
    <I18nProvider>
      <AccountTokenSummary usage={usage} error={false} />
    </I18nProvider>,
  )
  expect(screen.getByText("—")).toBeInTheDocument()
  expect(screen.getByText(/部分请求无缓存明细/)).toBeInTheDocument()
  view.rerender(
    <I18nProvider>
      <AccountTokenSummary usage={{ ...usage, cachedInputTokens: "800000" }} error={false} />
    </I18nProvider>,
  )
  expect(screen.getByText("≥ 800K")).toBeInTheDocument()
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
  expect(screen.getByTitle("9,007,199,254,740,993")).toBeInTheDocument()
  view.rerender(
    <I18nProvider>
      <AccountTokenSummary usage={usage} error />
    </I18nProvider>,
  )
  expect(screen.getAllByText("—")).toHaveLength(3)
  expect(screen.getByText("用量暂不可用")).toBeInTheDocument()
  expect(screen.queryByText("1M")).not.toBeInTheDocument()
})
