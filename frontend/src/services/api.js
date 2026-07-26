import * as DesktopApp from '../../wailsjs/go/main/DesktopApp.js'

const API_BASE = import.meta.env?.VITE_OMNIPROXY_API || 'http://127.0.0.1:3890/api'
const STATIC_CONTROL_TOKEN =
  import.meta.env?.VITE_OMNIPROXY_CONTROL_TOKEN || globalThis.__OMNIPROXY_CONTROL_TOKEN__ || ''
const CONTROL_TOKEN_HEADER = 'X-OmniProxy-Control-Token'
let controlTokenPromise = null

function useWailsBindings() {
  return (
    !import.meta.env?.VITE_OMNIPROXY_API &&
    typeof window !== 'undefined' &&
    Boolean(window.go?.main?.DesktopApp)
  )
}

function desktopMethod(name) {
  if (!useWailsBindings()) return null
  if (typeof DesktopApp[name] === 'function') return DesktopApp[name]
  const runtimeMethod = window.go?.main?.DesktopApp?.[name]
  return typeof runtimeMethod === 'function' ? runtimeMethod : null
}

function callDesktopOr(name, args, fallback) {
  const method = desktopMethod(name)
  return method ? method(...args) : fallback()
}

function desktopOnly(name, args, message) {
  return callDesktopOr(name, args, () => Promise.reject(new Error(message)))
}

async function request(path, options = {}) {
  const headers = {
    'Content-Type': 'application/json',
    ...(options.headers || {}),
  }
  const token = await getHTTPControlToken()
  if (token) {
    headers[CONTROL_TOKEN_HEADER] = token
  }

  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
  })

  if (response.status === 401 && token) {
    controlTokenPromise = null
    const retryToken = await getHTTPControlToken()
    if (retryToken && retryToken !== token) {
      return request(path, options)
    }
  }

  if (response.status === 204) {
    return null
  }

  const data = await response.json().catch(() => null)
  if (!response.ok) {
    throw new Error(data?.error || `请求失败：${response.status}`)
  }
  return data
}

async function getHTTPControlToken() {
  if (useWailsBindings()) {
    return ''
  }
  if (STATIC_CONTROL_TOKEN) {
    return STATIC_CONTROL_TOKEN
  }
  if (!controlTokenPromise) {
    controlTokenPromise = fetch(`${API_BASE}/control-token`, {
      headers: {
        Accept: 'application/json',
      },
    })
      .then(async (response) => {
        const data = await response.json().catch(() => null)
        if (!response.ok) {
          throw new Error(data?.error || `控制令牌获取失败：${response.status}`)
        }
        return data?.token || ''
      })
      .catch((error) => {
        controlTokenPromise = null
        throw error
      })
  }
  return controlTokenPromise
}

function historyFilter(filters = {}) {
  return {
    provider: filters.provider || 'all',
    client: filters.client || 'all',
    level: filters.level || 'all',
    status: filters.status || 'all',
    model: filters.model || '',
    tokenId: filters.tokenId || 'all',
    token: filters.token || '',
    search: filters.search || '',
    limit: Number(filters.limit || 1000),
  }
}

function historyQuery(filters = {}) {
  const params = new URLSearchParams()
  const normalized = historyFilter(filters)
  Object.entries(normalized).forEach(([key, value]) => {
    if (value === '' || value === undefined || value === null) return
    params.set(key, String(value))
  })
  return params.toString()
}

export function getTokens() {
  return callDesktopOr('Tokens', [], () => request('/tokens'))
}

export function createToken(payload) {
  return callDesktopOr('CreateToken', [payload], () => request('/tokens', {
    method: 'POST',
    body: JSON.stringify(payload),
  }))
}

export function importAPIKeys(payload) {
  return callDesktopOr('ImportAPIKeys', [payload], () => request('/tokens/import-api-keys', {
    method: 'POST',
    body: JSON.stringify(payload),
  }))
}

export function updateToken(id, payload) {
  return callDesktopOr('UpdateToken', [id, payload], () => request(`/tokens/${id}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
  }))
}

export function deleteToken(id) {
  return callDesktopOr('DeleteToken', [id], () => request(`/tokens/${id}`, {
    method: 'DELETE',
  }))
}

export function setTokenDisabled(id, disabled) {
  return callDesktopOr('SetTokenDisabled', [id, disabled], () => request(`/tokens/${id}/disabled`, {
    method: 'PUT',
    body: JSON.stringify({ disabled }),
  }))
}

export function useOnlyToken(id) {
  return callDesktopOr('UseOnlyToken', [id], () => request(`/tokens/${id}/exclusive`, {
    method: 'PUT',
  }))
}

export function cancelUseOnlyToken(id) {
  return callDesktopOr('CancelUseOnlyToken', [id], () => request(`/tokens/${id}/exclusive`, {
    method: 'DELETE',
  }))
}

export function setTokenSelected(id, selected) {
  return callDesktopOr('SetTokenSelected', [id, selected], () => request(`/tokens/${id}/selected`, {
    method: 'PUT',
    body: JSON.stringify({ selected }),
  }))
}

export function validateToken(id) {
  return callDesktopOr('ValidateToken', [id], () => request(`/tokens/${id}/validate`, {
    method: 'POST',
  }))
}

export function refreshTokenAuth(id) {
  return callDesktopOr('RefreshTokenAuth', [id], () => request(`/tokens/${id}/refresh`, {
    method: 'POST',
  }))
}

export function consumeCodexResetCredit(id) {
  return callDesktopOr('ConsumeCodexResetCredit', [id], () => request(`/tokens/${id}/reset-credit`, {
    method: 'POST',
  }))
}

export function startCodexOAuthLogin(refresh = false) {
  const forceRefresh = Boolean(refresh)
  return callDesktopOr('StartCodexOAuthLogin', [forceRefresh], () => request(`/codex/login/start?refresh=${forceRefresh}`, {
    method: 'POST',
  }))
}

export function getCodexOAuthLoginStatus(loginId) {
  const normalizedLoginId = String(loginId || '').trim()
  const params = new URLSearchParams({ loginId: normalizedLoginId })
  return callDesktopOr('CodexOAuthLoginStatus', [normalizedLoginId], () => request(`/codex/login/status?${params.toString()}`))
}

export function completeCodexOAuthLogin(loginId) {
  return callDesktopOr('CompleteCodexOAuthLogin', [loginId], () => request('/codex/login/complete', {
    method: 'POST',
    body: JSON.stringify({ loginId }),
  }))
}

export function startClaudeOAuthLogin(refresh = false) {
  const forceRefresh = Boolean(refresh)
  return callDesktopOr('StartClaudeOAuthLogin', [forceRefresh], () => request(`/claude/login/start?refresh=${forceRefresh}`, {
    method: 'POST',
  }))
}

export function getClaudeOAuthLoginStatus(loginId) {
  const normalizedLoginId = String(loginId || '').trim()
  const params = new URLSearchParams({ loginId: normalizedLoginId })
  return callDesktopOr('ClaudeOAuthLoginStatus', [normalizedLoginId], () => request(`/claude/login/status?${params.toString()}`))
}

export function completeClaudeOAuthLogin(loginId) {
  return callDesktopOr('CompleteClaudeOAuthLogin', [loginId], () => request('/claude/login/complete', {
    method: 'POST',
    body: JSON.stringify({ loginId }),
  }))
}

export function getOpenRouterModels(refresh = false) {
  const params = new URLSearchParams()
  if (refresh) params.set('refresh', 'true')
  return callDesktopOr('OpenRouterModels', [Boolean(refresh)], () => request(`/openrouter/models?${params.toString()}`))
}

export function sendOpenRouterChat(payload) {
  return callDesktopOr('OpenRouterChat', [payload], () => request('/openrouter/chat', {
    method: 'POST',
    body: JSON.stringify(payload),
  }))
}

export function getConfig() {
  return callDesktopOr('Config', [], () => request('/config'))
}

export function saveConfig(payload) {
  return callDesktopOr('SaveConfig', [payload], () => request('/config', {
    method: 'PUT',
    body: JSON.stringify(payload),
  }))
}

export function listConfigSnapshots() {
  return callDesktopOr('ConfigSnapshots', [], () => request('/config/snapshots'))
}

export function createConfigSnapshot(name = '') {
  return callDesktopOr('CreateConfigSnapshot', [String(name || '')], () => request('/config/snapshots', {
    method: 'POST',
    body: JSON.stringify({ name }),
  }))
}

export function restoreConfigSnapshot(id) {
  return callDesktopOr('RestoreConfigSnapshot', [id], () => request(`/config/snapshots/${encodeURIComponent(id)}`, {
    method: 'PUT',
  }))
}

export function deleteConfigSnapshot(id) {
  return callDesktopOr('DeleteConfigSnapshot', [id], () => request(`/config/snapshots/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  }))
}

export async function exportConfigBundle() {
  const exportBundle = desktopMethod('ExportConfigBundle')
  if (exportBundle) {
    return exportBundle()
  }
  const bundle = await request('/config/export')
  const data = JSON.stringify(bundle, null, 2)
  const fileName = `omniproxy-config-${new Date().toISOString().replace(/[:.]/g, '-')}.json`
  downloadJSON(fileName, data)
  return { fileName }
}

export async function importConfigBundle(file = null) {
  const importBundle = desktopMethod('ImportConfigBundle')
  if (importBundle) {
    return importBundle()
  }
  if (!file) {
    throw new Error('请选择要导入的配置文件')
  }
  return request('/config/import', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: await file.text(),
  })
}

export function getLogs() {
  return callDesktopOr('Logs', [], () => request('/logs'))
}

export function diagnoseGatewayRoute(payload) {
  return callDesktopOr('GatewayRouteDiagnostics', [payload], () => request('/gateway/diagnose', {
    method: 'POST',
    body: JSON.stringify(payload),
  }))
}

export function syncProviderModels(provider) {
  const payload = { provider: String(provider || '').trim() }
  return callDesktopOr('ProviderModels', [payload], () => request('/models/sync', {
    method: 'POST',
    body: JSON.stringify(payload),
  }))
}

export function getHistory(filters = {}) {
  return callDesktopOr('RequestHistory', [historyFilter(filters)], () => request(`/history?${historyQuery(filters)}`))
}

export function getHistorySummary(filters = {}, days = 14) {
  const normalizedDays = Number(days || 14)
  const params = new URLSearchParams(historyQuery(filters))
  params.set('days', String(normalizedDays))
  return callDesktopOr(
    'RequestHistorySummary',
    [historyFilter(filters), normalizedDays],
    () => request(`/history/summary?${params.toString()}`),
  )
}

export function getBillingUsage(date) {
  const value = String(date || '').trim()
  const params = new URLSearchParams()
  if (value) params.set('date', value)
  return callDesktopOr('BillingUsage', [value], () => request(`/billing/usage?${params.toString()}`))
}

export function getBillingDates(limit = 30) {
  const normalizedLimit = Number(limit || 30)
  const params = new URLSearchParams()
  params.set('limit', String(normalizedLimit))
  return callDesktopOr('BillingDates', [normalizedLimit], () => request(`/billing/dates?${params.toString()}`))
}

export function getBillingSummary(days = 30) {
  const normalizedDays = Number(days || 30)
  const params = new URLSearchParams()
  params.set('days', String(normalizedDays))
  return callDesktopOr('BillingSummary', [normalizedDays], () => request(`/billing/summary?${params.toString()}`))
}

export function clearBillingUsage() {
  return callDesktopOr('ClearBillingUsage', [], () => request('/billing/clear', {
    method: 'DELETE',
  }))
}

export function rebuildHistorySummaries() {
  return callDesktopOr('RebuildHistorySummaries', [], () => request('/history/rebuild-summaries', {
    method: 'POST',
  }))
}

export function clearRequestHistory() {
  return callDesktopOr('ClearRequestHistory', [], () => request('/history/clear', {
    method: 'DELETE',
  }))
}

export function getDataDirectory() {
  return callDesktopOr('DataDirectory', [], () => request('/data-directory'))
}

export function chooseDataDirectory(migrate = true) {
  return desktopOnly('ChooseDataDirectory', [Boolean(migrate)], '更改数据目录需要在桌面客户端中操作')
}

export function getTaskAutomationBrowserProfiles(browser = 'default') {
  return callDesktopOr('TaskAutomationBrowserProfiles', [String(browser || 'default')], () => Promise.resolve([]))
}

export function getClientConfigPreviews() {
  return callDesktopOr('ClientConfigPreviews', [], () => request('/clients/preview'))
}

export async function exportHistory(format, filters = {}, entries = []) {
  const exportRequestHistory = desktopMethod('ExportRequestHistory')
  if (exportRequestHistory) {
    return exportRequestHistory(format, historyFilter(filters))
  }
  const data = format === 'json' ? JSON.stringify(entries, null, 2) : historyCSV(entries)
  const type = format === 'json' ? 'application/json' : 'text/csv;charset=utf-8'
  const blob = new Blob([format === 'csv' ? '\uFEFF' : '', data], { type })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `omniproxy-request-history-${new Date().toISOString().replace(/[:.]/g, '-')}.${format}`
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
  return link.download
}

export function exportTokens() {
  return desktopOnly('ExportTokens', [], '导出账号池需要在桌面客户端中操作')
}

export function exportCodexAuthFiles() {
  return desktopOnly('ExportCodexAuthFiles', [], '导出 Codex auth 文件需要在桌面客户端中操作')
}

export async function exportDiagnosticsBundle() {
  const exportBundle = desktopMethod('ExportDiagnosticsBundle')
  if (exportBundle) {
    return exportBundle()
  }
  const bundle = await request('/diagnostics/bundle')
  const data = JSON.stringify(bundle, null, 2)
  const fileName = `omniproxy-diagnostics-${new Date().toISOString().replace(/[:.]/g, '-')}.json`
  const blob = new Blob([data], { type: 'application/json' })
  downloadBlob(fileName, blob)
  return { fileName }
}

function downloadJSON(fileName, data) {
  downloadBlob(fileName, new Blob([data], { type: 'application/json' }))
}

function downloadBlob(fileName, blob) {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = fileName
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}

function historyCSV(entries) {
  const header = [
    '时间',
    '级别',
    '方法',
    '路径',
    '路由厂商',
    '协议',
    '编程工具',
    '模型',
    '状态码',
    '耗时(ms)',
    '账号',
    '输入Token',
    '输出Token',
    '总Token',
    '触发冷却',
    '错误摘要',
    '重试链路',
  ]
  const rows = entries.map((entry) => [
    entry.time || '',
    entry.level || '',
    entry.method || '',
    entry.path || '',
    entry.provider || '',
    entry.protocol || '',
    entry.clientName || entry.clientKey || '',
    entry.model || '',
    entry.status || '',
    entry.durationMs || 0,
    entry.tokenName || '',
    entry.inputTokens || 0,
    entry.outputTokens || 0,
    entry.totalTokens || 0,
    entry.cooldownTriggered ? '是' : '否',
    entry.message || '',
    retryChainText(entry.retryChain || []),
  ])
  return [header, ...rows].map((row) => row.map(csvCell).join(',')).join('\n')
}

function retryChainText(chain) {
  return chain
    .map((attempt) => {
      const parts = [`#${attempt.attempt || '-'}`, attempt.provider || '-', attempt.tokenName || '-']
      if (attempt.status) parts.push(String(attempt.status))
      parts.push(`${attempt.durationMs || 0}ms`)
      if (attempt.cooldownTriggered) parts.push('冷却')
      if (attempt.message) parts.push(attempt.message)
      return parts.join(' ')
    })
    .join(' | ')
}

function csvCell(value) {
  const text = String(value ?? '')
  if (/[",\n\r]/.test(text)) {
    return `"${text.replace(/"/g, '""')}"`
  }
  return text
}

export function getProxyStatus() {
  return callDesktopOr('ProxyStatus', [], () => request('/proxy/status'))
}

export function getActiveRequests() {
  return callDesktopOr('ActiveProxyRequests', [], () => request('/proxy/active-requests'))
}

export function startProxy() {
  return callDesktopOr('StartProxy', [], () => request('/proxy/start', { method: 'POST' }))
}

export function stopProxy() {
  return callDesktopOr('StopProxy', [], () => request('/proxy/stop', { method: 'POST' }))
}

export function checkForUpdates() {
  return callDesktopOr('CheckForUpdates', [], () => request('/update/check'))
}

export function downloadUpdate(payload) {
  return callDesktopOr(
    'DownloadUpdate',
    [payload],
    () => request('/update/download', { method: 'POST', body: JSON.stringify(payload) }),
  )
}

export function getUpdateDownloadStatus() {
  return callDesktopOr('UpdateDownloadStatus', [], () => request('/update/download/status'))
}

export function getUpdateDiagnostics() {
  return callDesktopOr('UpdateDiagnostics', [], () => request('/update/diagnostics'))
}

export function installDownloadedUpdate() {
  return callDesktopOr('InstallDownloadedUpdate', [], () => request('/update/install', { method: 'POST' }))
}

export function getAppInfo() {
  return callDesktopOr('AppInfo', [], () => request('/app/info'))
}

export function openExternalURL(url) {
  if (!url) return
  if (useWailsBindings() && window.runtime?.BrowserOpenURL) {
    window.runtime.BrowserOpenURL(url)
    return
  }
  window.open(url, '_blank', 'noopener,noreferrer')
}

export function getAutoStartStatus() {
  return callDesktopOr('AutoStartStatus', [], () => Promise.resolve({ enabled: false }))
}

export function setAutoStart(enabled) {
  return callDesktopOr('SetAutoStart', [enabled], () => Promise.resolve({ enabled: false }))
}

export function configureCodex(models) {
  const payload = Array.isArray(models)
    ? { models: models.map((model) => String(model || '').trim()).filter(Boolean) }
    : { model: String(models || '').trim() }
  return callDesktopOr(
    'ConfigureCodex',
    [payload],
    () => request('/codex/configure', { method: 'POST', body: JSON.stringify(payload) }),
  )
}

export function restoreCodex() {
  return callDesktopOr('RestoreCodex', [], () => request('/codex/restore', { method: 'POST' }))
}

export function configureClaudeModels(models) {
  const payload = { models: Array.isArray(models) ? models : [] }
  return callDesktopOr(
    'ConfigureClaudeModels',
    [payload],
    () => request('/claude/models/configure', { method: 'POST', body: JSON.stringify(payload) }),
  )
}

export function configureClaudeDesktopModels(models) {
  const payload = { models: Array.isArray(models) ? models : [] }
  return callDesktopOr(
    'ConfigureClaudeDesktopModels',
    [payload],
    () => request('/claude/desktop/models/configure', { method: 'POST', body: JSON.stringify(payload) }),
  )
}

export function restoreClaudeDesktop() {
  return callDesktopOr('RestoreClaudeDesktop', [], () => request('/claude/desktop/restore', { method: 'POST' }))
}

export function restoreClaude() {
  return callDesktopOr('RestoreClaude', [], () => request('/claude/restore', { method: 'POST' }))
}

export function configureDeepSeekTUI() {
  return callDesktopOr('ConfigureDeepSeekTUI', [], () => request('/deepseek-tui/configure', { method: 'POST' }))
}

export function restoreDeepSeekTUI() {
  return callDesktopOr('RestoreDeepSeekTUI', [], () => request('/deepseek-tui/restore', { method: 'POST' }))
}

export function configureGemini() {
  return callDesktopOr('ConfigureGemini', [], () => request('/gemini/configure', { method: 'POST' }))
}

export function restoreGemini() {
  return callDesktopOr('RestoreGemini', [], () => request('/gemini/restore', { method: 'POST' }))
}

export function configureOpenCode() {
  return callDesktopOr('ConfigureOpenCode', [], () => request('/opencode/configure', { method: 'POST' }))
}

export function restoreOpenCode() {
  return callDesktopOr('RestoreOpenCode', [], () => request('/opencode/restore', { method: 'POST' }))
}

export function configurePi() {
  return callDesktopOr('ConfigurePi', [], () => request('/pi/configure', { method: 'POST' }))
}

export function restorePi() {
  return callDesktopOr('RestorePi', [], () => request('/pi/restore', { method: 'POST' }))
}
