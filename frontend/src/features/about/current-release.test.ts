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
  it.each(["2.0.0-beta.9", "v2.0.0-beta.9", "2.0.0-beta.9+local.123"])(
    "Beta.9 按运行版本 %s 提供 Codex 兼容修复与安装边界",
    (version) => {
      const notes = getCurrentReleaseNotes(version)
      expect(notes).toContain("## 文件变更摘要")
      expect(notes).toContain("## 上下文压缩")
      expect(notes).toContain("刷新受管模型目录")
      expect(notes).toContain("真实长会话自动压缩与 Desktop 界面仍需安装后验收")
    },
  )
  it.each([undefined, "", "9.9.9", "../../product/PRD", "a".repeat(129)])(
    "未知或无效版本 %s 不替换为最新版本或读取任意文件",
    (version) => expect(getCurrentReleaseNotes(version)).toBeUndefined(),
  )
  it.each(["2.0.0-beta.10", "v2.0.0-beta.10", "2.0.0-beta.10+local.123"])(
    "Beta.10 按运行版本 %s 提供额度修复与本地更新说明",
    (version) => {
      const notes = getCurrentReleaseNotes(version)
      expect(notes).toContain("## 额度查询超时修复")
      expect(notes).toContain("## 自动激活与安全边界")
      expect(notes).toContain("真实过期授权、上游网络及自动激活效果仍需安装后验收")
      expect(notes).toContain("本地安装包未签名")
    },
  )
  it.each(["2.0.0-beta.11", "v2.0.0-beta.11", "2.0.0-beta.11+local.123"])(
    "Beta.11 按运行版本 %s 提供长会话恢复与本地更新说明",
    (version) => {
      const notes = getCurrentReleaseNotes(version)
      expect(notes).toContain("## Codex 长会话恢复")
      expect(notes).toContain("## 授权进度与账号管理")
      expect(notes).toContain("## OpenAI 账号轮换设置")
      expect(notes).toContain("本版未包含额度与重置时间跨重启快照持久化")
      expect(notes).toContain("本地安装包未签名")
    },
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
