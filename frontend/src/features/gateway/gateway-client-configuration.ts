import { getConfiguredProxyAddress } from "./gateway-preparation"

export type GatewayClient = "codex" | "claude"

export const gatewayClientDefinitions = {
  codex: {
    title: "gateway.clients.codex.title",
    description: "gateway.clients.codex.description",
    protocol: "Responses API",
    docs: "https://learn.chatgpt.com/docs/enterprise/connect-to-a-gateway",
  },
  claude: {
    title: "gateway.clients.claude.title",
    description: "gateway.clients.claude.description",
    protocol: "Anthropic Messages",
    docs: "https://code.claude.com/docs/en/llm-gateway-connect",
  },
} as const

/** 模板只接受应用生成的规范回环地址，防止把异常配置拼进客户端配置文本。 */
export function getGatewayClientConnection(client: GatewayClient, address: string | null) {
  if (typeof address !== "string") return null
  const match = /^http:\/\/127\.0\.0\.1:(\d{1,5})\/v1$/.exec(address)
  if (!match) return null
  const root = getConfiguredProxyAddress(Number(match[1]))
  if (!root || `${root}/v1` !== address) return null

  return {
    baseUrl: client === "codex" ? address : root,
    requestUrl: `${root}/v1/${client === "codex" ? "responses" : "messages"}`,
  }
}

/** 手动接入摘要不含密钥；实际配置根据已有提供商身份由后端合并。 */
export function getGatewayClientConfig(client: GatewayClient, baseUrl: string) {
  if (client === "claude") {
    return JSON.stringify(
      {
        env: {
          ANTHROPIC_BASE_URL: baseUrl,
          ANTHROPIC_AUTH_TOKEN: "<gateway-token>",
          ANTHROPIC_MODEL: "<model-or-route-alias>",
        },
      },
      null,
      2,
    )
  }

  return [
    'model = "<model-or-route-alias>"',
    "# 保留原有 model_provider 和官方登录；内置 openai 不能重定义。",
    `openai_base_url = ${JSON.stringify(baseUrl.replace(/\/v1$/, "/agent/codex/<gateway-api-key>/v1"))}`,
    'web_search = "disabled"',
    "",
    "[features]",
    "enable_request_compression = false",
  ].join("\n")
}
