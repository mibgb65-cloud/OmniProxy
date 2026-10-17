import { expect, it } from "vitest"
import { agentInstallationError } from "./agent-installation-error"

it.each([
  "update_check_failed",
  "update_unsupported",
  "update_unavailable",
  "update_failed",
  "update_interrupted",
  "busy",
  "detection_failed",
])("保留明确的 %s 错误码", (code) => {
  expect(agentInstallationError(new Error(`agentinstall: ${code}`))).toBe(code)
  expect(agentInstallationError(`agentinstall: ${code}`)).toBe(code)
})

it("未知错误不会暴露响应、路径或凭据", () => {
  expect(agentInstallationError(new Error("private transport detail"))).toBe("failed")
  expect(
    agentInstallationError(new Error("agentinstall: update_failed\nprivate transport detail")),
  ).toBe("failed")
})
