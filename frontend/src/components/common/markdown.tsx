import { useMemo } from "react"
import {
  parseMarkdown,
  type InlineNode,
  type MarkdownBlock,
  type MarkdownListItem,
} from "../../lib/markdown"
import { cn } from "../../lib/utils"

const HEADING_TAGS = ["h3", "h4", "h5", "h6", "h6"] as const
const HEADING_STYLES = [
  "text-base font-semibold text-foreground",
  "text-sm font-semibold text-foreground",
  "text-sm font-medium text-foreground",
  "text-sm font-medium text-foreground",
  "text-sm font-medium text-foreground",
] as const

/** 把发布说明 Markdown 渲染成产品排版；所有内容都是 React 文本，不执行 HTML。 */
export function Markdown({ source, className }: { source: string; className?: string }) {
  const blocks = useMemo(() => parseMarkdown(source), [source])
  return (
    <div
      className={cn(
        "text-muted-foreground space-y-3 text-sm leading-relaxed [overflow-wrap:anywhere]",
        className,
      )}
    >
      {blocks.map((block, index) => (
        <Block key={index} block={block} />
      ))}
    </div>
  )
}

function Block({ block }: { block: MarkdownBlock }) {
  if (block.kind === "heading") {
    const level = Math.min(Math.max(block.level, 1), HEADING_TAGS.length) - 1
    const Heading = HEADING_TAGS[level]
    return (
      <Heading className={cn("pt-1", HEADING_STYLES[level])}>
        <Inline nodes={block.content} />
      </Heading>
    )
  }
  if (block.kind === "paragraph") {
    return (
      <p>
        <Inline nodes={block.content} />
      </p>
    )
  }
  if (block.kind === "list") {
    const List = block.ordered ? "ol" : "ul"
    return (
      <List
        className={cn(
          "space-y-1.5 pl-5",
          block.ordered ? "list-decimal" : "list-disc",
          "marker:text-muted-foreground/70",
        )}
        start={block.ordered ? block.start : undefined}
      >
        {block.items.map((item, index) => (
          <Item key={index} item={item} />
        ))}
      </List>
    )
  }
  if (block.kind === "quote") {
    return (
      <blockquote className="border-border space-y-2 border-l-2 pl-3">
        {block.children.map((child, index) => (
          <Block key={index} block={child} />
        ))}
      </blockquote>
    )
  }
  if (block.kind === "code") {
    return (
      <pre className="bg-muted/60 overflow-x-auto rounded-lg border p-3 text-xs leading-relaxed">
        <code>{block.text}</code>
      </pre>
    )
  }
  return <hr className="border-border/70" />
}

function Item({ item }: { item: MarkdownListItem }) {
  return (
    <li>
      <Inline nodes={item.content} />
      {item.children.length ? (
        <div className="mt-1.5 space-y-1.5">
          {item.children.map((child, index) => (
            <Block key={index} block={child} />
          ))}
        </div>
      ) : null}
    </li>
  )
}

function Inline({ nodes }: { nodes: InlineNode[] }) {
  return (
    <>
      {nodes.map((node, index) => (
        <InlineNodeView key={index} node={node} />
      ))}
    </>
  )
}

function InlineNodeView({ node }: { node: InlineNode }) {
  switch (node.kind) {
    case "text":
      return node.text
    case "break":
      return <br />
    case "code":
      return (
        <code className="bg-muted/70 text-foreground rounded px-1 py-0.5 font-mono text-[0.8125rem]">
          {node.text}
        </code>
      )
    case "strong":
      return (
        <strong className="text-foreground font-semibold">
          <Inline nodes={node.children} />
        </strong>
      )
    case "emphasis":
      return (
        <em>
          <Inline nodes={node.children} />
        </em>
      )
    case "delete":
      return (
        <del>
          <Inline nodes={node.children} />
        </del>
      )
    case "link":
      return (
        <a
          href={node.href}
          target="_blank"
          rel="noreferrer"
          className="text-primary font-medium [overflow-wrap:anywhere] underline underline-offset-2"
        >
          <Inline nodes={node.children} />
        </a>
      )
  }
}
