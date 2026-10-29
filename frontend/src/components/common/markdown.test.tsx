import { render, screen } from "@testing-library/react"
import { expect, it } from "vitest"
import { Markdown } from "./markdown"

it("标题、加粗、行内代码与安全链接按语义渲染，不显示 Markdown 记号", () => {
  const { container } = render(
    <Markdown
      source={
        "## 新特性\n\n- **网关端口管理**：`OMNIPROXY_PROXY_PORT`\n  - [发布页](https://example.com/release)"
      }
    />,
  )
  expect(screen.getByRole("heading", { name: "新特性" })).toBeInTheDocument()
  expect(screen.getByText("网关端口管理").tagName).toBe("STRONG")
  expect(screen.getByText("OMNIPROXY_PROXY_PORT").tagName).toBe("CODE")
  const link = screen.getByRole("link", { name: "发布页" })
  expect(link).toHaveAttribute("href", "https://example.com/release")
  expect(link).toHaveAttribute("target", "_blank")
  expect(link).toHaveAttribute("rel", "noreferrer")
  expect(container.textContent).not.toContain("##")
  expect(container.textContent).not.toContain("**")
})

it("HTML 标签与不安全链接只作为文本出现，不产生脚本或链接元素", () => {
  const { container } = render(
    <Markdown source={"[坏](javascript:alert(1)) 与 <script>alert(1)</script>"} />,
  )
  expect(container.querySelector("script")).toBeNull()
  expect(container.querySelector("a")).toBeNull()
  expect(container.textContent).toContain("javascript:alert(1)")
  expect(container.textContent).toContain("<script>alert(1)</script>")
})

it("列表渲染可识别的列表与代码块结构", () => {
  const { container } = render(<Markdown source={"- 一\n- 二\n\n```bash\ntask verify\n```"} />)
  expect(screen.getAllByRole("listitem")).toHaveLength(2)
  expect(container.querySelector("ol")).toBeNull()
  expect(container.querySelector("pre code")?.textContent).toBe("task verify")
})
