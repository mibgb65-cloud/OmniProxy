export const gatewayEndpointPaths = {
  openai: '/v1',
  codex: '/codex/v1',
  zo: '/zo/v1',
  prem: '/prem/v1',
  openrouter: '/openrouter/v1',
  featherless: '/featherless/v1',
  tokenrouter: '/tokenrouter/v1',
  sub2api: '/sub2api/v1',
  newapi: '/newapi/v1',
  anyrouter: '/anyrouter/v1',
  custom: '/custom/v1',
  claudeRouter: '/anthropic-router',
  anthropic: '/anthropic/v1',
  zoAnthropic: '/zo/v1',
  premAnthropic: '/prem/anthropic/v1',
  atria: '/atria/v1',
  atriaAnthropic: '/atria/anthropic/v1',
  sub2apiAnthropic: '/sub2api/anthropic/v1',
  newapiAnthropic: '/newapi/anthropic/v1',
  anyrouterAnthropic: '/anyrouter/anthropic/v1',
  customAnthropic: '/custom/anthropic/v1',
  deepseek: '/deepseek/v1',
  kimi: '/kimi/v1',
  xiaomi: '/xiaomi/v1',
  zhipu: '/zhipu/v1',
  minimax: '/minimax/v1',
  gemini: '/gemini',
  sub2apiGemini: '/sub2api/gemini',
  newapiGemini: '/newapi/gemini',
  opencodeRouter: '/opencode-router/v1',
  piRouter: '/pi-router/v1',
  claudeDesktop: '/claude-desktop',
}

export function localProxyBaseURL(port) {
  return `http://127.0.0.1:${Number(port) || 3000}`
}

export function localProxyEndpoint(port, path) {
  return `${localProxyBaseURL(port)}${path}`
}
