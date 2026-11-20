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

it("合并后的推理档位变体只显示基础模型名并计入同一行", () => {
  useUIStore.setState({ language: "zh" })
  render(
    <I18nProvider>
      <ProviderModelUsageView
        items={[
          {
            providerId: "builtin_google",
            providerName: "Google",
            model: "gemini-4-neon",
            variants: ["gemini-4-neon-high", "gemini-4-neon-xhigh"],
            summary: {
              inputTokens: "92,210,840",
              outputTokens: "1,161,618",
              totalTokens: "93,372,458",
              requests: 1284,
              completeUsage: 1284,
            },
          },
        ]}
      />
    </I18nProvider>,
  )
  const google = within(screen.getByRole("region", { name: "Google" }))
  expect(google.getByRole("cell", { name: "gemini-4-neon" })).toBeInTheDocument()
  expect(google.queryByRole("cell", { name: "gemini-4-neon-high" })).toBeNull()
  expect(google.getByText("1 个模型")).toBeInTheDocument()
  expect(google.getByText("1284")).toBeInTheDocument()
})
