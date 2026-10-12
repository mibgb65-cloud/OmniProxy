import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { SubscriptionAccountID } from "./subscription-account-id"

describe("subscription account ID", () => {
  beforeEach(() => useUIStore.setState({ language: "zh" }))

  it("shows a compact ID and reveals its complete value on focus", async () => {
    const user = userEvent.setup()
    const id = "11111111-2222-3333-4444-555555555555"
    render(
      <I18nProvider>
        <SubscriptionAccountID id={id} />
      </I18nProvider>,
    )
    expect(screen.getByText("account_id: 11111111…555555")).toBeInTheDocument()
    await user.tab()
    expect(await screen.findByRole("tooltip")).toHaveTextContent(`account_id: ${id}`)
    expect(screen.getByText("account_id: 11111111…555555")).not.toHaveAttribute("title")
  })

  it("shows a refresh hint for existing accounts without an ID", async () => {
    const user = userEvent.setup()
    render(
      <I18nProvider>
        <SubscriptionAccountID />
      </I18nProvider>,
    )
    expect(screen.getByText("account_id: 待获取")).toBeInTheDocument()
    await user.tab()
    expect(await screen.findByRole("tooltip")).toHaveTextContent("请点击刷新状态获取")
  })
})
