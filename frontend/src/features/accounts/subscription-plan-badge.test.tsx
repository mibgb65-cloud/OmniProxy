import { render, screen } from "@testing-library/react"
import { beforeEach, expect, it } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { SubscriptionPlanBadge } from "./subscription-plan-badge"

function setup(plan?: string) {
  return render(
    <I18nProvider>
      <SubscriptionPlanBadge plan={plan} />
    </I18nProvider>,
  )
}

beforeEach(() => {
  useUIStore.setState({ language: "zh" })
})

it("没有套餐字段时显示无套餐", () => {
  setup()
  expect(screen.getByText("无套餐")).toBeInTheDocument()
})

it("只有空白的套餐字段同样视为无套餐", () => {
  setup("  ")
  expect(screen.getByText("无套餐")).toBeInTheDocument()
})

it("Antigravity 的 Ultra 档位映射成可读名称", () => {
  setup("g1-ultra-tier")
  const badge = screen.getByText("Ultra")
  expect(badge.className).toContain("rose")
})

it("Antigravity 的 Pro 档位映射成可读名称并与 Ultra 配色不同", () => {
  setup("g1-pro-tier")
  const badge = screen.getByText("Pro")
  expect(badge.className).toContain("amber")
  expect(badge.className).not.toContain("rose")
})

it("免费档位标识映射成 Free", () => {
  setup("free-tier")
  expect(screen.getByText("Free")).toBeInTheDocument()
})

it("未识别的套餐标识保留原文并使用中性配色", () => {
  setup("g2-super-tier")
  expect(screen.getByText("g2-super-tier").className).toContain("bg-muted")
})

it("已支持的套餐标识沿用原有配色", () => {
  setup("plus")
  expect(screen.getByText("plus").className).toContain("blue")
})
