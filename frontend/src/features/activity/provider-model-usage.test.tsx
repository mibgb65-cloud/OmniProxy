import { render, screen, within } from "@testing-library/react"
import { expect, it } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { ProviderModelUsageView } from "./provider-model-usage"

it("同名模型在不同提供商下显示各自输入输出与总量", () => {
  useUIStore.setState({ language: "zh" })
  render(
    <I18nProvider>
      <ProviderModelUsageView
        items={[
          {
            providerId: "openai",
            providerName: "OpenAI",
            model: "shared-model",
            summary: {
              inputTokens: "100",
              outputTokens: "10",
              totalTokens: "110",
              requests: 2,
              completeUsage: 2,
            },
          },
          {
            providerId: "deepseek",
            providerName: "DeepSeek",
            model: "shared-model",
            summary: {
              inputTokens: "200",
              outputTokens: "20",
              totalTokens: "220",
              requests: 3,
              completeUsage: 3,
            },
          },
        ]}
      />
    </I18nProvider>,
  )
  const openai = within(screen.getByRole("region", { name: "OpenAI" }))
  const deepseek = within(screen.getByRole("region", { name: "DeepSeek" }))
  expect(openai.getByRole("cell", { name: "shared-model" })).toBeInTheDocument()
  expect(deepseek.getByRole("cell", { name: "shared-model" })).toBeInTheDocument()
  expect(openai.getByRole("cell", { name: "110" })).toBeInTheDocument()
  expect(deepseek.getByRole("cell", { name: "220" })).toBeInTheDocument()
})
