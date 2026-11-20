import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import {
  clearRequestContent,
  readRetention,
  setRequestContentRecording,
  setRetention,
} from "../../services/activity"
import { useUIStore } from "../../stores/ui-store"
import { RequestRetentionSettings } from "./request-retention"
vi.mock("../../services/activity", () => ({
  readRetention: vi.fn(),
  setRetention: vi.fn(),
  setRequestContentRecording: vi.fn(),
  clearRequestContent: vi.fn(),
}))
it("读取真实保留设置、校验整数边界并保存", async () => {
  useUIStore.setState({ language: "zh" })
  vi.mocked(readRetention).mockResolvedValue({
    maxRecords: 10000,
    storedRecords: 4,
    logCapacity: 5000,
    contentEnabled: false,
    contentBytes: 0,
    contentBudget: 536870912,
  })
  vi.mocked(setRetention).mockResolvedValue({
    maxRecords: 20000,
    storedRecords: 4,
    logCapacity: 5000,
    contentEnabled: false,
    contentBytes: 0,
    contentBudget: 536870912,
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

it("开启请求正文记录并清除已保存正文", async () => {
  useUIStore.setState({ language: "zh" })
  vi.mocked(readRetention).mockResolvedValue({
    maxRecords: 10000,
    storedRecords: 4,
    logCapacity: 5000,
    contentEnabled: false,
    contentBytes: 0,
    contentBudget: 536870912,
  })
  vi.mocked(setRequestContentRecording).mockResolvedValue({
    maxRecords: 10000,
    storedRecords: 4,
    logCapacity: 5000,
    contentEnabled: true,
    contentBytes: 4096,
    contentBudget: 536870912,
  })
  vi.mocked(clearRequestContent).mockResolvedValue(3)
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <RequestRetentionSettings />
      </QueryClientProvider>
    </I18nProvider>,
  )
  const user = userEvent.setup()
  const toggle = await screen.findByRole("switch", { name: "记录请求正文" })
  await waitFor(() => expect(toggle).toBeEnabled())
  await user.click(toggle)
  await waitFor(() => expect(setRequestContentRecording).toHaveBeenCalledWith(true))
  await waitFor(() => expect(toggle).toBeChecked())
  await user.click(screen.getByRole("button", { name: "清除已保存的正文" }))
  await waitFor(() => expect(clearRequestContent).toHaveBeenCalledTimes(1))
  expect(await screen.findByText("已清除 3 条请求正文。")).toBeInTheDocument()
})
