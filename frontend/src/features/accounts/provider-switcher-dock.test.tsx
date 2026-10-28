import { fireEvent, render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import type { ProviderConfiguration } from "../../services/omniproxy"
import { ProviderSwitcherDock } from "./provider-switcher-dock"

const providers: ProviderConfiguration[] = ["openai", "anthropic", "google", "deepseek"].map(
  (kind) => ({
    id: `builtin_${kind}`,
    kind,
    name: kind,
    source: "builtin",
    enabled: true,
    proxyMode: "inherit",
    baseUrl: "",
    createdAt: "",
    updatedAt: "",
  }),
)

function setup() {
  useUIStore.setState({ language: "zh" })
  const onSelect = vi.fn()
  const result = render(
    <I18nProvider>
      <ProviderSwitcherDock
        providers={providers}
        accountCounts={new Map([["builtin_openai", 2]])}
        onSelect={onSelect}
      />
    </I18nProvider>,
  )
  return { ...result, onSelect }
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe("ProviderSwitcherDock", () => {
  it("keeps complete provider names and counts, and switches without a link", async () => {
    const { onSelect } = setup()
    expect(screen.getByText("Anthropic")).toBeVisible()
    expect(screen.getByText("2 个账号")).toBeVisible()
    expect(screen.queryByRole("link")).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole("button", { name: "查看DeepSeek账号详情" }))
    expect(onSelect).toHaveBeenCalledWith("builtin_deepseek")
  })

  it("supports arrow keys, Home and End without selecting until activated", async () => {
    const { onSelect } = setup()
    const first = screen.getByRole("button", { name: "查看OpenAI账号详情" })
    first.focus()
    await userEvent.keyboard("{ArrowRight}")
    expect(screen.getByRole("button", { name: "查看Anthropic账号详情" })).toHaveFocus()
    await userEvent.keyboard("{End}")
    const last = screen.getByRole("button", { name: "查看DeepSeek账号详情" })
    expect(last).toHaveFocus()
    await userEvent.keyboard("{ArrowRight}")
    expect(first).toHaveFocus()
    await userEvent.keyboard("{ArrowLeft}")
    expect(last).toHaveFocus()
    await userEvent.keyboard("{Home}{Enter}")
    expect(first).toHaveFocus()
    expect(onSelect).toHaveBeenCalledExactlyOnceWith("builtin_openai")
  })

  it("updates overflow controls and uses instant scrolling for reduced motion", () => {
    vi.stubGlobal(
      "matchMedia",
      vi.fn(() => ({ matches: true })),
    )
    const { container } = setup()
    const viewport = container.querySelector(".bento-dock-viewport") as HTMLElement
    Object.defineProperties(viewport, {
      clientWidth: { configurable: true, value: 400 },
      scrollWidth: { configurable: true, value: 1000 },
    })
    const scrollBy = vi.fn()
    viewport.scrollBy = scrollBy
    fireEvent.scroll(viewport)
    expect(screen.getByRole("button", { name: "向前滚动提供商" })).toBeDisabled()
    fireEvent.click(screen.getByRole("button", { name: "向后滚动提供商" }))
    expect(scrollBy).toHaveBeenCalledWith({ left: 300, behavior: "instant" })
    viewport.scrollLeft = 600
    fireEvent.scroll(viewport)
    expect(screen.getByRole("button", { name: "向后滚动提供商" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "向前滚动提供商" })).toBeEnabled()
  })
})
