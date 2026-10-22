import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { openProjectResource } from "../../services/project-resources"
import { useUIStore } from "../../stores/ui-store"
import { ProjectResources } from "./project-resources"

vi.mock("../../services/project-resources", () => ({ openProjectResource: vi.fn() }))

function renderResources() {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
  return render(
    <I18nProvider>
      <QueryClientProvider client={client}>
        <ProjectResources />
      </QueryClientProvider>
    </I18nProvider>,
  )
}

beforeEach(() => {
  vi.resetAllMocks()
  useUIStore.setState({ language: "zh" })
  vi.mocked(openProjectResource).mockResolvedValue()
})

describe("ProjectResources", () => {
  it.each([
    ["官网", "website"],
    ["开源仓库", "repository"],
    ["免责声明", "disclaimer"],
    ["使用协议", "terms"],
    ["隐私政策", "privacy"],
  ] as const)("%s 仅在点击时提交固定目标标识", async (label, resource) => {
    const user = userEvent.setup()
    renderResources()
    expect(openProjectResource).not.toHaveBeenCalled()
    await user.click(screen.getByRole("button", { name: label }))
    await waitFor(() => expect(openProjectResource).toHaveBeenCalledExactlyOnceWith(resource))
    expect(screen.queryByRole("alert")).not.toBeInTheDocument()
  })

  it("等待浏览器期间禁用入口，完成后恢复", async () => {
    let complete!: () => void
    vi.mocked(openProjectResource).mockImplementation(
      () => new Promise<void>((resolve) => (complete = resolve)),
    )
    const user = userEvent.setup()
    renderResources()
    await user.click(screen.getByRole("button", { name: "官网" }))
    for (const button of screen.getAllByRole("button")) expect(button).toBeDisabled()
    await user.click(screen.getByRole("button", { name: "隐私政策" }))
    expect(openProjectResource).toHaveBeenCalledTimes(1)
    complete()
    await waitFor(() => {
      for (const button of screen.getAllByRole("button")) expect(button).toBeEnabled()
    })
  })

  it.each([
    ["zh", "官网", "无法打开系统浏览器，请稍后重试。"],
    ["en", "Official website", "Could not open your browser. Please try again."],
  ] as const)("%s 安全提示失败并允许重试，不暴露底层错误", async (language, label, message) => {
    useUIStore.setState({ language })
    vi.mocked(openProjectResource)
      .mockRejectedValueOnce(new Error("private launch details"))
      .mockResolvedValueOnce()
    const user = userEvent.setup()
    renderResources()
    const button = screen.getByRole("button", { name: label })
    await user.click(button)
    expect(await screen.findByRole("alert")).toHaveTextContent(message)
    expect(screen.queryByText(/private launch details/)).not.toBeInTheDocument()
    await user.click(button)
    await waitFor(() => expect(screen.queryByRole("alert")).not.toBeInTheDocument())
    expect(openProjectResource).toHaveBeenCalledTimes(2)
  })

  it("支持键盘激活", async () => {
    const user = userEvent.setup()
    renderResources()
    await user.tab()
    expect(screen.getByRole("button", { name: "官网" })).toHaveFocus()
    await user.keyboard("{Enter}")
    await waitFor(() => expect(openProjectResource).toHaveBeenCalledExactlyOnceWith("website"))
  })
})
