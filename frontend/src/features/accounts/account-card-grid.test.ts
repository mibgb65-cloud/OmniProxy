import { readFileSync } from "node:fs"
import { describe, expect, it } from "vitest"

/**
 * 账号卡片网格的列宽规则是纯 CSS，jsdom 不会做布局计算，
 * 这里直接读样式源码，锁住「按可用宽度自动增列 + 单卡宽度封顶」这两个约束。
 */
const readCss = (name: string) => readFileSync(new URL(name, import.meta.url), "utf8")

const gridCss = readCss("./account-card-grid.css")
const bentoCss = readCss("./bento-accounts.css")

/** 取出所有匹配选择器的声明块；找不到时返回空数组，让断言给出明确失败。 */
function declarations(css: string, selector: string) {
  const escaped = selector.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
  return [...css.matchAll(new RegExp(`${escaped}\\s*\\{([^}]*)\\}`, "g"))].map((match) => match[1])
}

describe("账号卡片网格的列宽", () => {
  it("基础网格按可用宽度自动增列，并给单张卡片设上限", () => {
    const rules = declarations(gridCss, ".account-card-grid")[0] ?? ""
    expect(rules).toContain("auto-fill")
    expect(rules).toContain("--account-card-max-width: 28rem")
    expect(rules).toMatch(
      /clamp\(\s*var\(--account-card-min-width\),[\s\S]*var\(--account-card-max-width\)\s*\)/,
    )
  })

  it("展开面板只覆盖间距，不固定列数", () => {
    const rules = declarations(bentoCss, ".bento-panel .account-card-grid")
    expect(rules[0]).toContain("--account-card-gap")
    // 固定列数（repeat(<数字>, …)）会让宽屏单卡拉成半个屏幕宽。
    expect(bentoCss).not.toMatch(/\.bento-panel \.account-card-grid[^}]*repeat\(\s*\d/)
  })

  it("面板里的列宽覆盖都带上限，不会无界拉伸", () => {
    const overrides = declarations(bentoCss, ".bento-panel .account-card-grid")
      .map((rules) => rules.match(/grid-template-columns:\s*([^;]+);/)?.[1] ?? "")
      .filter(Boolean)
    expect(overrides.length).toBeGreaterThan(0)
    for (const value of overrides) {
      expect(value).toContain("var(--account-card-max-width)")
    }
  })
})
