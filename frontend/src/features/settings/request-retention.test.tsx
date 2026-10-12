import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { readRetention, setRetention } from "../../services/activity"
import { useUIStore } from "../../stores/ui-store"
import { RequestRetentionSettings } from "./request-retention"
vi.mock("../../services/activity", () => ({ readRetention: vi.fn(), setRetention: vi.fn() }))
it("读取真实保留设置、校验整数边界并保存", async () => {
  useUIStore.setState({ language: "zh" })
  vi.mocked(readRetention).mockResolvedValue({
    maxRecords: 10000,
    storedRecords: 4,
    logCapacity: 5000,
  })
  vi.mocked(setRetention).mockResolvedValue({
    maxRecords: 20000,
    storedRecords: 4,
    logCapacity: 5000,
  })
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <RequestRetentionSettings />
      </QueryClientProvider>
    </I18nProvider>,
  )
  const user = userEvent.setup()
  const field = await screen.findByRole("spinbutton", { name: "请求明细上限" })
  await waitFor(() => expect(field).toBeEnabled())
  expect(field).toHaveValue(10000)
  await user.clear(field)
  await user.type(field, "99")
  expect(screen.getByRole("button", { name: "保存设置" })).toBeDisabled()
  expect(screen.getByRole("alert")).toHaveTextContent("100–1000000")
  await user.clear(field)
  await user.type(field, "20000")
  await user.click(screen.getByRole("button", { name: "保存设置" }))
  await waitFor(() => expect(setRetention).toHaveBeenCalledWith(20000))
  expect(await screen.findByRole("status")).toHaveTextContent("设置已保存")
})
