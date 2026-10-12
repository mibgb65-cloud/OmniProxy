import { selectOption } from "../../test/select-option"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { listModelCatalog, syncAccountModels } from "../../services/models"
import { ModelCatalog } from "./model-catalog"
import { useUIStore } from "../../stores/ui-store"
vi.mock("../../services/models", () => ({ listModelCatalog: vi.fn(), syncAccountModels: vi.fn() }))
it("按账号同步并在失败时保留原模型目录", async () => {
  useUIStore.setState({ language: "zh" })
  vi.mocked(listModelCatalog).mockResolvedValue([
    { accountId: "a", providerId: "p", model: "gpt-test", name: "GPT Test", fetchedAt: "now" },
  ])
  vi.mocked(syncAccountModels).mockRejectedValue(new Error("secret token"))
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <ModelCatalog
          accounts={[
            {
              id: "a",
              providerId: "p",
              name: "Account",
              enabled: true,
              createdAt: "",
              updatedAt: "",
              authMethod: "subscription_login",
              authState: "authenticated",
            },
          ]}
          providers={[
            {
              id: "p",
              kind: "openai",
              source: "builtin",
              name: "OpenAI",
              baseUrl: "",
              enabled: true,
              createdAt: "",
              updatedAt: "",
            },
          ]}
        />
      </QueryClientProvider>
    </I18nProvider>,
  )
  const user = userEvent.setup()
  expect(await screen.findByText("gpt-test")).toBeInTheDocument()
  await selectOption(user, screen.getByRole("combobox"), "Account · OpenAI")
  await user.click(screen.getByRole("button", { name: "同步官方模型" }))
  expect(await screen.findByRole("alert")).toHaveTextContent("保留上次成功")
  expect(screen.getByText("gpt-test")).toBeInTheDocument()
  expect(syncAccountModels).toHaveBeenCalledWith("a")
  expect(screen.queryByText("secret token")).not.toBeInTheDocument()
})
