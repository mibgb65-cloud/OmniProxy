import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import i18next from "i18next"
import { beforeEach, describe, expect, it } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { defaultLanguage } from "../../i18n/translation"
import { useUIStore } from "../../stores/ui-store"
import { LanguageSwitcher } from "./language-switcher"

function renderSwitcher(collapsed = false) {
  return render(
    <I18nProvider>
      <LanguageSwitcher collapsed={collapsed} />
    </I18nProvider>,
  )
}

describe("LanguageSwitcher", () => {
  beforeEach(() => {
    localStorage.clear()
    useUIStore.setState({ language: defaultLanguage })
  })

  it("marks the active language", () => {
    renderSwitcher()
    expect(screen.getByRole("button", { name: "中文" })).toHaveAttribute("aria-pressed", "true")
    expect(screen.getByRole("button", { name: "English" })).toHaveAttribute("aria-pressed", "false")
  })

  it("updates the store and i18next when another language is picked", async () => {
    const user = userEvent.setup()
    renderSwitcher()

    await user.click(screen.getByRole("button", { name: "English" }))

    expect(useUIStore.getState().language).toBe("en")
    await waitFor(() => {
      expect(i18next.resolvedLanguage).toBe("en")
    })
  })

  it("renders a single toggle button when collapsed", async () => {
    const user = userEvent.setup()
    renderSwitcher(true)

    await user.click(screen.getByRole("button", { name: "语言" }))

    expect(useUIStore.getState().language).toBe("en")
  })
})
