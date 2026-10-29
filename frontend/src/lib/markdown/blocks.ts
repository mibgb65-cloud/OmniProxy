// 块级语法：标题、段落、列表（含嵌套与有序）、引用、围栏代码块与分隔线。
// 说明文字里的 HTML 标签没有对应分支，因此只会作为段落文本出现。
import { parseInline } from "./inline"
import type { MarkdownBlock, MarkdownListItem } from "./types"

const FENCE = /^\s{0,3}(`{3,}|~{3,})\s*([^\s`]*)\s*$/
const HEADING = /^\s{0,3}(#{1,6})\s+(.*?)\s*#*\s*$/
const RULE = /^\s{0,3}(?:(?:\*\s*){3,}|(?:-\s*){3,}|(?:_\s*){3,})$/
const QUOTE = /^\s{0,3}>\s?(.*)$/
const ITEM = /^(\s*)([-*+]|\d{1,9}[.)])\s+(.*)$/
const HARD_BREAK = /(?: {2,}|\\)$/

/** 把发布说明源码解析为纯数据块序列。 */
export function parseMarkdown(source: string): MarkdownBlock[] {
  return parseBlocks(source.replace(/\r\n?/g, "\n").split("\n"))
}

function parseBlocks(lines: string[]): MarkdownBlock[] {
  const blocks: MarkdownBlock[] = []
  let index = 0
  while (index < lines.length) {
    const line = lines[index]
    if (!line.trim()) {
      index += 1
      continue
    }
    const fence = FENCE.exec(line)
    if (fence) {
      const closing = new RegExp(`^\\s{0,3}${fence[1][0]}{3,}\\s*$`)
      const body: string[] = []
      index += 1
      while (index < lines.length && !closing.test(lines[index])) {
        body.push(lines[index])
        index += 1
      }
      blocks.push({ kind: "code", language: fence[2] ?? "", text: body.join("\n") })
      index += 1
      continue
    }
    const heading = HEADING.exec(line)
    if (heading) {
      blocks.push({ kind: "heading", level: heading[1].length, content: parseInline(heading[2]) })
      index += 1
      continue
    }
    if (RULE.test(line)) {
      blocks.push({ kind: "rule" })
      index += 1
      continue
    }
    if (QUOTE.test(line)) {
      const inner: string[] = []
      while (index < lines.length && QUOTE.test(lines[index])) {
        inner.push(QUOTE.exec(lines[index])?.[1] ?? "")
        index += 1
      }
      blocks.push({ kind: "quote", children: parseBlocks(inner) })
      continue
    }
    if (ITEM.test(line)) {
      index = parseList(lines, index, blocks)
      continue
    }
    const paragraph: string[] = []
    while (index < lines.length && lines[index].trim() && !startsBlock(lines[index])) {
      paragraph.push(lines[index])
      index += 1
    }
    blocks.push({ kind: "paragraph", content: parseInline(joinParagraph(paragraph)) })
  }
  return blocks
}

function startsBlock(line: string): boolean {
  return (
    FENCE.test(line) || HEADING.test(line) || RULE.test(line) || QUOTE.test(line) || ITEM.test(line)
  )
}

function parseList(lines: string[], start: number, blocks: MarkdownBlock[]): number {
  const head = ITEM.exec(lines[start])
  if (!head) return start + 1
  const baseIndent = head[1].length
  const ordered = /\d/.test(head[2])
  const items: MarkdownListItem[] = []
  let index = start
  while (index < lines.length) {
    const line = lines[index]
    if (!line.trim()) {
      const next = lines[index + 1]
      if (!next || !ITEM.test(next) || indentOf(next) < baseIndent) break
      index += 1
      continue
    }
    const marker = ITEM.exec(line)
    if (!marker || marker[1].length !== baseIndent || /\d/.test(marker[2]) !== ordered) break
    const textLines = [marker[3]]
    const nestedLines: string[] = []
    index += 1
    while (index < lines.length) {
      const current = lines[index]
      if (!current.trim()) {
        const next = lines[index + 1]
        const deeper = next ? indentOf(next) > baseIndent : false
        if (!next || !deeper) break
        nestedLines.push("")
        index += 1
        continue
      }
      if (ITEM.test(current) && indentOf(current) <= baseIndent) break
      if (indentOf(current) > baseIndent + 1) {
        nestedLines.push(current.trim())
        index += 1
        continue
      }
      // 与标记同缩进的续行属于本条目正文。
      textLines.push(current.trim())
      index += 1
    }
    items.push({
      content: parseInline(joinParagraph(textLines)),
      children: nestedLines.length ? parseBlocks(nestedLines) : [],
    })
  }
  blocks.push({ kind: "list", ordered, start: ordered ? Number.parseInt(head[2], 10) : 1, items })
  return index
}

function indentOf(line: string): number {
  return /^\s*/.exec(line)?.[0].length ?? 0
}

/** 段落内的软换行合并为空格；行尾两个空格或反斜杠按硬换行处理。 */
function joinParagraph(lines: string[]): string {
  const parts: string[] = []
  for (const line of lines) {
    const hard = HARD_BREAK.test(line)
    const text = line.trim().replace(HARD_BREAK, "")
    parts.push(text, hard ? "\n" : " ")
  }
  return parts.join("").trimEnd()
}
