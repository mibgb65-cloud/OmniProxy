import { describe, expect, it } from "vitest"
import { getCurrentReleaseNotes, versionChannel, withoutTopHeading } from "./current-release"

describe("当前版本发行说明", () => {
  it.each(["2.0.0-beta.18", "v2.0.0-beta.18", "2.0.0-beta.18+local.123"])(
    "Beta.18 按运行版本 %s 提供内存优化、周额度与本地更新边界",
    (version) => {
      const notes = getCurrentReleaseNotes(version)
      expect(notes).toContain("## Codex 中转内存优化")
      expect(notes).toContain("## 独立周额度估算与卡片精简")
      expect(notes).toContain("仍可能短暂需要约 1 GiB")
      expect(notes).toContain("不是官方美元余额")
      expect(notes).toContain("不宣称解除限流")
      expect(notes).toContain("本地安装包未签名")
    },
  )
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
  it.each(["2.0.0-beta.12", "v2.0.0-beta.12", "2.0.0-beta.12+local.123"])(
    "Beta.12 按运行版本 %s 提供刷新、留存及模型分页更新说明",
    (version) => {
      const notes = getCurrentReleaseNotes(version)
      expect(notes).toContain("## 订阅账号刷新体验")
      expect(notes).toContain("## 额度与重置时间留存")
      expect(notes).toContain("## River 模型与请求分页")
      expect(notes).toContain("DeepSeek-V4.1-Flash-1017")
      expect(notes).toContain("数据库迁移 036")
      expect(notes).toContain("本地安装包未签名")
      expect(notes).not.toContain("本版未包含额度与重置时间跨重启快照持久化")
    },
  )
  it.each(["2.0.0-beta.13", "v2.0.0-beta.13", "2.0.0-beta.13+local.123"])(
    "Beta.13 按运行版本 %s 提供 Google 自动同步及安全接入说明",
    (version) => {
      const notes = getCurrentReleaseNotes(version)
      expect(notes).toContain("## Google 模型自动同步")
      expect(notes).toContain("## Agent 接入与配置保护")
      expect(notes).toContain("不会自动扩展")
      expect(notes).toContain("不自动启动网关")
      expect(notes).toContain("本地安装包未签名")
    },
  )
  it.each(["2.0.0-beta.14", "v2.0.0-beta.14", "2.0.0-beta.14+local.123"])(
    "Beta.14 按运行版本 %s 提供美元估算及图片能力边界说明",
    (version) => {
      const notes = getCurrentReleaseNotes(version)
      expect(notes).toContain("## 额度美元参考估算")
      expect(notes).toContain("## 图片工具兼容性提示")
      expect(notes).toContain("## 启动网关偏好")
      expect(notes).toContain("不是官方美元余额")
      expect(notes).toContain("没有为文本通道新增视觉能力")
      expect(notes).toContain("数据库迁移 037")
      expect(notes).toContain("本地安装包未签名")
    },
  )
  it.each(["2.0.0-beta.15", "v2.0.0-beta.15", "2.0.0-beta.15+local.123"])(
    "Beta.15 按运行版本 %s 提供长会话、Google 留存和 River 设置说明",
    (version) => {
      const notes = getCurrentReleaseNotes(version)
      expect(notes).toContain("## Claude 长会话修复")
      expect(notes).toContain("## 额度展示与 Token 用量")
      expect(notes).toContain("## Google 额度留存")
      expect(notes).toContain("## River 使用方式与失败切换")
      expect(notes).toContain("数据库迁移 038／039")
      expect(notes).toContain("不是官方美元余额")
      expect(notes).toContain("原失败请求未保存")
      expect(notes).toContain("本地安装包未签名")
    },
  )
  it("去掉顶级版本标题，正文不变", () => {
    expect(withoutTopHeading("# App v1\n\n简介\n\n## 修复\n\n- 第一项")).toBe(
      "简介\n\n## 修复\n\n- 第一项",
    )
  })
  it("兼容 Windows 换行且不保留顶级版本标题", () => {
    expect(withoutTopHeading("# App v1\r\n\r\n## 内容\r\n\r\n- A\r\n- B")).toBe(
      "## 内容\r\n\r\n- A\r\n- B",
    )
  })
  it("没有顶级标题时保持原文", () => {
    expect(withoutTopHeading("## 内容\n\n- A")).toBe("## 内容\n\n- A")
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
