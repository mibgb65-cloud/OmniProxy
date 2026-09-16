import { credentialTypes, providers } from '../constants/app'
import { knownClientTools } from '../constants/clientTools'
import { formatDuration, formatTime } from '../utils/format'

const providerBaseUrlKeys = new Set(['sub2api', 'newapi', 'anyrouter'])

export function createTokenHelpers(state) {
  function providerRequiresBaseUrl(provider) {
    return providerBaseUrlKeys.has(String(provider || '').trim())
  }

  function providerDefaultBaseUrl(provider) {
    if (provider === 'sub2api') return state.config.sub2apiBaseUrl
    if (provider === 'newapi') return state.config.newapiBaseUrl
    if (provider === 'anyrouter') return state.config.anyrouterBaseUrl
    return ''
  }

  function validateProviderBaseUrl(provider, baseUrl) {
    if (!providerRequiresBaseUrl(provider)) return true
    const label = providerLabel(provider)
    if (!baseUrl) {
      state.errorMessage.value = `${label} Base URL 不能为空`
      return false
    }
    try {
      const parsed = new URL(baseUrl)
      if (!['http:', 'https:'].includes(parsed.protocol) || !parsed.host) {
        state.errorMessage.value = `${label} Base URL 格式不正确`
        return false
      }
    } catch {
      state.errorMessage.value = `${label} Base URL 格式不正确`
      return false
    }
    return true
  }

  function credentialDisplay(item) {
    if (item.credentialType === 'codex_auth_json') {
      return item.accountId ? `account_id: ${item.accountId}` : item.maskedTokenValue || 'auth.json'
    }
    if (item.maskedTokenValue) return item.maskedTokenValue
    if (item.credentialType === 'claude_oauth_json') return 'OAuth JSON'
    return item.hasTokenValue ? '已保存' : '-'
  }

  function credentialPlaceholder() {
    if (state.form.editingId) {
      return '留空表示保留当前凭据'
    }
    if (state.form.credentialType === 'codex_auth_json') {
      return '粘贴 ~/.codex/auth.json 或 CLIProxyAPI Codex JSON 的完整内容'
    }
    if (state.form.credentialType === 'claude_oauth_json') {
      return '粘贴 Claude OAuth JSON，需包含 access_token / refresh_token / expired'
    }
    if (state.form.credentialType === 'mimo_token_plan') {
      return '粘贴 tp- 开头的 MiMo Token Plan Key'
    }
    if (state.form.credentialType === 'coding_plan') {
      return '粘贴 GLM Coding Plan Key'
    }
    if (state.form.provider === 'xiaomi') {
      return '粘贴 sk- 开头的 MiMo 按量 API Key'
    }
    if (state.form.provider === 'kimi') {
      return '粘贴 Kimi Code API Key'
    }
    if (state.form.provider === 'zhipu') {
      return '粘贴 Zhipu GLM API Key，格式通常为 id.secret'
    }
    if (state.form.provider === 'minimax') {
      return '粘贴 MiniMax API Key'
    }
    if (state.form.provider === 'gemini') {
      return '粘贴 Google Gemini API Key'
    }
    if (state.form.provider === 'openrouter') {
      return '粘贴 OpenRouter API Key'
    }
    if (state.form.provider === 'tokenrouter') {
      return '粘贴 tr_ 开头的 TokenRouter API Key'
    }
    if (state.form.provider === 'sub2api') {
      return '粘贴 sub2api API Key'
    }
    if (state.form.provider === 'newapi') {
      return '粘贴 new-api API Key'
    }
    if (state.form.provider === 'anyrouter') {
      return '粘贴 sk- 开头的 AnyRouter API Key'
    }
    if (state.form.provider === 'zo') {
      return '粘贴 zo_sk_ 开头的 Zo Access Token'
    }
    if (state.form.provider === 'prem') {
      return '粘贴 Prem API Key'
    }
    if (state.form.provider === 'atria') {
      return '粘贴 atr_ 开头的 Atria API Key'
    }
    if (state.form.provider === 'custom') {
      return '粘贴自定义网关 API Key'
    }
    return '粘贴 API Key'
  }

  function providerTokens(provider) {
    return state.tokens.value.filter((item) => item.provider === provider)
  }

  function selectProvider(provider) {
    state.activeProvider.value = provider
  }

  function credentialLabel(item) {
    const label = credentialTypes[item.credentialType || 'api_key'] || item.credentialType || 'API Key'
    if (item.provider === 'xiaomi' && item.credentialType === 'mimo_token_plan') {
      const regionLabels = {
        cn: '中国区',
        sgp: '新加坡 SGP',
        ams: '欧洲 AMS',
      }
      return `${label} · ${regionLabels[item.region] || '中国区'}`
    }
    return label
  }

  function normalizedCredentialType(provider, credentialType) {
    if (provider === 'openai') {
      return credentialType === 'codex_auth_json' ? 'codex_auth_json' : 'api_key'
    }
    if (provider === 'anthropic') {
      return credentialType === 'claude_oauth_json' ? 'claude_oauth_json' : 'api_key'
    }
    if (provider === 'xiaomi') {
      return credentialType === 'mimo_token_plan' ? 'mimo_token_plan' : 'api_key'
    }
    if (provider === 'zhipu') {
      return credentialType === 'coding_plan' ? 'coding_plan' : 'api_key'
    }
    return 'api_key'
  }

  function providerLabel(providerKey) {
    return providers.find((item) => item.key === providerKey)?.label || providerKey
  }

  function clientToolLabel(key, fallback = '') {
    return knownClientTools.find((item) => item.key === key)?.label || fallback || key || '未知工具'
  }

  function buildToolUsageRows(activeItems, historyItems) {
    const byClient = new Map()
    for (const item of activeItems || []) {
      const key = item.clientKey || 'api'
      const current = byClient.get(key) || {
        clientKey: key,
        clientName: item.clientName || clientToolLabel(key),
        active: true,
        activeCount: 0,
        tokenNames: new Set(),
        providerNames: new Set(),
        models: new Set(),
        startedAt: item.startedAt,
        lastSeenAt: item.startedAt,
      }
      current.active = true
      current.activeCount += 1
      if (item.tokenName) current.tokenNames.add(item.tokenName)
      if (item.provider) current.providerNames.add(providerLabel(item.provider))
      if (item.model) current.models.add(item.model)
      if (!current.startedAt || new Date(item.startedAt).getTime() < new Date(current.startedAt).getTime()) {
        current.startedAt = item.startedAt
      }
      byClient.set(key, current)
    }

    const sortedHistory = [...(historyItems || [])].sort((a, b) => {
      return new Date(b.time || 0).getTime() - new Date(a.time || 0).getTime()
    })
    for (const entry of sortedHistory) {
      if (entry.method === 'CHECK') continue
      const key = entry.clientKey || ''
      if (!key) continue
      if (byClient.has(key)) continue
      byClient.set(key, {
        clientKey: key,
        clientName: entry.clientName || clientToolLabel(key),
        active: false,
        activeCount: 0,
        tokenNames: new Set(entry.tokenName ? [entry.tokenName] : []),
        providerNames: new Set(entry.provider ? [providerLabel(entry.provider)] : []),
        models: new Set(entry.model ? [entry.model] : []),
        startedAt: '',
        lastSeenAt: entry.time,
        status: entry.status,
      })
    }

    const order = new Map(knownClientTools.map((item, index) => [item.key, index]))
    return [...byClient.values()]
      .map((item) => ({
        ...item,
        tokenText: [...item.tokenNames].join('、') || '-',
        providerText: [...item.providerNames].join('、') || '-',
        modelText: [...item.models].join('、') || '-',
      }))
      .sort((a, b) => {
        if (a.active !== b.active) return a.active ? -1 : 1
        const rankA = order.has(a.clientKey) ? order.get(a.clientKey) : 999
        const rankB = order.has(b.clientKey) ? order.get(b.clientKey) : 999
        if (rankA !== rankB) return rankA - rankB
        return new Date(b.lastSeenAt || 0).getTime() - new Date(a.lastSeenAt || 0).getTime()
      })
      .slice(0, 8)
  }

  function toolUsageMeta(row) {
    const parts = []
    if (row.providerText && row.providerText !== '-') parts.push(row.providerText)
    if (row.modelText && row.modelText !== '-') parts.push(row.modelText)
    if (!row.active && row.lastSeenAt) parts.push(`最近 ${formatTime(row.lastSeenAt)}`)
    return parts.join(' · ') || '-'
  }

  function toolUsageDuration(row) {
    if (!row.active || !row.startedAt) return ''
    return `已运行 ${formatDuration(Math.max(0, Date.now() - new Date(row.startedAt).getTime()))}`
  }

  return {
    providerRequiresBaseUrl,
    providerDefaultBaseUrl,
    validateProviderBaseUrl,
    credentialDisplay,
    credentialPlaceholder,
    providerTokens,
    selectProvider,
    credentialLabel,
    normalizedCredentialType,
    providerLabel,
    clientToolLabel,
    buildToolUsageRows,
    toolUsageMeta,
    toolUsageDuration,
  }
}
