import { expect, it, vi } from "vitest"
import { agentError } from "./agents"

vi.mock("../../bindings/omniproxy/internal/transport/wails", () => ({ AppService: {} }))

it.each(["write_failed", "read_failed", "unsafe_config", "config_locked", "backup_failed"])(
  "保留安全配置错误 %s，避免统一误报为网关异常",
  (code) => {
    expect(agentError(new Error(`agent: ${code}`))).toBe(code)
  },
)

it("组合错误只展示稳定错误代码，不回显原始内容", () => {
  expect(agentError(new Error("agent: write_failed\nprivate diagnostic"))).toBe("write_failed")
  expect(agentError(new Error("private configuration or credentials"))).toBe("failed")
})
