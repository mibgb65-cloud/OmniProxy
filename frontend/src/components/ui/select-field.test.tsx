import { useState } from "react"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { expect, it } from "vitest"
import { SelectField } from "./select-field"
import { selectOption } from "../../test/select-option"

function Fixture() {
  const [value, setValue] = useState("")
  return (
    <>
      <SelectField
        aria-label="结果"
        value={value}
        onValueChange={setValue}
        options={[
          { value: "", label: "全部结果" },
          { value: "ok", label: "成功" },
          { value: "value:", label: "特殊模型名" },
          { value: "blocked", label: "不可用", disabled: true },
        ]}
      />
      <output data-testid="selected">{value}</output>
    </>
  )
}
it("自定义菜单可选择、清空和保留特殊业务值", async () => {
  const user = userEvent.setup()
  render(<Fixture />)
  const trigger = screen.getByRole("combobox", { name: "结果" })
  expect(trigger.tagName).toBe("BUTTON")
  await selectOption(user, trigger, "成功")
  expect(trigger).toHaveTextContent("成功")
  expect(screen.getByTestId("selected")).toHaveTextContent("ok")
  await selectOption(user, trigger, "全部结果")
  expect(screen.getByTestId("selected")).toBeEmptyDOMElement()
  await selectOption(user, trigger, "特殊模型名")
  expect(screen.getByTestId("selected")).toHaveTextContent("value:")
})
it("键盘打开、禁用选项与 Escape 返回焦点", async () => {
  const user = userEvent.setup()
  render(<Fixture />)
  const trigger = screen.getByRole("combobox", { name: "结果" })
  trigger.focus()
  await user.keyboard("{Enter}")
  expect(await screen.findByRole("listbox")).toBeInTheDocument()
  expect(screen.getByRole("option", { name: "不可用" })).toHaveAttribute("aria-disabled", "true")
  await user.keyboard("{Escape}")
  expect(screen.queryByRole("listbox")).not.toBeInTheDocument()
  expect(trigger).toHaveFocus()
})
