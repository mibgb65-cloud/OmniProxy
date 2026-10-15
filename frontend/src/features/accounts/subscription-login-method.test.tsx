import { useState } from "react"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { expect, it } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { SubscriptionLoginMethod } from "./subscription-login-method"

function MethodChoice() {
  const [value, setValue] = useState("browser")
  return <SubscriptionLoginMethod id="login-method" value={value} onChange={setValue} />
}

it("selects a method from its whole card and retains accessible radio descriptions", async () => {
  useUIStore.setState({ language: "zh" })
  const user = userEvent.setup()
  render(
    <I18nProvider>
      <MethodChoice />
    </I18nProvider>,
  )
  const browser = screen.getByRole("radio", { name: "浏览器授权" })
  const device = screen.getByRole("radio", { name: "设备码授权" })
  expect(browser).toBeChecked()
  expect(device).toHaveAccessibleDescription("在官方页面输入设备码完成授权。")
  await user.click(screen.getByText("在官方页面输入设备码完成授权。"))
  expect(device).toBeChecked()
  expect(browser).not.toBeChecked()
  browser.focus()
  await user.keyboard(" ")
  expect(browser).toBeChecked()
})
