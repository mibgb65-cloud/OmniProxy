import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { TypographyProvider } from "../../app/typography-provider"
import { useUIStore } from "../../stores/ui-store"
import { AppearanceSettings } from "./appearance-settings"

describe("global text size", () => {
  beforeEach(() => {
    localStorage.clear()
    useUIStore.setState({ language: "zh", fontScale: 100 })
  })
  it("supports keyboard adjustment, live application, saving and restoring default", async () => {
    const user = userEvent.setup()
    render(
      <I18nProvider>
        <TypographyProvider>
          <AppearanceSettings />
        </TypographyProvider>
      </I18nProvider>,
    )
    const slider = screen.getByRole("slider", { name: "全局文字大小" })
    expect(slider).toHaveAttribute("aria-valuenow", "100")
    expect(screen.getByRole("button", { name: "恢复默认" })).toBeDisabled()
    await user.tab()
    expect(slider).toHaveFocus()
    await user.keyboard("{End}")
    expect(slider).toHaveAttribute("aria-valuenow", "140")
    expect(document.documentElement.style.fontSize).toBe("140%")
    await user.keyboard("{ArrowLeft}")
    expect(useUIStore.getState().fontScale).toBe(135)
    expect(JSON.parse(localStorage.getItem("omniproxy-ui-state")!).state.fontScale).toBe(135)
    await user.click(screen.getByRole("button", { name: "恢复默认" }))
    expect(useUIStore.getState().fontScale).toBe(100)
    expect(document.documentElement.style.fontSize).toBe("100%")
  })
})
