// 行内语法：行内代码、链接、自动链接、粗体、斜体、删除线与硬换行。
// 未闭合或无法识别的定界符一律按纯文本保留，绝不解释 HTML。
import type { InlineNode } from "./types"

const CODE_SPAN = /^(`+)([\s\S]*?)\1/
const IMAGE = /^!\[([^\]]*)\]\(([^()\s]*)\)/
const LINK = /^\[([^\]]*)\]\(\s*<?([^()\s>]+)>?(?:\s+"[^"]*")?\s*\)/
const AUTOLINK = /^<(https?:\/\/[^\s<>]+)>/
const BARE_URL = /^https?:\/\/[^\s<>()[\]]+/u
const ESCAPABLE = /^[\\`*_{}[\]()#+\-.!~>|]$/
const WORD_CHAR = /[\p{L}\p{N}]/u
const TRAILING_PUNCTUATION = /[.,;:!?、。）】"'»]+$/u

type EmphasisKind = "strong" | "emphasis" | "delete"

const EMPHASIS: Record<string, { pattern: RegExp; kind: EmphasisKind }[]> = {
  "*": [
    { pattern: /^\*\*([\s\S]+?)\*\*/, kind: "strong" },
    { pattern: /^\*([\s\S]+?)\*/, kind: "emphasis" },
  ],
  _: [
    { pattern: /^__([\s\S]+?)__/, kind: "strong" },
    { pattern: /^_([\s\S]+?)_/, kind: "emphasis" },
  ],
  "~": [{ pattern: /^~~([\s\S]+?)~~/, kind: "delete" }],
}

/** 只放行 http(s) 与站内相对地址；`javascript:` 等协议保持纯文本。 */
export function isSafeHref(href: string): boolean {
  const value = href.trim()
  if (!value) return false
  if (value.startsWith("/") && !value.startsWith("//")) return true
  return /^https?:\/\/[^\s]+$/i.test(value)
}

/** 把一段行内 Markdown 源码解析为纯数据节点。 */
export function parseInline(source: string): InlineNode[] {
  const nodes: InlineNode[] = []
  let text = ""
  let index = 0
  const flush = () => {
    if (!text) return
    nodes.push({ kind: "text", text })
    text = ""
  }
  while (index < source.length) {
    const rest = source.slice(index)
    const char = source[index]
    if (char === "\\" && ESCAPABLE.test(source[index + 1] ?? "")) {
      text += source[index + 1]
      index += 2
      continue
    }
    if (char === "\n") {
      flush()
      nodes.push({ kind: "break" })
      index += 1
      continue
    }
    if (char === "`") {
      const match = CODE_SPAN.exec(rest)
      if (match) {
        flush()
        nodes.push({ kind: "code", text: trimCodeSpan(match[2]) })
        index += match[0].length
        continue
      }
    }
    if (char === "!" || char === "[") {
      const image = char === "!" ? IMAGE.exec(rest) : null
      if (image) {
        // 不加载远端图片，避免发布说明触发外部请求；只保留替代文字。
        flush()
        if (image[1] || image[2]) nodes.push({ kind: "text", text: image[1] || image[2] })
        index += image[0].length
        continue
      }
      const link = LINK.exec(rest)
      if (link && isSafeHref(link[2])) {
        flush()
        nodes.push({ kind: "link", href: link[2].trim(), children: parseInline(link[1]) })
        index += link[0].length
        continue
      }
    }
    if (char === "<") {
      const match = AUTOLINK.exec(rest)
      if (match) {
        flush()
        nodes.push({ kind: "link", href: match[1], children: [{ kind: "text", text: match[1] }] })
        index += match[0].length
        continue
      }
    }
    if (char === "*" || char === "_" || char === "~") {
      const emphasis = matchEmphasis(rest, char, source[index - 1])
      if (emphasis) {
        flush()
        nodes.push(emphasisNode(emphasis.kind, emphasis.text))
        index += emphasis.length
        continue
      }
    }
    if (rest.startsWith("http://") || rest.startsWith("https://")) {
      const match = BARE_URL.exec(rest)
      if (match) {
        const href = match[0].replace(TRAILING_PUNCTUATION, "")
        if (isSafeHref(href)) {
          flush()
          nodes.push({ kind: "link", href, children: [{ kind: "text", text: href }] })
          index += href.length
          continue
        }
      }
    }
    text += char
    index += 1
  }
  flush()
  return nodes
}

function emphasisNode(kind: EmphasisKind, source: string): InlineNode {
  const children = parseInline(source)
  if (kind === "delete") return { kind: "delete", children }
  return kind === "strong" ? { kind: "strong", children } : { kind: "emphasis", children }
}

function matchEmphasis(
  rest: string,
  delimiter: string,
  previous: string | undefined,
): { kind: EmphasisKind; text: string; length: number } | undefined {
  // 词内下划线（如 snake_case）不是斜体，避免把标识符拆碎。
  if (delimiter === "_" && previous && WORD_CHAR.test(previous)) return undefined
  for (const rule of EMPHASIS[delimiter] ?? []) {
    const match = rule.pattern.exec(rest)
    if (!match) continue
    const inner = match[1]
    if (!inner.trim() || /^\s/.test(inner) || /\s$/.test(inner)) continue
    return { kind: rule.kind, text: inner, length: match[0].length }
  }
  return undefined
}

/** 行内代码两端各保留一个空格时按规范去掉一个。 */
function trimCodeSpan(code: string): string {
  if (code.length > 2 && code.startsWith(" ") && code.endsWith(" ") && code.trim()) {
    return code.slice(1, -1)
  }
  return code
}
