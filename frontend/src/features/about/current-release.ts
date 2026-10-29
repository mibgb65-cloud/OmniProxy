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

/** 卡片页眉已显示运行版本，正文不再重复发行说明的顶级版本标题。 */
export function withoutTopHeading(source: string): string {
  return source.replace(/^\s*#[ \t]+[^\n]*(?:\r?\n)*/, "")
}
