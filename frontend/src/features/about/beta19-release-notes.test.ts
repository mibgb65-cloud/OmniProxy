import { expect, it } from "vitest"
import { getCurrentReleaseNotes } from "./current-release"

it.each(["2.0.0-beta.19", "v2.0.0-beta.19", "2.0.0-beta.19+local.123"])(
  "Beta.19 按运行版本 %s 提供 Fast、正式协议及安全更新说明",
  (version) => {
    const notes = getCurrentReleaseNotes(version)
    expect(notes).toContain("# OmniProxy v2.0.0-beta.19")
    expect(notes).toContain("## Codex Fast 模式入口修复")
    expect(notes).toContain("## 项目与协议正式入口")
    expect(notes).toContain("同步官方模型")
    expect(notes).toContain("不自动开启 Fast")
    expect(notes).toContain("不宣称解除限流")
    expect(notes).toContain("本地安装包未签名")
    expect(notes).toContain("真实覆盖安装与数据保留由用户本地验收")
    expect(notes).not.toContain("## Codex 中转内存优化")
  },
)
