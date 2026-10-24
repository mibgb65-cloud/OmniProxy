import { describe, expect, it } from "vitest"
import { gatewayErrorKey, gatewayOperationErrorKey } from "./gateway-errors"

describe("WebSocket 握手错误分类", () => {
  it.each(["request_timeout", "credential_prepare_timeout", "upstream_socket_timeout"])(
    "将 %s 显示为超时而不是凭据损坏",
    (code) => {
      expect(gatewayErrorKey(code)).toBe("gateway.error.timeout")
      expect(gatewayOperationErrorKey(new Error(`gateway: ${code}`))).toBe("gateway.error.timeout")
    },
  )

  it("不把未知底层错误显示到界面", () => {
    expect(gatewayOperationErrorKey(new Error("private-credential-material"))).toBe(
      "gateway.error.unknown",
    )
  })
})
