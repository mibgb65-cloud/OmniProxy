// 构建时收录已审阅的本地发行说明，不因打开设置页而联网，也不混用更新候选的日志。
const releases = import.meta.glob<string>("../../../../docs/releases/v*.md", {
  query: "?raw",
  import: "default",
  eager: true,
})

export function getCurrentReleaseNotes(version: string | undefined): string | undefined {
  if (!version || version.length > 128) return undefined
  const normalized = version.replace(/^v/, "").split("+", 1)[0]
  return releases[`../../../../docs/releases/v${normalized}.md`]
}

export function versionChannel(version: string): "stable" | "beta" | "prerelease" {
  const base = version.split("+", 1)[0]
  if (!base.includes("-")) return "stable"
  return /-beta(?:[.-]|$)/.test(base) ? "beta" : "prerelease"
}

export type ReleaseNoteBlock =
  { kind: "heading" | "paragraph"; text: string } | { kind: "list"; items: string[] }

// 仅将仓库说明中的标题、段落和无序列表结构化；正文始终是 React 文本，不解析 HTML 或执行链接。
export function parseReleaseNotes(source: string): ReleaseNoteBlock[] {
  return source
    .trim()
    .split(/\r?\n\s*\r?\n/)
    .flatMap((block): ReleaseNoteBlock[] => {
      if (/^# /.test(block)) return []
      if (/^#{2,6} /.test(block)) {
        return [{ kind: "heading", text: block.replace(/^#{2,6} /, "") }]
      }
      const lines = block.split(/\r?\n/)
      if (lines.every((line) => /^[-*] /.test(line))) {
        return [{ kind: "list", items: lines.map((line) => line.slice(2)) }]
      }
      return [{ kind: "paragraph", text: block }]
    })
}
