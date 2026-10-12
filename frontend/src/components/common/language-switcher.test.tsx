import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import i18next from "i18next"
import { beforeEach, describe, expect, it } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { defaultLanguage } from "../../i18n/translation"
import { useUIStore } from "../../stores/ui-store"
import { LanguageSwitcher } from "./language-switcher"

function renderSwitcher() {
  return render(
    <I18nProvider>
      <LanguageSwitcher />
    </I18nProvider>,
  )
}

describe("LanguageSwitcher", () => {
  beforeEach(() => {
    localStorage.clear()
    useUIStore.setState({ language: defaultLanguage })
  })

  it("only shows the language button until clicked, then marks the active language", async () => {
    const user = userEvent.setup()
    renderSwitcher()
    expect(screen.queryByText("中文")).not.toBeInTheDocument()
    expect(screen.queryByText("English")).not.toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "语言" }))
    expect(screen.getByRole("menuitemradio", { name: "中文" })).toHaveAttribute(
      "aria-checked",
      "true",
    )
    expect(screen.getByRole("menuitemradio", { name: "English" })).toHaveAttribute(
      "aria-checked",
      "false",
    )
  })

  it("updates the store and i18next when another language is picked", async () => {
    const user = userEvent.setup()
    renderSwitcher()

    await user.click(screen.getByRole("button", { name: "语言" }))
    await user.click(screen.getByRole("menuitemradio", { name: "English" }))

    expect(useUIStore.getState().language).toBe("en")
    await waitFor(() => {
      expect(i18next.resolvedLanguage).toBe("en")
    })
    expect(JSON.parse(localStorage.getItem("omniproxy-ui-state")!).state.language).toBe("en")
    expect(screen.queryByRole("menu")).not.toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Language" })).toHaveFocus()
  })

  it("keeps both language options available when the sidebar is collapsed", async () => {
    const user = userEvent.setup()
    useUIStore.setState({ sidebarCollapsed: true })
    renderSwitcher()

    await user.click(screen.getByRole("button", { name: "语言" }))
    expect(screen.getByRole("menuitemradio", { name: "中文" })).toBeInTheDocument()
    await user.click(screen.getByRole("menuitemradio", { name: "English" }))

    expect(useUIStore.getState().language).toBe("en")
  })

  it("opens with the keyboard and restores focus when dismissed without selecting", async () => {
    const user = userEvent.setup()
    renderSwitcher()
    const trigger = screen.getByRole("button", { name: "语言" })
    trigger.focus()
    await user.keyboard("{Enter}")
    expect(await screen.findByRole("menu")).toBeInTheDocument()
    await user.keyboard("{Escape}")
    expect(screen.queryByRole("menu")).not.toBeInTheDocument()
    expect(trigger).toHaveFocus()
    expect(useUIStore.getState().language).toBe("zh")
  })
})
