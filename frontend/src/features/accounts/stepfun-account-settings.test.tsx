import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { getMiMoSettings, setMiMoPreferredBillingMode } from "../../services/mimo-settings"
import { getStepFunSettings, setStepFunPreferredBillingMode } from "../../services/stepfun-settings"
import { useUIStore } from "../../stores/ui-store"
import { MiMoAccountSettings } from "./mimo-account-settings"
import { StepFunAccountSettings } from "./stepfun-account-settings"

vi.mock("../../services/mimo-settings", () => ({
  getMiMoSettings: vi.fn(),
  setMiMoPreferredBillingMode: vi.fn(),
}))
vi.mock("../../services/stepfun-settings", () => ({
  getStepFunSettings: vi.fn(),
  setStepFunPreferredBillingMode: vi.fn(),
}))

beforeEach(() => {
  vi.clearAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(getStepFunSettings).mockResolvedValue({ preferredBillingMode: "step_plan" })
  vi.mocked(setStepFunPreferredBillingMode).mockResolvedValue({ preferredBillingMode: "payg" })
  vi.mocked(getMiMoSettings).mockResolvedValue({ preferredBillingMode: "token_plan" })
})

it("saves StepFun order on demand and keeps MiMo settings independent", async () => {
  const user = userEvent.setup()
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <MiMoAccountSettings />
        <StepFunAccountSettings />
      </QueryClientProvider>
    </I18nProvider>,
  )
  expect(getStepFunSettings).not.toHaveBeenCalled()
  const trigger = screen.getByRole("button", { name: "阶跃星辰账号设置" })
  await user.click(trigger)
  const select = screen.getByRole("combobox", { name: "默认使用顺序" })
  await waitFor(() => expect(select).toHaveTextContent("Step Plan → API Key（按量）"))
  await waitFor(() => expect(select).toBeEnabled())
  await user.click(select)
  await user.click(screen.getByRole("option", { name: "API Key（按量） → Step Plan" }))
  await waitFor(() =>
    expect(setStepFunPreferredBillingMode).toHaveBeenCalledWith("payg", expect.anything()),
  )
  await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("已保存并生效"))
  await user.click(screen.getByRole("button", { name: "关闭账号设置" }))
  vi.mocked(getStepFunSettings).mockResolvedValue({ preferredBillingMode: "payg" })
  await user.click(trigger)
  await waitFor(() =>
    expect(screen.getByRole("combobox")).toHaveTextContent("API Key（按量） → Step Plan"),
  )
  await user.click(screen.getByRole("button", { name: "关闭账号设置" }))
  await user.click(screen.getByRole("button", { name: "小米 MiMo 账号设置" }))
  await waitFor(() =>
    expect(screen.getByRole("combobox")).toHaveTextContent("Token Plan → API Key（按量）"),
  )
  expect(setMiMoPreferredBillingMode).not.toHaveBeenCalled()
  expect(client.getQueryData(["stepfun-settings"])).toEqual({ preferredBillingMode: "payg" })
  expect(client.getQueryData(["mimo-settings"])).toEqual({ preferredBillingMode: "token_plan" })
})
