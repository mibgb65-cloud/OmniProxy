import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { OpenAIAccountSettings } from "./openai-account-settings"

describe("OpenAI account settings", () => {
  beforeEach(() => useUIStore.setState({ language: "zh" }))
  it("opens an empty global settings dialog and restores focus when closed", async () => {
    const user = userEvent.setup()
    render(
      <I18nProvider>
        <OpenAIAccountSettings />
      </I18nProvider>,
    )
    const trigger = screen.getByRole("button", { name: "OpenAI 账号设置" })
    await user.click(trigger)
    const panel = screen.getByRole("dialog", { name: "OpenAI 账号设置" })
    expect(panel).toHaveAttribute("aria-modal", "true")
    expect(panel.querySelector("input,select,textarea")).toBeNull()
    expect(panel).toHaveTextContent("OpenAI 账号设置")
    await user.keyboard("{Escape}")
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
    expect(trigger).toHaveFocus()
    await user.click(trigger)
    await user.click(screen.getByRole("button", { name: "关闭账号设置" }))
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument())
  })
})
