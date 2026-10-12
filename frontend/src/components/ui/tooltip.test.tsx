import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it } from "vitest"
import { Button } from "./button"
import { Tooltip, TooltipProvider } from "./tooltip"

describe("styled tooltip", () => {
  it("opens on hover and dismisses with Escape without a native title", async () => {
    const user = userEvent.setup()
    render(
      <Tooltip content="编辑账号">
        <Button aria-label="编辑测试账号">编辑</Button>
      </Tooltip>,
    )
    const button = screen.getByRole("button", { name: "编辑测试账号" })
    expect(button).not.toHaveAttribute("title")
    await user.hover(button)
    expect(await screen.findByRole("tooltip")).toHaveTextContent("编辑账号")
    await user.keyboard("{Escape}")
    await waitFor(() => expect(screen.queryByRole("tooltip")).not.toBeInTheDocument())
  })

  it("provides the same information on keyboard focus and closes on blur", async () => {
    const user = userEvent.setup()
    render(
      <TooltipProvider>
        <Tooltip content="刷新状态">
          <Button aria-label="刷新状态" />
        </Tooltip>
        <Button>下一项</Button>
      </TooltipProvider>,
    )
    await user.tab()
    expect(screen.getByRole("button", { name: "刷新状态" })).toHaveFocus()
    expect(await screen.findByRole("tooltip")).toHaveTextContent("刷新状态")
    await user.tab()
    await waitFor(() => expect(screen.queryByRole("tooltip")).not.toBeInTheDocument())
  })
})
