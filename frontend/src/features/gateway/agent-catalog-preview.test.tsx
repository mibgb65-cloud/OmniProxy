import { render, screen } from "@testing-library/react"
import { expect, it } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { AgentCatalogPreview } from "./agent-catalog-preview"

it("CLI 预览展示模型菜单模式及模型名称和真实 ID", () => {
  useUIStore.setState({ language: "zh" })
  render(
    <I18nProvider>
      <AgentCatalogPreview
        plan={{
          summary: "",
          mode: "cli-model-list",
          defaultModel: "coding-route",
          catalogRevision: "revision",
          models: [
            {
              id: "coding-route",
              name: "Coding Route",
              targetModel: "gpt-test",
              providerId: "p",
              providerKind: "openai",
              alias: true,
              metadata: {},
            },
          ],
        }}
      />
    </I18nProvider>,
  )
  expect(screen.getByText("CLI 模型菜单")).toBeInTheDocument()
  expect(screen.getByText("Coding Route")).toBeInTheDocument()
  expect(screen.getByText("将提供 1 个模型／路由")).toBeInTheDocument()
  expect(screen.queryByText("客户端默认模型")).not.toBeInTheDocument()
})
