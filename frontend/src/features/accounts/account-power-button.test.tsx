import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { AccountPowerButton } from "./account-power-button"

describe("account power button", () => {
  beforeEach(() => useUIStore.setState({ language: "zh" }))

  it("shows enabled state and supports keyboard shutdown", async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    render(
      <I18nProvider>
        <AccountPowerButton name="测试账号" enabled={true} disabled={false} onChange={onChange} />
      </I18nProvider>,
    )
    const button = screen.getByRole("button", { name: "切换账号“测试账号”的启用状态" })
    expect(button).toHaveAttribute("aria-pressed", "true")
    expect(button.textContent).toBe("")
    await user.tab()
    expect(await screen.findByRole("tooltip")).toHaveTextContent("停用账号")
    await user.keyboard(" ")
    expect(onChange).toHaveBeenCalledWith(false)
  })

  it("can enable a stopped account and blocks changes while busy", async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    const { rerender } = render(
      <I18nProvider>
        <AccountPowerButton name="测试账号" enabled={false} disabled={false} onChange={onChange} />
      </I18nProvider>,
    )
    const button = screen.getByRole("button", { name: "切换账号“测试账号”的启用状态" })
    expect(button).toHaveAttribute("aria-pressed", "false")
    await user.click(button)
    expect(onChange).toHaveBeenCalledWith(true)
    rerender(
      <I18nProvider>
        <AccountPowerButton name="测试账号" enabled={false} disabled={true} onChange={onChange} />
      </I18nProvider>,
    )
    await user.click(button)
    expect(onChange).toHaveBeenCalledTimes(1)
  })
})
