import { describe, expect, it } from "vitest"
import { isSafeHref, parseInline, parseMarkdown } from "./index"

describe("发布说明 Markdown 解析", () => {
  it("把标题、段落与分隔线拆成独立块", () => {
    expect(parseMarkdown("# v2.1.0 标题\n\n第一段\n第二行\n\n---")).toEqual([
      { kind: "heading", level: 1, content: [{ kind: "text", text: "v2.1.0 标题" }] },
      { kind: "paragraph", content: [{ kind: "text", text: "第一段 第二行" }] },
      { kind: "rule" },
    ])
  })

  it("把两级列表、加粗与行内代码拆开", () => {
    const [list] = parseMarkdown(
      "- **网关端口管理**：保存后重启\n  - **受管客户端**：`OMNIPROXY_PROXY_PORT` 优先",
    )
    expect(list.kind).toBe("list")
    if (list.kind !== "list") return
    expect(list.ordered).toBe(false)
    expect(list.items).toHaveLength(1)
    expect(list.items[0].content).toEqual([
      { kind: "strong", children: [{ kind: "text", text: "网关端口管理" }] },
      { kind: "text", text: "：保存后重启" },
    ])
    const nested = list.items[0].children[0]
    expect(nested.kind).toBe("list")
    if (nested.kind !== "list") return
    expect(nested.items[0].content).toEqual([
      { kind: "strong", children: [{ kind: "text", text: "受管客户端" }] },
      { kind: "text", text: "：" },
      { kind: "code", text: "OMNIPROXY_PROXY_PORT" },
      { kind: "text", text: " 优先" },
    ])
  })

  it("保留有序列表的起始编号", () => {
    const [list] = parseMarkdown("3. 三\n4. 四")
    expect(list.kind === "list" && list.ordered).toBe(true)
    expect(list.kind === "list" && list.start).toBe(3)
    expect(list.kind === "list" && list.items).toHaveLength(2)
  })

  it("解析围栏代码块且不把内容当作行内格式", () => {
    expect(parseMarkdown("```bash\ntask verify\n```")).toEqual([
      { kind: "code", language: "bash", text: "task verify" },
    ])
  })

  it("解析引用与硬换行", () => {
    expect(parseMarkdown("> 引用第一行\n> 引用第二行")).toEqual([
      {
        kind: "quote",
        children: [
          { kind: "paragraph", content: [{ kind: "text", text: "引用第一行 引用第二行" }] },
        ],
      },
    ])
    expect(parseMarkdown("第一行  \n第二行")).toEqual([
      {
        kind: "paragraph",
        content: [
          { kind: "text", text: "第一行" },
          { kind: "break" },
          { kind: "text", text: "第二行" },
        ],
      },
    ])
  })

  it("解析斜体、删除线与自动链接", () => {
    expect(parseInline("~~旧~~ *新* <https://example.com/a>")).toEqual([
      { kind: "delete", children: [{ kind: "text", text: "旧" }] },
      { kind: "text", text: " " },
      { kind: "emphasis", children: [{ kind: "text", text: "新" }] },
      { kind: "text", text: " " },
      {
        kind: "link",
        href: "https://example.com/a",
        children: [{ kind: "text", text: "https://example.com/a" }],
      },
    ])
  })

  it("不把词内下划线当成斜体", () => {
    expect(parseInline("snake_case_name")).toEqual([{ kind: "text", text: "snake_case_name" }])
  })

  it("未闭合的定界符保持原文", () => {
    expect(parseInline("**未闭合 与 `代码")).toEqual([{ kind: "text", text: "**未闭合 与 `代码" }])
  })

  it("只把 http(s) 与站内相对地址渲染成链接", () => {
    expect(isSafeHref("https://example.com/x")).toBe(true)
    expect(isSafeHref("/docs/UPGRADE.md")).toBe(true)
    expect(isSafeHref("javascript:alert(1)")).toBe(false)
    expect(isSafeHref("//evil.example.com")).toBe(false)
    expect(isSafeHref("")).toBe(false)
    expect(parseInline("[坏](javascript:alert(1))")).toEqual([
      { kind: "text", text: "[坏](javascript:alert(1))" },
    ])
  })

  it("把 HTML 标签当作普通文本", () => {
    expect(parseMarkdown("<script>alert(1)</script>")).toEqual([
      { kind: "paragraph", content: [{ kind: "text", text: "<script>alert(1)</script>" }] },
    ])
  })

  it("兼容 Windows 换行", () => {
    expect(parseMarkdown("# 标题\r\n\r\n- A\r\n- B")).toEqual([
      { kind: "heading", level: 1, content: [{ kind: "text", text: "标题" }] },
      {
        kind: "list",
        ordered: false,
        start: 1,
        items: [
          { content: [{ kind: "text", text: "A" }], children: [] },
          { content: [{ kind: "text", text: "B" }], children: [] },
        ],
      },
    ])
  })

  it("空说明与纯空白输入不产生块", () => {
    expect(parseMarkdown("")).toEqual([])
    expect(parseMarkdown(" \n\n ")).toEqual([])
  })
})
