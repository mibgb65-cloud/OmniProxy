import { render, screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { ProviderCard } from "./provider-card"

function setup(kind = "mimo") {
  const onProbe = vi.fn(),
    onOutside = vi.fn(),
    user = userEvent.setup()
  const view = render(
    <I18nProvider>
      <ProviderCard
        kind={kind}
        name="小米 MiMo"
        enabled
        disabled={false}
        authMethods={["api_key"]}
        plannedAuthMethods={[]}
        models={["mimo-v2.6-pro", "mimo-v2.6-flash", "mimo-v2.6-pro-ultraspeed"]}
        endpoint="https://api.xiaomimimo.com/v1"
        probePending={false}
        onToggle={vi.fn()}
        onProbe={onProbe}
      />
      <button onClick={onOutside}>浮层外操作</button>
    </I18nProvider>,
  )
  return { ...view, onProbe, onOutside, user }
}
beforeEach(() => useUIStore.setState({ language: "zh" }))

it("shows API Key plus Token Plan support and exposes three cluster base URLs outside the card layout", async () => {
  const { container, user, onProbe } = setup()
  expect(screen.getByText("API Key")).toBeInTheDocument()
  expect(screen.getByText("按量入口 · https://api.xiaomimimo.com/v1")).toBeInTheDocument()
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
  const trigger = screen.getByRole("button", { name: "查看 Token Plan 接入信息" })
  await user.click(trigger)
  const panel = screen.getByRole("dialog", { name: "Token Plan 接入" })
  expect(container).not.toContainElement(panel)
  expect(panel).not.toHaveAttribute("aria-modal", "true")
  expect(trigger).toHaveAttribute("aria-expanded", "true")
  for (const [label, region] of [
    ["中国", "cn"],
    ["新加坡", "sgp"],
    ["欧洲", "ams"],
  ]) {
    expect(within(panel).getByRole("heading", { name: `${label} ${region}` })).toBeInTheDocument()
    expect(
      within(panel).getByText(`https://token-plan-${region}.xiaomimimo.com/v1`),
    ).toBeInTheDocument()
    expect(
      within(panel).getByText(`https://token-plan-${region}.xiaomimimo.com/anthropic`),
    ).toBeInTheDocument()
  }
  expect(within(panel).getByText(/个人版使用 tp- Key，团队版使用 ttp- Key/)).toBeInTheDocument()
  expect(within(panel).getByText(/套餐支持 mimo-v2.6-pro、mimo-v2.6-flash/)).toHaveTextContent(
    "高速版使用普通按量 API",
  )
  expect(within(panel).getByText(/卡片测试范围：普通按量默认入口/)).toBeInTheDocument()
  expect(onProbe).not.toHaveBeenCalled()
})
it.each(["Escape", "close"])(
  "opens from the keyboard and restores trigger focus after %s",
  async (method) => {
    const { user } = setup()
    const trigger = screen.getByRole("button", { name: "查看 Token Plan 接入信息" })
    trigger.focus()
    await user.keyboard("{Enter}")
    expect(screen.getByRole("dialog", { name: "Token Plan 接入" })).toBeInTheDocument()
    if (method === "Escape") await user.keyboard("{Escape}")
    else await user.click(screen.getByRole("button", { name: "关闭 Token Plan 接入信息" }))
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    expect(trigger).toHaveFocus()
  },
)
it("closes on outside interaction and preserves the existing pay-as-you-go probe action", async () => {
  const { user, onProbe, onOutside } = setup()
  await user.click(screen.getByRole("button", { name: "查看 Token Plan 接入信息" }))
  await user.click(screen.getByRole("button", { name: "浮层外操作" }))
  expect(onOutside).toHaveBeenCalledOnce()
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
  await user.click(screen.getByRole("button", { name: "测试按量入口" }))
  expect(onProbe).toHaveBeenCalledOnce()
})
it("does not advertise MiMo plans for other provider kinds", () => {
  setup("deepseek")
  expect(screen.queryByRole("button", { name: "查看 Token Plan 接入信息" })).not.toBeInTheDocument()
  expect(screen.getByRole("button", { name: "测试" })).toBeInTheDocument()
  expect(screen.getByText("https://api.xiaomimimo.com/v1")).toBeInTheDocument()
})
it("presents the English plan information and cluster labels", async () => {
  useUIStore.setState({ language: "en" })
  const { user } = setup()
  await user.click(screen.getByRole("button", { name: "View Token Plan access details" }))
  const panel = screen.getByRole("dialog", { name: "Token Plan access" })
  expect(within(panel).getByRole("heading", { name: "Singapore sgp" })).toBeInTheDocument()
  expect(within(panel).getByRole("heading", { name: "Europe ams" })).toBeInTheDocument()
  expect(within(panel).getByText(/These are base URLs/)).toBeInTheDocument()
})
