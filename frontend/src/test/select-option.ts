import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"

/** 操作真实弹层中的选项，避免依赖系统 select 的专用模拟。 */
export async function selectOption(
  user: ReturnType<typeof userEvent.setup>,
  trigger: HTMLElement,
  label: string | RegExp,
) {
  trigger.focus()
  await user.keyboard("{Enter}")
  await user.click(await screen.findByRole("option", { name: label }))
}
