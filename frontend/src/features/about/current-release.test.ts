import { describe, expect, it } from "vitest"
import { getCurrentReleaseNotes, parseReleaseNotes, versionChannel } from "./current-release"

describe("当前版本发行说明", () => {
  it.each(["2.0.0-beta.7", "v2.0.0-beta.7", "2.0.0-beta.7+local.123"])(
    "按运行版本 %s 读取随包日志",
    (version) => {
      const notes = getCurrentReleaseNotes(version)
      expect(notes).toContain("## 登录修复")
      expect(notes).toContain("## 日志修复")
      expect(notes).not.toContain("## 5h／一周额度耗尽后轮换")
    },
  )
  it.each(["2.0.0-beta.8", "v2.0.0-beta.8", "2.0.0-beta.8+local.123"])(
    "Beta.8 按运行版本 %s 提供额度续期与更新说明",
    (version) => {
      const notes = getCurrentReleaseNotes(version)
      expect(notes).toContain("## 额度查询与授权续期")
      expect(notes).toContain("## 更新页面")
      expect(notes).toContain("真实历史失败本次未复现")
    },
  )
  it.each([undefined, "", "9.9.9", "../../product/PRD", "a".repeat(129)])(
    "未知或无效版本 %s 不替换为最新版本或读取任意文件",
    (version) => expect(getCurrentReleaseNotes(version)).toBeUndefined(),
  )
  it("只组织文本结构，不解释 HTML、链接或脚本", () => {
    expect(
      parseReleaseNotes(
        "# App v1\n\n简介\n\n## 修复\n\n- 第一项\n- <script>alert(1)</script>\n\n[链接](javascript:alert(1))",
      ),
    ).toEqual([
      { kind: "paragraph", text: "简介" },
      { kind: "heading", text: "修复" },
      { kind: "list", items: ["第一项", "<script>alert(1)</script>"] },
      { kind: "paragraph", text: "[链接](javascript:alert(1))" },
    ])
  })
  it("兼容 Windows 换行且不保留顶级版本标题", () => {
    expect(parseReleaseNotes("# App v1\r\n\r\n## 内容\r\n\r\n- A\r\n- B")).toEqual([
      { kind: "heading", text: "内容" },
      { kind: "list", items: ["A", "B"] },
    ])
  })
})

it.each([
  ["2.0.0", "stable"],
  ["2.0.0+beta.local", "stable"],
  ["v2.0.0-beta.7", "beta"],
  ["2.0.0-rc.1", "prerelease"],
])("运行版本 %s 显示 %s 渠道徽标", (version, channel) => {
  expect(versionChannel(version)).toBe(channel)
})
