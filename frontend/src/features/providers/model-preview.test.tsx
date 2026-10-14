import { render, screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { ProviderCard } from "./provider-card"

const models = ["mimo-v2.6-pro", "mimo-v2.6-flash", "mimo-v2.6-pro-ultraspeed"]

function setup(items = models) {
  const outsideAction = vi.fn()
  const { container } = render(
    <I18nProvider>
      <ProviderCard
        kind="mimo"
        name="小米 MiMo"
        enabled
        disabled={false}
        authMethods={["api_key"]}
        plannedAuthMethods={[]}
        models={items}
        endpoint="https://api.xiaomimimo.com/v1"
        probePending={false}
        onToggle={vi.fn()}
        onProbe={vi.fn()}
      />
      <button onClick={outsideAction}>面板外操作</button>
    </I18nProvider>,
  )
  return { container, outsideAction, user: userEvent.setup() }
}

describe("提供商模型浮层", () => {
  beforeEach(() => useUIStore.setState({ language: "zh" }))

  it("将完整列表挂到卡片外，不插入卡片布局", async () => {
    const { container, user } = setup()
    const trigger = screen.getByRole("button", { name: /查看全部预设模型/ })
    expect(trigger).toHaveAttribute("aria-expanded", "false")
    expect(within(trigger).getByText("+1")).toBeInTheDocument()
    expect(screen.queryByRole("list")).not.toBeInTheDocument()
    await user.click(trigger)
    const panel = screen.getByRole("dialog", { name: "小米 MiMo · 预设模型" })
    expect(container).not.toContainElement(panel)
    expect(container.querySelector("ul")).toBeNull()
    expect(panel).not.toHaveAttribute("aria-modal", "true")
    expect(trigger).toHaveAttribute("aria-expanded", "true")
    expect(
      within(panel)
        .getAllByRole("listitem")
        .map((item) => item.textContent),
    ).toEqual(models)
  })

  it.each(["Escape", "入口", "关闭"])("键盘打开后通过%s关闭并恢复焦点", async (action) => {
    const { user } = setup()
    const trigger = screen.getByRole("button", { name: /查看全部预设模型/ })
    trigger.focus()
    await user.keyboard("{Enter}")
    expect(screen.getByRole("dialog")).toBeInTheDocument()
    if (action === "Escape") await user.keyboard("{Escape}")
    else if (action === "入口") await user.click(trigger)
    else await user.click(screen.getByRole("button", { name: "收起预设模型" }))
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    expect(trigger).toHaveAttribute("aria-expanded", "false")
    expect(trigger).toHaveFocus()
  })

  it("点击外部操作直接响应并关闭浮层", async () => {
    const { user, outsideAction } = setup()
    await user.click(screen.getByRole("button", { name: /查看全部预设模型/ }))
    const outside = screen.getByRole("button", { name: "面板外操作" })
    await user.click(outside)
    expect(outsideAction).toHaveBeenCalledOnce()
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    expect(outside).toHaveFocus()
  })

  it("无预设模型时保留自定义接口说明", () => {
    setup([])
    expect(screen.getByText("模型由自定义接口提供")).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /查看全部预设模型/ })).not.toBeInTheDocument()
  })
})
