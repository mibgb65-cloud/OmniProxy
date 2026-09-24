import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it } from "vitest"
import { LanguageSwitcher } from "./language-switcher"
import { useUIStore } from "../../stores/ui-store"

describe("LanguageSwitcher", () => {
  beforeEach(() => {
    localStorage.clear()
    useUIStore.setState({ language: "zh" })
  })

  it("marks the active language", () => {
    render(<LanguageSwitcher collapsed={false} />)
    expect(screen.getByRole("button", { name: "中文" })).toHaveAttribute("aria-pressed", "true")
    expect(screen.getByRole("button", { name: "English" })).toHaveAttribute("aria-pressed", "false")
  })

  it("updates the store when another language is picked", async () => {
    const user = userEvent.setup()
    render(<LanguageSwitcher collapsed={false} />)

    await user.click(screen.getByRole("button", { name: "English" }))

    expect(useUIStore.getState().language).toBe("en")
  })

  it("renders a single toggle button when collapsed", async () => {
    const user = userEvent.setup()
    render(<LanguageSwitcher collapsed />)

    const toggle = screen.getByRole("button", { name: "语言" })
    await user.click(toggle)

    expect(useUIStore.getState().language).toBe("en")
  })
})
