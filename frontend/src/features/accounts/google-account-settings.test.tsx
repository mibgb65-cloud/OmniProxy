import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { act, render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { getGoogleSettings, setGoogleAccountRotationMode } from "../../services/google-settings"
import { GoogleAccountSettings } from "./google-account-settings"

vi.mock("../../services/google-settings", () => ({
  getGoogleSettings: vi.fn(),
  setGoogleAccountRotationMode: vi.fn(),
}))

beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(getGoogleSettings).mockResolvedValue({ accountRotationMode: "queue" })
  vi.mocked(setGoogleAccountRotationMode).mockResolvedValue({ accountRotationMode: "balanced" })
})

async function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <GoogleAccountSettings />
      </QueryClientProvider>
    </I18nProvider>,
  )
  const user = userEvent.setup()
  await user.click(screen.getByRole("button", { name: "Google 账号设置" }))
  return { user, client }
}

it("saves balanced mode and keeps the confirmed choice on reopening", async () => {
  const { user, client } = await setup()
  await waitFor(() => expect(screen.getByRole("radio", { name: "队列轮换" })).toBeEnabled())
  expect(screen.getByRole("radio", { name: "队列轮换" })).toBeChecked()
  expect(screen.getByText(/只对 Google AI 订阅账号生效/)).toBeInTheDocument()
  await user.click(screen.getByRole("radio", { name: "平衡切换" }))
  await waitFor(() => expect(screen.getByRole("radio", { name: "平衡切换" })).toBeChecked())
  expect(setGoogleAccountRotationMode).toHaveBeenCalledWith("balanced", expect.anything())
  expect(client.getQueryData(["google-settings"])).toEqual({ accountRotationMode: "balanced" })
  await user.click(screen.getByRole("button", { name: "关闭 Google 账号设置" }))
  await user.click(screen.getByRole("button", { name: "Google 账号设置" }))
  expect(screen.getByRole("radio", { name: "平衡切换" })).toBeChecked()
  act(() => useUIStore.setState({ language: "en" }))
  expect(await screen.findByRole("radio", { name: "Balanced" })).toBeChecked()
})

it("keeps the saved mode and hides backend details when saving fails", async () => {
  vi.mocked(setGoogleAccountRotationMode).mockRejectedValue(new Error("private-setting-marker"))
  const { user } = await setup()
  await waitFor(() => expect(screen.getByRole("radio", { name: "平衡切换" })).toBeEnabled())
  await user.click(screen.getByRole("radio", { name: "平衡切换" }))
  expect(await screen.findByRole("alert")).not.toHaveTextContent("private-setting-marker")
  expect(screen.getByRole("radio", { name: "队列轮换" })).toBeChecked()
})

it("locks both choices while saving", async () => {
  let finish!: (value: { accountRotationMode: string }) => void
  vi.mocked(setGoogleAccountRotationMode).mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve
      }),
  )
  const { user } = await setup()
  await waitFor(() => expect(screen.getByRole("radio", { name: "平衡切换" })).toBeEnabled())
  await user.click(screen.getByRole("radio", { name: "平衡切换" }))
  expect(screen.getByRole("radio", { name: "队列轮换" })).toBeDisabled()
  await act(async () => finish({ accountRotationMode: "balanced" }))
  await waitFor(() => expect(screen.getByRole("radio", { name: "平衡切换" })).toBeChecked())
})

it("shows a stable load error instead of allowing changes", async () => {
  vi.mocked(getGoogleSettings).mockRejectedValue(new Error("private-load-marker"))
  await setup()
  expect(await screen.findByRole("alert")).toHaveTextContent("无法读取 Google 设置")
  expect(screen.getByRole("radio", { name: "队列轮换" })).toBeDisabled()
  expect(screen.queryByText("private-load-marker")).not.toBeInTheDocument()
})
