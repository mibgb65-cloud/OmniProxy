import { render } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { ProviderCard } from "./provider-card"

const brandIcons = {
  openai: "openai.svg",
  anthropic: "anthropic.svg",
  google: "google-color.svg",
  deepseek: "deepseek-color.svg",
  zhipu: "zhipu-color.svg",
  kimi: "kimi-color.svg",
  minimax: "minimax-color.svg",
  mimo: "xiaomimimo.svg",
  stepfun: "stepfun-color.svg",
} as const

function renderCard(kind: string) {
  return render(
    <I18nProvider>
      <ProviderCard
        kind={kind}
        name={kind}
        enabled
        disabled={false}
        authMethods={["api_key"]}
        plannedAuthMethods={[]}
        models={[]}
        endpoint="https://example.test/v1"
        probePending={false}
        onToggle={vi.fn()}
        onProbe={vi.fn()}
      />
    </I18nProvider>,
  )
}

describe("ProviderCard icons", () => {
  it.each(Object.entries(brandIcons))("uses the %s brand asset", (kind, file) => {
    const { container } = renderCard(kind)
    expect(container.querySelector("img")).toHaveAttribute("src", `/provider-icons/${file}`)
    expect(container.querySelector("img")).toHaveAttribute("alt", "")
  })

  it("keeps the generic icon for custom providers", () => {
    const { container } = renderCard("openai-compatible")
    expect(container.querySelector("img")).toBeNull()
    expect(container.querySelector("svg")).toBeInTheDocument()
  })
})
