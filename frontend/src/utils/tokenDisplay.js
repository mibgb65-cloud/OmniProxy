import { statusMeta } from '../constants/app.js'
import { resolvePrice } from '../domain/billing/pricing.js'
import { formatNumber, formatTime } from './format.js'

const DAY_MS = 24 * 60 * 60 * 1000
const WEEK_MS = 7 * DAY_MS
const CODEX_WEEKLY_ESTIMATE_MODEL = 'gpt-5.6-sol'
const MIN_CODEX_WEEKLY_ESTIMATE_USED_PERCENT = 5

export function statusLabel(status) {
  return statusMeta[status]?.label || status
}

export function statusClass(status) {
  return statusMeta[status]?.className || 'muted'
}

export function statusType(status) {
  const className = statusClass(status)
  if (className === 'success') return 'success'
  if (className === 'warning') return 'warning'
  if (className === 'danger') return 'danger'
  return 'info'
}

export function planLabel(plan) {
  const normalized = String(plan || '').toLowerCase()
  const labels = {
    team: 'Team',
    pro: 'Pro',
    plus: 'Plus',
    free: 'Free',
    enterprise: 'Enterprise',
  }
  return labels[normalized] || plan || '未知'
}

export function usageUpdatedAt(item) {
  return item.usage?.updatedAt ? formatTime(item.usage.updatedAt) : '-'
}

export function quotaPercentValue(item, field) {
  if (!item?.usage?.subscriptionQuotaAvailable) return 0
  const value = Number(item.usage?.[field])
  if (!Number.isFinite(value)) return 0
  return Math.max(0, Math.min(100, Math.round(value)))
}

export function quotaPercentText(item, field) {
  if (!field || !item?.usage?.subscriptionQuotaAvailable) return '-'
  if (field.endsWith('UsedPercent')) {
    const exact = Number(item.usage?.[`${field}Exact`])
    if (Number.isFinite(exact) && exact > 0) return `${formatPercent(exact)}%`
  }
  return `${quotaPercentValue(item, field)}%`
}

export function formatBalance(value) {
  const number = Number(value || 0)
  const fractionDigits = Math.abs(number) > 0 && Math.abs(number) < 1 ? 4 : 2
  return new Intl.NumberFormat('zh-CN', {
    minimumFractionDigits: fractionDigits,
    maximumFractionDigits: fractionDigits,
  }).format(number)
}

export function apiBalanceSummaryMeta(summary) {
  const parts = [`${formatNumber(summary.count)} 个 API Key`]
  if (Number(summary.total || 0) > 0) {
    parts.push(`总额 ${formatBalance(summary.total)} ${summary.unit}`)
  }
  if (Number(summary.used || 0) > 0) {
    parts.push(`已用 ${formatBalance(summary.used)} ${summary.unit}`)
  }
  return parts.join(' · ')
}

export function hasBalanceUsage(item) {
  return Boolean(item.usage?.balanceUnit)
}

export function hasOpenRouterQuotaUsage(item) {
  if (item?.provider !== 'openrouter') return false
  return hasBalanceUsage(item) || Boolean(item.usage?.balanceUnlimited)
}

export function quotaDisplay(item) {
  if (item.usage?.balanceUnlimited) {
    return '不限制'
  }
  if (hasBalanceUsage(item)) {
    return `${formatBalance(item.usage.balanceRemaining)} ${item.usage.balanceUnit}`
  }
  return `${item.remaining}%`
}

export function quotaStatLabel(item) {
  return hasBalanceUsage(item) ? '账户余额' : 'API 剩余额度'
}

export function quotaStatMeta(item) {
  if (hasBalanceUsage(item)) {
    const parts = []
    if (item.usage?.balanceUnlimited) {
      parts.push('未设置消费上限')
    }
    if (Number(item.usage?.balanceTotal || 0) > 0) {
      parts.push(`总额 ${formatBalance(item.usage.balanceTotal)} ${item.usage.balanceUnit}`)
    }
    if (Number(item.usage?.balanceUsed || 0) > 0) {
      parts.push(`已用 ${formatBalance(item.usage.balanceUsed)} ${item.usage.balanceUnit}`)
    }
    parts.push(`最后更新 ${usageUpdatedAt(item)}`)
    return parts.join(' · ')
  }
  return `最后更新 ${usageUpdatedAt(item)}`
}

export function openRouterQuotaValue(item, field) {
  if (!hasOpenRouterQuotaUsage(item)) {
    return '-'
  }
  return `${formatBalance(item.usage?.[field])} ${item.usage.balanceUnit}`
}

export function openRouterQuotaRemaining(item) {
  if (!hasOpenRouterQuotaUsage(item)) {
    return '待刷新'
  }
  if (item.usage?.balanceUnlimited) {
    return '不限制'
  }
  if (Number(item.usage?.balanceTotal || 0) <= 0 && Number(item.usage?.balanceRemaining || 0) <= 0) {
    return '未返回'
  }
  return openRouterQuotaValue(item, 'balanceRemaining')
}

export function openRouterQuotaLimit(item) {
  if (!hasOpenRouterQuotaUsage(item)) {
    return '-'
  }
  if (item.usage?.balanceUnlimited) {
    return '不限制'
  }
  if (Number(item.usage?.balanceTotal || 0) <= 0) {
    return '未设置'
  }
  return openRouterQuotaValue(item, 'balanceTotal')
}

export function openRouterQuotaMeta(item) {
  if (!hasOpenRouterQuotaUsage(item)) {
    return item?.disabled ? '已停用，启用后可刷新额度' : '点击刷新额度获取 OpenRouter /key 余额'
  }
  const parts = []
  if (item?.usage?.planType) {
    parts.push(item.usage.planType)
  }
  if (item?.usage?.message) {
    parts.push(item.usage.message)
  }
  parts.push(`最后更新 ${usageUpdatedAt(item)}`)
  return parts.join(' · ')
}

export function validationSuccessMessage(token, result) {
  if (token?.provider === 'openrouter' && result?.usage) {
    const usage = result.usage
    if (usage.balanceUnlimited) {
      const used = usage.balanceUnit ? `${formatBalance(usage.balanceUsed)} ${usage.balanceUnit}` : '-'
      return `OpenRouter 额度已刷新：消费上限不限制，已用 ${used}`
    }
    if (usage.balanceUnit) {
      const remaining = `${formatBalance(usage.balanceRemaining)} ${usage.balanceUnit}`
      const used = `${formatBalance(usage.balanceUsed)} ${usage.balanceUnit}`
      return `OpenRouter 额度已刷新：剩余 ${remaining}，已用 ${used}`
    }
  }
  return `验证通过：${result.status}，耗时 ${result.durationMs}ms`
}

export function balancePackages(item) {
  return Array.isArray(item?.usage?.balancePackages) ? item.usage.balancePackages : []
}

export function balancePackageCounts(pkg) {
  const status = String(pkg?.status || '').toUpperCase()
  const type = String(pkg?.consumeType || '').toUpperCase()
  return (!status || status === 'EFFECTIVE') && (!type || type === 'TOKENS')
}

export function balancePackageTypeLabel(pkg) {
  const type = String(pkg?.consumeType || '').toUpperCase()
  if (type === 'TIMES') return '次数包'
  if (type === 'TOKENS' || !type) return 'Token 包'
  return type
}

export function balancePackageAmount(pkg) {
  const unit = pkg?.unit || (String(pkg?.consumeType || '').toUpperCase() === 'TIMES' ? '次' : 'Token')
  return `${formatNumber(pkg?.balanceRemaining)} ${unit}`
}

export function balancePackageMeta(pkg) {
  const parts = []
  if (pkg?.balanceTotal && Number(pkg.balanceTotal) !== Number(pkg.balanceRemaining || 0)) {
    parts.push(`总量 ${formatNumber(pkg.balanceTotal)}`)
  }
  if (pkg?.status && pkg.status !== 'EFFECTIVE') {
    parts.push(pkg.status)
  }
  if (pkg?.expirationTime) {
    parts.push(`到期 ${formatPackageExpiration(pkg.expirationTime)}`)
  }
  if (pkg?.suitableModel) {
    parts.push(pkg.suitableModel)
  }
  return parts.join(' · ') || (balancePackageCounts(pkg) ? '计入 Token 余额' : '仅展示，不计入 Token 余额')
}

function formatPackageExpiration(value) {
  if (!value) return '-'
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}

export function apiQuotaDisplay(item) {
  if (hasBalanceUsage(item)) {
    return quotaDisplay(item)
  }
  const remaining = Number(item.usage?.apiRemaining || 0)
  if (remaining > 0) {
    return `余量 ${formatNumber(remaining)}`
  }
  return displayStatusLabel(item)
}

export function apiQuotaMeta(item) {
  if (hasBalanceUsage(item)) {
    return quotaStatMeta(item)
  }
  if (Number(item.usage?.apiRemaining || 0) > 0) {
    return `来自上游 rate-limit header · 最后更新 ${usageUpdatedAt(item)}`
  }
  return healthSummary(item)
}

export function isCodexToken(item) {
  return item?.provider === 'openai' && item?.credentialType === 'codex_auth_json'
}

export function isClaudeOAuthToken(item) {
  return item?.provider === 'anthropic' && item?.credentialType === 'claude_oauth_json'
}

export function codexResetCreditsAvailable(item) {
  const value = Number(item?.usage?.codexResetCreditsAvailable)
  if (!Number.isFinite(value)) return null
  return Math.max(0, Math.trunc(value))
}

export function codexResetCredits(item) {
  return Array.isArray(item?.usage?.codexResetCredits) ? item.usage.codexResetCredits : []
}

export function codexResetCreditStatusMeta(credit) {
  const status = String(credit?.status || credit?.rawStatus || '').trim().toLowerCase()
  if (['redeemed', 'used', 'consumed'].includes(status)) return { label: '已使用', type: 'info' }
  const expiresAt = Number(credit?.expiresAt || 0)
  if (status === 'expired' || (expiresAt > 0 && expiresAt <= Date.now() / 1000)) return { label: '已过期', type: 'warning' }
  if (status === 'available' || !status) return { label: '可使用', type: 'success' }
  return { label: '未知', type: 'info' }
}

export function codexResetCreditTypeLabel(credit) {
  const resetType = String(credit?.resetType || '').trim()
  if (!resetType || resetType === 'codex_rate_limits') return 'Codex 限额刷新'
  return resetType.replaceAll('_', ' ')
}

export function isMimoTokenPlan(item) {
  return item?.provider === 'xiaomi' && item?.credentialType === 'mimo_token_plan'
}

export function isZhipuCodingPlan(item) {
  return item?.provider === 'zhipu' && item?.credentialType === 'coding_plan'
}

export function showQuotaWindows(item) {
  return isCodexToken(item) || isMimoTokenPlan(item) || Boolean(item?.usage?.subscriptionQuotaAvailable)
}

export function quotaWindowAvailable(item, windowName) {
  if (!item?.usage?.subscriptionQuotaAvailable) return false
  const prefix = windowName === 'secondary' ? 'secondary' : 'primary'
  return ['UsedPercent', 'RemainingPercent', 'ResetAt'].some((suffix) => {
    const value = Number(item.usage?.[`${prefix}${suffix}`])
    return Number.isFinite(value) && value > 0
  })
}

export function isCodexFreePlan(item) {
  return isCodexToken(item) && String(item?.usage?.planType || '').trim().toLowerCase() === 'free'
}

export function showPrimaryQuotaWindow(item) {
  if (!showQuotaWindows(item)) return false
  if (!item?.usage?.subscriptionQuotaAvailable) return !isCodexToken(item)
  if (isCodexFreePlan(item) && quotaWindowAvailable(item, 'secondary')) return false
  return quotaWindowAvailable(item, 'primary')
}

export function showSecondaryQuotaWindow(item) {
  if (!showQuotaWindows(item)) return false
  if (!item?.usage?.subscriptionQuotaAvailable) return !isCodexToken(item)
  if (isCodexFreePlan(item) && quotaWindowAvailable(item, 'primary')) return false
  return quotaWindowAvailable(item, 'secondary')
}

export function quotaWindowCount(item) {
  return Number(showPrimaryQuotaWindow(item)) + Number(showSecondaryQuotaWindow(item))
}

export function quotaPrimaryLabel(item) {
  if (isZhipuCodingPlan(item)) return '窗口额度'
  if (isCodexFreePlan(item)) return '1 周额度'
  return isMimoTokenPlan(item) ? '本月额度' : '5h额度'
}

export function quotaSecondaryLabel(item) {
  if (isZhipuCodingPlan(item)) return '周额度'
  return isMimoTokenPlan(item) ? '套餐额度' : '1 周额度'
}

export function quotaOverviewWindow(item) {
  if (showPrimaryQuotaWindow(item)) return 'primary'
  if (showSecondaryQuotaWindow(item)) return 'secondary'
  return ''
}

export function quotaOverviewLabel(item) {
  const windowName = quotaOverviewWindow(item)
  if (!windowName) return '额度待刷新'
  return windowName === 'secondary' ? quotaSecondaryLabel(item) : quotaPrimaryLabel(item)
}

export function quotaOverviewRemainingField(item) {
  const windowName = quotaOverviewWindow(item)
  return windowName ? `${windowName}RemainingPercent` : ''
}

export function codexWeeklyQuotaEstimate(item) {
  if (!isCodexToken(item) || !quotaWindowAvailable(item, 'secondary')) return null
  const usedPercent = quotaUsedPercent(item, 'secondary')
  if (usedPercent < MIN_CODEX_WEEKLY_ESTIMATE_USED_PERCENT) return null

  const usage = weeklyEstimateUsageStats(item)
  if (usage.totalTokens <= 0) return null

  const price = resolvePrice(CODEX_WEEKLY_ESTIMATE_MODEL)
  if (!price || price.currency !== 'USD') return null

  const scale = 100 / usedPercent
  const usedCost = usageCostUSD(usage, price)
  const amount = usedCost * scale
  if (!Number.isFinite(amount) || amount <= 0) return null

  return {
    amount,
    amountText: formatUSD(amount),
    priceLabel: price.label,
    totalTokens: Math.round(usage.totalTokens * scale),
    usedPercent,
    usedCost,
    usedCostText: formatUSD(usedCost),
    usedTokens: usage.totalTokens,
  }
}

export function codexWeeklyQuotaEstimateText(item) {
  const estimate = codexWeeklyQuotaEstimate(item)
  return estimate ? `${estimate.amountText} / 周` : ''
}

export function codexWeeklyQuotaEstimateMeta(item) {
  const estimate = codexWeeklyQuotaEstimate(item)
  if (!estimate) return ''
  return `按本地代理当前周窗口 ${formatNumber(estimate.usedTokens)} Token、基础计价成本 ${estimate.usedCostText} 和上游已用 ${formatPercent(estimate.usedPercent)}% 粗估 · ${estimate.priceLabel}`
}

export function quotaResetLabel(item) {
  return isMimoTokenPlan(item) ? '到期' : '重置'
}

export function quotaUnavailableText(item) {
  if (isCodexToken(item)) return '点击刷新额度获取'
  if (isMimoTokenPlan(item)) return 'Token Plan 暂无订阅额度'
  return '暂无订阅额度'
}

export function weeklyLimitReached(item) {
  if (!item?.usage?.subscriptionQuotaAvailable) return false
  if (!isZhipuCodingPlan(item) && !isCodexToken(item) && !isMimoTokenPlan(item)) return false
  const remaining = Number(item.usage?.secondaryRemainingPercent)
  const used = Number(item.usage?.secondaryUsedPercent)
  return Number.isFinite(remaining) && remaining <= 0 && Number.isFinite(used) && used > 0
}

export function tokenUsageMetrics(item) {
  const total = Number(item.stats?.totalTokens || 0)
  const input = Number(item.stats?.inputTokens || 0)
  const output = Number(item.stats?.outputTokens || 0)
  const cacheRead = Number(item.stats?.cacheReadTokens || 0)
  const requests = Number(item.stats?.requestCount || 0)
  if (total > 0) {
    const metrics = [
      { label: '累计 Token', value: formatNumber(total) },
      { label: '入', value: formatNumber(input) },
      { label: '出', value: formatNumber(output) },
    ]
    if (cacheRead > 0) metrics.splice(1, 0, { label: '缓存读取', value: formatNumber(cacheRead) })
    return metrics
  }
  return [{ label: '累计 Token', value: requests > 0 ? '未上报' : '0' }]
}

function quotaUsedPercent(item, windowName) {
  const exact = Number(item?.usage?.[`${windowName}UsedPercentExact`])
  if (Number.isFinite(exact) && exact > 0) return Math.max(0, Math.min(100, exact))
  const rawUsed = item?.usage?.[`${windowName}UsedPercent`]
  if (rawUsed !== undefined && rawUsed !== null && Number.isFinite(Number(rawUsed))) {
    return Math.max(0, Math.min(100, Number(rawUsed)))
  }
  return 100 - quotaPercentValue(item, `${windowName}RemainingPercent`)
}

function weeklyEstimateUsageStats(item) {
  const resetAt = numberValue(item?.usage?.secondaryResetAt)
  const daily = Array.isArray(item?.stats?.daily) ? item.stats.daily : []
  if (resetAt <= 0 || !daily.length) return emptyTokenStats()

  const resetMs = resetAt * 1000
  const startMs = resetMs - WEEK_MS
  const windowStats = daily.reduce((sum, row) => {
    if (!dailyRowOverlapsWindow(row, startMs, resetMs)) return sum
    return addTokenStats(sum, normalizeTokenStats(row))
  }, emptyTokenStats())
  return windowStats.totalTokens > 0 ? windowStats : emptyTokenStats()
}

function dailyRowOverlapsWindow(row, startMs, resetMs) {
  const dayStart = localDateStartMs(row?.date)
  if (!Number.isFinite(dayStart)) return false
  const dayEnd = dayStart + DAY_MS
  return dayEnd > startMs && dayStart <= resetMs
}

function localDateStartMs(value) {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(String(value || ''))
  if (!match) return NaN
  return new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3])).getTime()
}

function normalizeTokenStats(value) {
  const inputTokens = numberValue(value?.inputTokens)
  const outputTokens = numberValue(value?.outputTokens)
  const totalTokens = numberValue(value?.totalTokens) || inputTokens + outputTokens
  const cacheCreationTokens = numberValue(value?.cacheCreationTokens)
  const cacheReadTokens = numberValue(value?.cacheReadTokens)
  return {
    inputTokens,
    outputTokens,
    totalTokens,
    cacheCreationTokens,
    cacheReadTokens,
  }
}

function addTokenStats(left, right) {
  return {
    inputTokens: left.inputTokens + right.inputTokens,
    outputTokens: left.outputTokens + right.outputTokens,
    totalTokens: left.totalTokens + right.totalTokens,
    cacheCreationTokens: left.cacheCreationTokens + right.cacheCreationTokens,
    cacheReadTokens: left.cacheReadTokens + right.cacheReadTokens,
  }
}

function emptyTokenStats() {
  return { inputTokens: 0, outputTokens: 0, totalTokens: 0, cacheCreationTokens: 0, cacheReadTokens: 0 }
}

function usageCostUSD(usage, price) {
  let inputTokens = usage.inputTokens
  const outputTokens = usage.outputTokens
  const cacheCreationTokens = usage.cacheCreationTokens
  const cacheReadTokens = usage.cacheReadTokens

  if (inputTokens <= 0 && outputTokens <= 0) {
    inputTokens = usage.totalTokens
  } else if (inputTokens <= 0 && usage.totalTokens > outputTokens) {
    inputTokens = usage.totalTokens - outputTokens
  }

  const billableInputTokens = Math.max(0, inputTokens - cacheReadTokens)
  const cacheCreationPrice = priceValue(price.cacheCreation, price.input)
  const cacheReadPrice = priceValue(price.cacheRead, price.input)
  return (
    (billableInputTokens / 1_000_000) * price.input +
    (outputTokens / 1_000_000) * price.output +
    (cacheCreationTokens / 1_000_000) * cacheCreationPrice +
    (cacheReadTokens / 1_000_000) * cacheReadPrice
  )
}

function priceValue(value, fallback) {
  const number = Number(value)
  return Number.isFinite(number) && number >= 0 ? number : fallback
}

function numberValue(value) {
  const number = Number(value || 0)
  return Number.isFinite(number) && number > 0 ? number : 0
}

function formatUSD(value) {
  const number = Number(value || 0)
  const fractionDigits = Math.abs(number) > 0 && Math.abs(number) < 1 ? 4 : 2
  return `$${new Intl.NumberFormat('en-US', {
    minimumFractionDigits: fractionDigits,
    maximumFractionDigits: fractionDigits,
  }).format(number)}`
}

function formatPercent(value) {
  return new Intl.NumberFormat('zh-CN', { maximumFractionDigits: 2 }).format(Number(value || 0))
}

export function normalizeBillingDailyRows(rows) {
  return [...(rows || [])]
    .map((row) => ({
      date: String(row.date || ''),
      requestCount: Number(row.requestCount || 0),
      inputTokens: Number(row.inputTokens || 0),
      outputTokens: Number(row.outputTokens || 0),
      totalTokens: Number(row.totalTokens || 0),
      cacheCreationTokens: Number(row.cacheCreationTokens || 0),
      cacheReadTokens: Number(row.cacheReadTokens || 0),
    }))
    .filter((row) => row.date)
    .sort((a, b) => b.date.localeCompare(a.date))
}

export function isCooling(item) {
  return item?.cooldownUntil && new Date(item.cooldownUntil).getTime() > Date.now()
}

export function displayStatusLabel(item) {
  if (item?.disabled) return '已停用'
  if (isCooling(item)) return '冷却中'
  return statusLabel(item.status)
}

export function displayStatusClass(item) {
  if (item?.disabled) return 'muted'
  if (isCooling(item)) return 'warning'
  return statusClass(item.status)
}

export function displayStatusType(item) {
  if (item?.disabled) return 'info'
  if (isCooling(item)) return 'warning'
  return statusType(item.status)
}

export function healthSummary(item) {
  if (item?.healthMessage) {
    const label = {
      healthy: '健康',
      watch: '关注',
      risk: '高风险',
      disabled: '已停用',
    }[item.healthLevel] || '健康'
    const score = Number.isFinite(Number(item.healthScore)) ? ` · ${Math.round(Number(item.healthScore))} 分` : ''
    return `${label}${score} · ${item.healthMessage}`
  }
  if (item?.disabled) {
    return '已停用，不参与调度和自动检查'
  }
  if (isCooling(item)) {
    return `冷却到 ${formatTime(item.cooldownUntil)} 后自动复检`
  }
  if (item.health?.lastCheckedAt) {
    const status = item.health.lastStatus ? ` · ${item.health.lastStatus}` : ''
    return `健康检查 ${formatTime(item.health.lastCheckedAt)}${status}`
  }
  return '等待健康检查'
}
