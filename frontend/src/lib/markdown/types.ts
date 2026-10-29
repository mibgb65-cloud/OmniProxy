/** 发布说明的纯数据 AST：只描述结构，样式交给渲染层，绝不携带 HTML。 */

export type InlineNode =
  | { kind: "text"; text: string }
  | { kind: "break" }
  | { kind: "code"; text: string }
  | { kind: "strong"; children: InlineNode[] }
  | { kind: "emphasis"; children: InlineNode[] }
  | { kind: "delete"; children: InlineNode[] }
  | { kind: "link"; href: string; children: InlineNode[] }

export type MarkdownListItem = {
  content: InlineNode[]
  /** 列表项内的嵌套块（下一级列表、续行段落等）。 */
  children: MarkdownBlock[]
}

export type MarkdownBlock =
  | { kind: "heading"; level: number; content: InlineNode[] }
  | { kind: "paragraph"; content: InlineNode[] }
  | { kind: "list"; ordered: boolean; start: number; items: MarkdownListItem[] }
  | { kind: "quote"; children: MarkdownBlock[] }
  | { kind: "code"; language: string; text: string }
  | { kind: "rule" }
