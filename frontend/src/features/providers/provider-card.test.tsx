import { render, screen } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { ProviderCard } from "./provider-card"
import { useUIStore } from "../../stores/ui-store"

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
  atria: "atria.png",
  river: "river.png",
} as const

function renderCard(kind: string, endpoint = "https://example.test/v1", authMethods = ["api_key"]) {
  return render(
    <I18nProvider>
      <ProviderCard
        kind={kind}
        name={kind}
        enabled
        disabled={false}
        authMethods={authMethods}
        plannedAuthMethods={[]}
        models={[]}
        endpoint={endpoint}
        probePending={false}
        onToggle={vi.fn()}
        onProbe={vi.fn()}
      />
    </I18nProvider>,
  )
}

describe("ProviderCard icons", () => {
  beforeEach(() => useUIStore.setState({ language: "zh" }))

  it.each(["zh", "en"] as const)(
    "keeps Google and OpenAI subscription brands separate in %s",
    (language) => {
      useUIStore.setState({ language })
      const google = renderCard("google", undefined, ["api_key", "subscription_login"])
      expect(screen.getByText("Google / Antigravity")).toBeInTheDocument()
      expect(screen.queryByText("ChatGPT / Codex")).not.toBeInTheDocument()
      google.unmount()
      renderCard("openai", undefined, ["api_key", "subscription_login"])
      expect(screen.getByText("ChatGPT / Codex")).toBeInTheDocument()
      expect(screen.queryByText("Google / Antigravity")).not.toBeInTheDocument()
    },
  )

  it("uses a neutral subscription label for an unknown provider", () => {
    renderCard("openai-compatible", undefined, ["subscription_login"])
    expect(screen.getByText("订阅登录")).toBeInTheDocument()
    expect(screen.queryByText("ChatGPT / Codex")).not.toBeInTheDocument()
  })
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

  it.each(["https://api.atria-asi.ai/v1", "https://API.ATRIA-ASI.AI:443/v1/"])(
    "recognizes an existing custom Atria provider at %s",
    (endpoint) => {
      const { container } = renderCard("openai-compatible", endpoint)
      expect(container.querySelector("img")).toHaveAttribute("src", "/provider-icons/atria.png")
    },
  )

  it.each([
    "https://api.atria-asi.ai.example.test/v1",
    "https://example.test/api.atria-asi.ai/v1",
    "https://api.atria-asi.ai@example.test/v1",
    "https://user@api.atria-asi.ai/v1",
    "https://api.atria-asi.ai:8443/v1",
    "http://api.atria-asi.ai/v1",
    "invalid-url",
    "",
    `https://api.atria-asi.ai/${"a".repeat(2048)}`,
  ])("keeps the generic icon for an unrecognized endpoint %s", (endpoint) => {
    const { container } = renderCard("openai-compatible", endpoint)
    expect(container.querySelector("img")).toBeNull()
  })

  it("keeps a known provider's brand when its endpoint changes", () => {
    const { container } = renderCard("deepseek", "https://api.atria-asi.ai/v1")
    expect(container.querySelector("img")).toHaveAttribute(
      "src",
      "/provider-icons/deepseek-color.svg",
    )
  })
})
