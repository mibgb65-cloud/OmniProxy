import { render, screen, within } from "@testing-library/react"
import { expect, it, vi } from "vitest"
import { I18nProvider } from "../../i18n/provider"
import { useUIStore } from "../../stores/ui-store"
import { CurrentReleaseCard } from "./current-release-card"
import { getCurrentReleaseNotes } from "./current-release"

vi.mock("./current-release", { spy: true })

function renderCard(version: string | undefined) {
  useUIStore.setState({ language: "zh" })
  return render(
    <I18nProvider>
      <CurrentReleaseCard version={version} pending={false} failed={false} />
    </I18nProvider>,
  )
}

it("按 Markdown 排版渲染随包日志，不再显示原始记号", () => {
  const { container } = renderCard("2.1.1-beta.1")
  const notes = screen.getByRole("region", { name: "当前版本更新日志" })
  expect(within(notes).getByRole("heading", { name: "新特性 / Features" })).toBeInTheDocument()
  expect(within(notes).getByText("升级浮窗三段式布局").tagName).toBe("STRONG")
  expect(within(notes).getByText("行内代码").tagName).toBe("CODE")
  const text = notes.textContent ?? ""
  expect(text).not.toContain("## 新特性")
  expect(text).not.toContain("**升级浮窗三段式布局**")
  // 卡片页眉已显示运行版本，正文不重复顶级版本标题。
  expect(text).not.toContain("OmniProxy v2.1.1-beta.1")
  expect(container.querySelector("h3")?.textContent).toBe("当前版本更新日志")
})

it("只把 http(s) 与站内链接做成入口，HTML 与不安全协议仍是文本", () => {
  const notes = [
    "# 标题",
    "",
    "- [发布页](https://example.com/release)",
    "- [站内](/docs/UPGRADE.md)",
    "- [坏](javascript:alert(1))",
    "- <script>alert(1)</script>",
  ].join("\n")
  vi.mocked(getCurrentReleaseNotes).mockReturnValue(notes)
  const { container, unmount } = renderCard("2.1.1-beta.1")
  const external = screen.getByRole("link", { name: "发布页" })
  expect(external).toHaveAttribute("href", "https://example.com/release")
  expect(external).toHaveAttribute("rel", "noreferrer")
  expect(screen.getByRole("link", { name: "站内" })).toHaveAttribute("href", "/docs/UPGRADE.md")
  expect(container.querySelectorAll("a")).toHaveLength(2)
  expect(container.querySelector("script")).toBeNull()
  expect(container.textContent).toContain("javascript:alert(1)")
  unmount()
})

it("未知版本或加载失败时给出明确提示而不是空白", () => {
  vi.mocked(getCurrentReleaseNotes).mockReturnValue(undefined)
  renderCard("9.9.9")
  expect(screen.getByText("此版本尚未附带更新日志。")).toBeInTheDocument()
})
