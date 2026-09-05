import { gatewayEndpointPaths, localProxyEndpoint } from '../../constants/gatewayEndpoints.js'

const openAICompatibleProviders = [
  'openai',
  'deepseek',
  'kimi',
  'xiaomi',
  'zhipu',
  'minimax',
  'openrouter',
  'tokenrouter',
  'sub2api',
  'newapi',
  'anyrouter',
  'zo',
  'prem',
  'custom',
]

export const routeDefinitions = [
  {
    key: 'codex',
    title: 'Codex',
    protocol: 'OpenAI Responses',
    endpoint: (port) => localProxyEndpoint(port, gatewayEndpointPaths.codex),
    fallback: { provider: 'openai', credentialType: '', model: 'gpt-5.6-sol' },
    providers: openAICompatibleProviders,
    modelPresets: ['gpt-6-astra', 'gpt-5.6-sol', 'gpt-5.6-terra', 'gpt-5.6-luna', 'gpt-5.5', 'gpt-5.4', 'gpt-5.4-mini', 'gpt-5.6-sol-high', 'gpt-5.4-high', 'gpt-5.5-high', 'gpt-5-codex', 'deepseek-v4-pro', 'deepseek-v4-flash', 'mimo-v2.5-pro', 'kimi-for-coding', 'glm-5.1', 'MiniMax-M2.7'],
  },
  {
    key: 'claude',
    title: 'Claude Code / Desktop',
    protocol: 'Anthropic Messages',
    endpoint: (port) => localProxyEndpoint(port, gatewayEndpointPaths.claudeRouter),
    fallback: { provider: 'anthropic', credentialType: '', model: 'default' },
    providers: [
      'anthropic',
      'deepseek',
      'kimi',
      'xiaomi',
      'zhipu',
      'minimax',
      'sub2api',
      'newapi',
      'anyrouter',
      'zo',
      'prem',
      'custom',
    ],
    modelPresets: [
      'default',
      'sonnet',
      'opus',
      'haiku',
      'claude-opus-4-7',
      'claude-sonnet-4-6',
      'deepseek-v4-pro',
      'deepseek-v4-pro[1m]',
      'mimo-v2.5-pro[1m]',
      'kimi-for-coding',
      'glm-5.1',
    ],
  },
  {
    key: 'openai',
    title: 'OpenAI 兼容',
    protocol: 'Chat / Responses',
    endpoint: (port) => localProxyEndpoint(port, gatewayEndpointPaths.opencodeRouter),
    fallback: { provider: 'openai', credentialType: '', model: 'gpt-5.6-terra' },
    providers: openAICompatibleProviders,
    modelPresets: ['gpt-6-astra', 'gpt-5.6-terra', 'gpt-5.6-sol', 'gpt-5.6-luna', 'gpt-5.4', 'gpt-5.4-high', 'gpt-5.5', 'gpt-5.5-high', 'deepseek-v4-pro', 'kimi-for-coding', 'glm-5.1', 'MiniMax-M2.7'],
  },
  {
    key: 'gemini',
    title: 'Gemini CLI',
    protocol: 'Gemini Native',
    endpoint: (port) => localProxyEndpoint(port, gatewayEndpointPaths.gemini),
    fallback: { provider: 'gemini', credentialType: '', model: 'gemini-3-pro-preview' },
    providers: ['gemini', 'sub2api', 'newapi'],
    modelPresets: ['gemini-3-pro-preview', 'gemini-3-flash-preview', 'gemini-2.5-pro', 'gemini-2.5-flash'],
  },
]

export const gatewayPlatformPresets = [
  {
    key: 'openai',
    routeCredentials: { codex: 'codex_auth_json', openai: 'api_key' },
    models: [
      routeModel('gpt-6-astra', ['codex', 'openai'], 'GPT-6 Astra'),
      routeModel('gpt-5.6-sol', ['codex', 'openai'], 'GPT-5.6 Sol'),
      routeModel('gpt-5.6-terra', ['codex', 'openai'], 'GPT-5.6 Terra'),
      routeModel('gpt-5.6-luna', ['codex', 'openai'], 'GPT-5.6 Luna'),
      routeModel('gpt-5.5', ['codex', 'openai']),
      routeModel('gpt-5.4', ['codex', 'openai']),
      routeModel('gpt-5.4-mini', ['codex', 'openai']),
      routeModel('gpt-5.4-high', ['codex', 'openai']),
      routeModel('gpt-5-codex', ['codex', 'openai']),
    ],
  },
  {
    key: 'anthropic',
    routeCredentials: { claude: 'api_key' },
    models: [
      routeModel('default', ['claude'], 'Claude Default'),
      routeModel('sonnet', ['claude'], 'Claude Sonnet'),
      routeModel('opus', ['claude'], 'Claude Opus'),
      routeModel('haiku', ['claude'], 'Claude Haiku'),
    ],
  },
  {
    key: 'deepseek',
    models: [
      routeModel('deepseek-v4-pro', ['codex', 'openai', 'claude'], 'DeepSeek V4 Pro'),
      {
        id: 'deepseek-v4-pro-1m',
        label: 'DeepSeek V4 Pro [1m]',
        routeModels: { claude: 'deepseek-v4-pro[1m]' },
      },
      routeModel('deepseek-v4-flash', ['codex', 'openai', 'claude'], 'DeepSeek V4 Flash'),
    ],
  },
  {
    key: 'kimi',
    models: [
      routeModel('kimi-for-coding', ['codex', 'openai', 'claude'], 'Kimi for Coding'),
    ],
  },
  {
    key: 'xiaomi',
    models: [
      {
        id: 'mimo-v2.5-pro',
        label: 'MiMo V2.5 Pro',
        routeModels: { codex: 'mimo-v2.5-pro', openai: 'mimo-v2.5-pro', claude: 'mimo-v2.5-pro[1m]' },
      },
      routeModel('mimo-v2.5', ['openai', 'claude'], 'MiMo V2.5'),
    ],
  },
  {
    key: 'zhipu',
    models: [
      routeModel('glm-5.1', ['codex', 'openai', 'claude'], 'GLM-5.1'),
    ],
  },
  {
    key: 'minimax',
    models: [
      routeModel('MiniMax-M2.7', ['codex', 'openai', 'claude'], 'MiniMax M2.7'),
    ],
  },
  {
    key: 'gemini',
    routeCredentials: { gemini: 'api_key' },
    models: [
      routeModel('gemini-3-pro-preview', ['gemini'], 'Gemini 3 Pro Preview'),
      routeModel('gemini-3-flash-preview', ['gemini'], 'Gemini 3 Flash Preview'),
      routeModel('gemini-2.5-pro', ['gemini'], 'Gemini 2.5 Pro'),
      routeModel('gemini-2.5-flash', ['gemini'], 'Gemini 2.5 Flash'),
    ],
  },
  {
    key: 'openrouter',
    routeCredentials: { codex: 'api_key', openai: 'api_key' },
    models: [
      routeModel('openrouter/auto', ['codex', 'openai'], 'OpenRouter Auto'),
      routeModel('anthropic/claude-sonnet-4.5', ['openai'], 'Claude Sonnet via OpenRouter'),
      routeModel('openai/gpt-5.4', ['codex', 'openai'], 'GPT-5.4 via OpenRouter'),
    ],
  },
  {
    key: 'tokenrouter',
    routeCredentials: { codex: 'api_key', openai: 'api_key' },
    models: [
      routeModel('auto:balance', ['codex', 'openai'], 'Auto Balance'),
      routeModel('auto:quality', ['codex', 'openai'], 'Auto Quality'),
      routeModel('auto:speed', ['codex', 'openai'], 'Auto Speed'),
      routeModel('auto:cost', ['codex', 'openai'], 'Auto Cost'),
    ],
  },
  {
    key: 'sub2api',
    routeCredentials: { codex: 'api_key', openai: 'api_key', claude: 'api_key', gemini: 'api_key' },
    models: [
      routeModel('gpt-5.4', ['codex', 'openai']),
      routeModel('claude-sonnet-4-5', ['claude'], 'Claude Sonnet 4.5'),
      routeModel('gemini-3-pro-preview', ['gemini'], 'Gemini 3 Pro Preview'),
    ],
  },
  {
    key: 'newapi',
    routeCredentials: { codex: 'api_key', openai: 'api_key', claude: 'api_key', gemini: 'api_key' },
    models: [
      routeModel('gpt-5.4', ['codex', 'openai']),
      routeModel('claude-sonnet-4-5', ['claude'], 'Claude Sonnet 4.5'),
      routeModel('gemini-3-pro-preview', ['gemini'], 'Gemini 3 Pro Preview'),
    ],
  },
  {
    key: 'anyrouter',
    routeCredentials: { codex: 'api_key', openai: 'api_key', claude: 'api_key' },
    models: [
      routeModel('gpt-5-codex', ['codex', 'openai'], 'GPT-5 Codex'),
      routeModel('claude-opus-4-5-20251101', ['claude'], 'Claude Opus 4.5'),
    ],
  },
  {
    key: 'zo',
    routeCredentials: { codex: 'api_key', openai: 'api_key', claude: 'api_key' },
    models: [
      routeModel('gpt-5.4', ['codex', 'openai'], 'Zo GPT-5.4'),
      routeModel('claude-opus-4-7', ['claude'], 'Zo Claude Opus 4.7'),
      routeModel('claude-sonnet-4-6', ['claude'], 'Zo Claude Sonnet 4.6'),
    ],
  },
  {
    key: 'prem',
    routeCredentials: { codex: 'api_key', openai: 'api_key', claude: 'api_key' },
    models: [
      routeModel('deepseek-v4-pro', ['codex', 'openai', 'claude'], 'DeepSeek V4 Pro'),
      {
        id: 'prem-deepseek-v4-pro-1m',
        label: 'DeepSeek V4 Pro [1m]',
        routeModels: { claude: 'deepseek-v4-pro[1m]' },
      },
      routeModel('qwen3.5', ['codex', 'openai'], 'Qwen 3.5'),
    ],
  },
  {
    key: 'custom',
    routeCredentials: { codex: 'api_key', openai: 'api_key', claude: 'api_key' },
    models: [
      routeModel('custom-model', ['codex', 'openai', 'claude'], 'Custom Model'),
    ],
  },
]

export const routeStrategyPresets = [
  {
    key: 'stable',
    label: '稳定优先',
    description: '官方与主流托管优先',
    providers: ['openai', 'anthropic', 'gemini', 'deepseek', 'kimi', 'zhipu', 'minimax', 'openrouter', 'tokenrouter', 'prem', 'custom'],
  },
  {
    key: 'cost',
    label: '成本优先',
    description: '低成本和聚合网关优先',
    providers: ['deepseek', 'kimi', 'xiaomi', 'zhipu', 'minimax', 'tokenrouter', 'openrouter', 'sub2api', 'newapi', 'anyrouter', 'prem', 'custom', 'openai', 'anthropic', 'gemini'],
  },
  {
    key: 'speed',
    label: '速度优先',
    description: '本地和高速中转优先',
    providers: ['prem', 'zo', 'openai', 'anthropic', 'gemini', 'tokenrouter', 'openrouter', 'deepseek', 'kimi', 'xiaomi', 'zhipu', 'minimax', 'custom'],
  },
  {
    key: 'quota',
    label: '额度轮转',
    description: '尽量保留更多备用链',
    providers: ['openai', 'deepseek', 'kimi', 'xiaomi', 'zhipu', 'minimax', 'anthropic', 'gemini', 'openrouter', 'tokenrouter', 'sub2api', 'newapi', 'anyrouter', 'zo', 'prem', 'custom'],
  },
]

export function routeStrategyChain(availableProviders, strategyKey) {
  const available = Array.from(new Set((availableProviders || []).map(normalizeProviderKey).filter(Boolean)))
  const preset = routeStrategyPresets.find((item) => item.key === strategyKey) || routeStrategyPresets[0]
  const ordered = [
    ...preset.providers.filter((provider) => available.includes(provider)),
    ...available.filter((provider) => !preset.providers.includes(provider)),
  ]
  return Array.from(new Set(ordered))
}

export function inferGatewayProviderForModel(model) {
  const normalized = String(model || '').trim().toLowerCase()
  if (normalized === 'claude-opus-4-7' || normalized === 'claude-sonnet-4-6') return 'zo'
  if (normalized.startsWith('claude-')) return 'anthropic'
  if (normalized.startsWith('deepseek-')) return 'deepseek'
  if (normalized.startsWith('kimi-')) return 'kimi'
  if (normalized.startsWith('mimo-')) return 'xiaomi'
  if (normalized.startsWith('glm-') || normalized.startsWith('zhipu-')) return 'zhipu'
  if (normalized.startsWith('minimax-')) return 'minimax'
  if (normalized.startsWith('gemini-')) return 'gemini'
  if (normalized.startsWith('auto:') || normalized.startsWith('tokenrouter:') || normalized.startsWith('tokenrouter/')) {
    return 'tokenrouter'
  }
  if (normalized.includes('/')) return 'openrouter'
  if (normalized.startsWith('custom-')) return 'custom'
  return 'openai'
}

function normalizeProviderKey(provider) {
  return String(provider || '').trim().toLowerCase()
}

function routeModel(model, routeKeys, label = model) {
  return {
    id: `${model}:${routeKeys.join(',')}`,
    label,
    routeModels: Object.fromEntries(routeKeys.map((key) => [key, model])),
  }
}
