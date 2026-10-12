import { describe, expect, it } from "vitest"
import { getGatewayClientConfig, getGatewayClientConnection } from "./gateway-client-configuration"

describe("客户端接入配置预览", () => {
  it.each([1, 80, 3000, 3001, 65535])("为端口 %i 生成各协议正确的地址", (port) => {
    const root = `http://127.0.0.1:${port}`
    expect(getGatewayClientConnection("codex", `${root}/v1`)).toEqual({
      baseUrl: `${root}/v1`,
      requestUrl: `${root}/v1/responses`,
    })
    expect(getGatewayClientConnection("claude", `${root}/v1`)).toEqual({
      baseUrl: root,
      requestUrl: `${root}/v1/messages`,
    })
  })

  it.each([
    null,
    "",
    "http://127.0.0.1:0/v1",
    "http://127.0.0.1:65536/v1",
    "http://127.0.0.1:03001/v1",
    "http://localhost:3001/v1",
    "https://127.0.0.1:3001/v1",
    "http://127.0.0.1:3001/v1?token=private",
    "http://127.0.0.1:3001/v1#fragment",
    "http://127.0.0.1:3001/v1\n",
    'http://127.0.0.1:3001/v1"\nmodel="injected"',
    "http://user:password@127.0.0.1:3001/v1",
  ])("拒绝异常地址 %s", (address) => {
    expect(getGatewayClientConnection("codex", address)).toBeNull()
    expect(getGatewayClientConnection("claude", address)).toBeNull()
  })

  it("Codex 模板保留内置提供商标识，以专用路径管理本地接入 Key", () => {
    const config = getGatewayClientConfig("codex", "http://127.0.0.1:3001/v1")
    expect(config).toContain(
      'openai_base_url = "http://127.0.0.1:3001/agent/codex/<gateway-api-key>/v1"',
    )
    expect(config).not.toContain("[model_providers.openai]")
    expect(config).toContain("enable_request_compression = false")
    expect(config).not.toMatch(/^model_provider\s*=/m)
  })

  it("Claude CLI JSON 使用根地址及明确的模型和网关凭证占位值", () => {
    expect(JSON.parse(getGatewayClientConfig("claude", "http://127.0.0.1:3001"))).toEqual({
      env: {
        ANTHROPIC_BASE_URL: "http://127.0.0.1:3001",
        ANTHROPIC_AUTH_TOKEN: "<gateway-token>",
        ANTHROPIC_MODEL: "<model-or-route-alias>",
      },
    })
  })
})
