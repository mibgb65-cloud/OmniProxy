import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { CancellablePromise } from "@wailsio/runtime"
import { render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import {
  getKimiSettings,
  setKimiPreferredBillingMode,
  setKimiPreferredRegion,
} from "../../services/kimi-settings"
import { useUIStore } from "../../stores/ui-store"
import { KimiAccountSettings } from "./kimi-account-settings"

vi.mock("../../services/kimi-settings", () => ({
  getKimiSettings: vi.fn(),
  setKimiPreferredBillingMode: vi.fn(),
  setKimiPreferredRegion: vi.fn(),
  openKimiCodeConsole: vi.fn(),
}))
beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(getKimiSettings).mockResolvedValue({
    preferredBillingMode: "coding",
    preferredRegion: "cn",
  })
})
function renderSettings() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <KimiAccountSettings />
      </QueryClientProvider>
    </I18nProvider>,
  )
  return client
}

it("saves independent billing and region preferences immediately and locks both controls", async () => {
  const user = userEvent.setup()
  let finish!: () => void
  vi.mocked(setKimiPreferredRegion).mockImplementationOnce(
    () =>
      new CancellablePromise((resolve) => {
        finish = () => resolve({ preferredBillingMode: "coding", preferredRegion: "global" })
      }),
  )
  vi.mocked(setKimiPreferredBillingMode).mockResolvedValue({
    preferredBillingMode: "payg",
    preferredRegion: "global",
  })
  const client = renderSettings()
  expect(getKimiSettings).not.toHaveBeenCalled()
  await user.click(screen.getByRole("button", { name: "Kimi 账号设置" }))
  const region = screen.getByRole("combobox", { name: "默认地区顺序" })
  await waitFor(() => expect(region).toBeEnabled())
  await user.click(region)
  await user.click(screen.getByRole("option", { name: "海外 → 国内" }))
  expect(screen.getByRole("combobox", { name: "默认使用顺序" })).toBeDisabled()
  expect(region).toBeDisabled()
  await user.click(screen.getByRole("button", { name: "关闭账号设置" }))
  await user.click(screen.getByRole("button", { name: "Kimi 账号设置" }))
  expect(screen.getByRole("combobox", { name: "默认使用顺序" })).toBeDisabled()
  finish()
  await waitFor(() => expect(screen.getByRole("combobox", { name: "默认使用顺序" })).toBeEnabled())
  await user.click(screen.getByRole("combobox", { name: "默认使用顺序" }))
  await user.click(screen.getByRole("option", { name: "API Key（按量） → Kimi Code 订阅" }))
  await waitFor(() =>
    expect(client.getQueryData(["kimi-settings"])).toEqual({
      preferredBillingMode: "payg",
      preferredRegion: "global",
    }),
  )
  expect(setKimiPreferredRegion).toHaveBeenCalledWith("global", expect.anything())
  expect(setKimiPreferredBillingMode).toHaveBeenCalledWith("payg", expect.anything())
})

it("keeps the previous region on save failure and hides private errors", async () => {
  const user = userEvent.setup()
  vi.mocked(setKimiPreferredRegion).mockRejectedValueOnce(new Error("private credential path"))
  renderSettings()
  await user.click(screen.getByRole("button", { name: "Kimi 账号设置" }))
  const region = screen.getByRole("combobox", { name: "默认地区顺序" })
  await waitFor(() => expect(region).toBeEnabled())
  await user.click(region)
  await user.click(screen.getByRole("option", { name: "海外 → 国内" }))
  const section = screen.getByRole("region", { name: "默认地区顺序" })
  expect(await within(section).findByRole("alert")).toHaveTextContent("原有顺序已保留")
  expect(screen.getByRole("combobox", { name: "默认地区顺序" })).toHaveTextContent("国内 → 海外")
  expect(screen.queryByText("private credential path")).not.toBeInTheDocument()
})
