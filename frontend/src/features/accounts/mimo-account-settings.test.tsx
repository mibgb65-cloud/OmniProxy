import { act, render, screen, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { CancellablePromise } from "@wailsio/runtime"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { getMiMoSettings, setMiMoPreferredBillingMode } from "../../services/mimo-settings"
import { useUIStore } from "../../stores/ui-store"
import { MiMoAccountSettings } from "./mimo-account-settings"

vi.mock("../../services/mimo-settings", () => ({
  getMiMoSettings: vi.fn(),
  setMiMoPreferredBillingMode: vi.fn(),
}))

function renderSettings() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <MiMoAccountSettings />
      </QueryClientProvider>
    </I18nProvider>,
  )
  return client
}

async function chooseAPI(user: ReturnType<typeof userEvent.setup>) {
  const select = await screen.findByRole("combobox", { name: "默认使用顺序" })
  await waitFor(() => expect(select).toBeEnabled())
  await user.click(select)
  await user.click(screen.getByRole("option", { name: "API Key（按量） → Token Plan" }))
}

describe("MiMo default usage order", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useUIStore.setState({ language: "zh" })
    vi.mocked(getMiMoSettings).mockResolvedValue({ preferredBillingMode: "token_plan" })
    vi.mocked(setMiMoPreferredBillingMode).mockResolvedValue({ preferredBillingMode: "payg" })
  })

  it("loads on demand, saves immediately, and reads the persisted order on reopen", async () => {
    const user = userEvent.setup()
    const client = renderSettings()
    const invalidate = vi.spyOn(client, "invalidateQueries")
    expect(getMiMoSettings).not.toHaveBeenCalled()
    const trigger = screen.getByRole("button", { name: "小米 MiMo 账号设置" })
    await user.click(trigger)
    expect(await screen.findByRole("combobox")).toHaveTextContent("Token Plan → API Key（按量）")
    await chooseAPI(user)
    await waitFor(() =>
      expect(setMiMoPreferredBillingMode).toHaveBeenCalledWith("payg", expect.anything()),
    )
    expect(await screen.findByRole("status")).toHaveTextContent("已保存并生效")
    expect(screen.getByRole("combobox")).toHaveTextContent("API Key（按量） → Token Plan")
    for (const key of ["routes", "default-routes", "gateway-runtime", "agent-clients"]) {
      expect(invalidate).toHaveBeenCalledWith({ queryKey: [key] })
    }
    await user.keyboard("{Escape}")
    vi.mocked(getMiMoSettings).mockResolvedValue({ preferredBillingMode: "payg" })
    await user.click(trigger)
    expect(await screen.findByRole("combobox")).toHaveTextContent("API Key（按量） → Token Plan")
  })

  it("preserves the previous order on failure, hides private errors, and permits retry", async () => {
    vi.mocked(setMiMoPreferredBillingMode).mockRejectedValueOnce(new Error("private database path"))
    const user = userEvent.setup()
    renderSettings()
    await user.click(screen.getByRole("button", { name: "小米 MiMo 账号设置" }))
    await chooseAPI(user)
    expect(await screen.findByRole("alert")).toHaveTextContent("原有顺序已保留")
    expect(screen.queryByText("private database path")).not.toBeInTheDocument()
    expect(screen.getByRole("combobox")).toHaveTextContent("Token Plan → API Key（按量）")
    await chooseAPI(user)
    expect(await screen.findByRole("status")).toHaveTextContent("已保存并生效")
  })

  it("disables the order after a read failure and provides reload", async () => {
    vi.mocked(getMiMoSettings).mockRejectedValueOnce(new Error("secret"))
    const user = userEvent.setup()
    renderSettings()
    await user.click(screen.getByRole("button", { name: "小米 MiMo 账号设置" }))
    expect(await screen.findByRole("alert")).toHaveTextContent("无法读取默认使用顺序")
    expect(screen.getByRole("combobox")).toBeDisabled()
    expect(screen.queryByText("secret")).not.toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "重新读取" }))
    await waitFor(() => expect(screen.getByRole("combobox")).toBeEnabled())
    expect(setMiMoPreferredBillingMode).not.toHaveBeenCalled()
  })

  it("locks saves across reopen and prevents an old read from overwriting the saved order", async () => {
    let finishSave!: () => void
    let finishRead!: () => void
    vi.mocked(setMiMoPreferredBillingMode).mockImplementationOnce(
      () =>
        new CancellablePromise((resolve) => {
          finishSave = () => resolve({ preferredBillingMode: "payg" })
        }),
    )
    const user = userEvent.setup()
    renderSettings()
    const trigger = screen.getByRole("button", { name: "小米 MiMo 账号设置" })
    await user.click(trigger)
    await chooseAPI(user)
    expect(screen.getByRole("combobox")).toBeDisabled()
    await user.click(screen.getByRole("button", { name: "关闭账号设置" }))
    vi.mocked(getMiMoSettings).mockImplementationOnce(
      () =>
        new CancellablePromise((resolve) => {
          finishRead = () => resolve({ preferredBillingMode: "token_plan" })
        }),
    )
    await user.click(trigger)
    expect(screen.getByRole("combobox")).toBeDisabled()
    expect(setMiMoPreferredBillingMode).toHaveBeenCalledTimes(1)
    finishSave()
    await waitFor(() => expect(screen.getByRole("combobox")).toBeEnabled())
    await act(async () => finishRead())
    expect(screen.getByRole("combobox")).toHaveTextContent("API Key（按量） → Token Plan")
  })
})
