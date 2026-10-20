import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { act, render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { getRiverSettings, setRiverAccountRotationMode } from "../../services/river-settings"
import { RiverAccountSettings } from "./river-account-settings"

vi.mock("../../services/river-settings", () => ({
  getRiverSettings: vi.fn(),
  setRiverAccountRotationMode: vi.fn(),
}))

beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(getRiverSettings).mockResolvedValue({ accountRotationMode: "queue" })
  vi.mocked(setRiverAccountRotationMode).mockResolvedValue({ accountRotationMode: "balanced" })
})

async function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <RiverAccountSettings />
      </QueryClientProvider>
    </I18nProvider>,
  )
  const user = userEvent.setup()
  await user.click(screen.getByRole("button", { name: "River 账号设置" }))
  return { user, client }
}

it("saves balanced mode and retains the confirmed choice on reopening", async () => {
  const { user, client } = await setup()
  await waitFor(() => expect(screen.getByRole("radio", { name: "队列使用" })).toBeEnabled())
  expect(screen.getByRole("radio", { name: "队列使用" })).toBeChecked()
  await user.click(screen.getByRole("radio", { name: "均衡使用" }))
  await waitFor(() => expect(screen.getByRole("radio", { name: "均衡使用" })).toBeChecked())
  expect(setRiverAccountRotationMode).toHaveBeenCalledWith("balanced", expect.anything())
  expect(client.getQueryData(["river-settings"])).toEqual({ accountRotationMode: "balanced" })
  await user.click(screen.getByRole("button", { name: "关闭 River 账号设置" }))
  await user.click(screen.getByRole("button", { name: "River 账号设置" }))
  expect(screen.getByRole("radio", { name: "均衡使用" })).toBeChecked()
  act(() => useUIStore.setState({ language: "en" }))
  expect(await screen.findByRole("radio", { name: "Balanced" })).toBeChecked()
})

it("retains the saved mode on failure and hides backend details", async () => {
  vi.mocked(setRiverAccountRotationMode).mockRejectedValue(new Error("private-setting-marker"))
  const { user } = await setup()
  await waitFor(() => expect(screen.getByRole("radio", { name: "均衡使用" })).toBeEnabled())
  await user.click(screen.getByRole("radio", { name: "均衡使用" }))
  expect(await screen.findByRole("alert")).not.toHaveTextContent("private-setting-marker")
  expect(screen.getByRole("radio", { name: "队列使用" })).toBeChecked()
})

it("locks both choices while saving, including after closing and reopening", async () => {
  let finish!: (value: { accountRotationMode: string }) => void
  vi.mocked(setRiverAccountRotationMode).mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve
      }),
  )
  const { user } = await setup()
  await waitFor(() => expect(screen.getByRole("radio", { name: "均衡使用" })).toBeEnabled())
  await user.click(screen.getByRole("radio", { name: "均衡使用" }))
  expect(screen.getByRole("radio", { name: "队列使用" })).toBeDisabled()
  await user.click(screen.getByRole("button", { name: "关闭 River 账号设置" }))
  await user.click(screen.getByRole("button", { name: "River 账号设置" }))
  expect(screen.getByRole("radio", { name: "均衡使用" })).toBeDisabled()
  await act(async () => finish({ accountRotationMode: "balanced" }))
  await waitFor(() => expect(screen.getByRole("radio", { name: "均衡使用" })).toBeChecked())
})

it("shows a stable load error instead of allowing changes", async () => {
  vi.mocked(getRiverSettings).mockRejectedValue(new Error("private-load-marker"))
  await setup()
  expect(await screen.findByRole("alert")).toHaveTextContent("无法读取 River 设置")
  expect(screen.getByRole("radio", { name: "队列使用" })).toBeDisabled()
  expect(screen.queryByText("private-load-marker")).not.toBeInTheDocument()
})
