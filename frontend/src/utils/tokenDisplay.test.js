import test from 'node:test'
import assert from 'node:assert/strict'

import {
  codexResetCreditStatusMeta,
  codexResetCreditTypeLabel,
  codexResetCredits,
  codexResetCreditsAvailable,
  codexWeeklyQuotaEstimateMeta,
  codexWeeklyQuotaEstimateText,
  displayStatusClass,
  normalizeBillingDailyRows,
  openRouterQuotaRemaining,
  quotaOverviewLabel,
  quotaOverviewRemainingField,
  quotaPrimaryLabel,
  quotaWindowCount,
  showPrimaryQuotaWindow,
  showSecondaryQuotaWindow,
  validationSuccessMessage,
  weeklyLimitReached,
} from './tokenDisplay.js'

test('normalizeBillingDailyRows sorts and normalizes usage rows', () => {
  assert.deepEqual(
    normalizeBillingDailyRows([
      { date: '2026-05-26', requestCount: '2', totalTokens: '30' },
      { date: '', requestCount: 10 },
      { date: '2026-05-27', inputTokens: '4', outputTokens: '6' },
    ]),
    [
      { date: '2026-05-27', requestCount: 0, inputTokens: 4, outputTokens: 6, totalTokens: 0, cacheCreationTokens: 0, cacheReadTokens: 0 },
      { date: '2026-05-26', requestCount: 2, inputTokens: 0, outputTokens: 0, totalTokens: 30, cacheCreationTokens: 0, cacheReadTokens: 0 },
    ],
  )
})

test('OpenRouter quota display uses refreshed balance data', () => {
  const token = {
    provider: 'openrouter',
    usage: {
      balanceRemaining: 12.5,
      balanceTotal: 20,
      balanceUnit: 'USD',
    },
  }

  assert.equal(openRouterQuotaRemaining(token), '12.50 USD')
  assert.equal(
    validationSuccessMessage({ provider: 'openrouter' }, { usage: { balanceRemaining: 1, balanceUsed: 2, balanceUnit: 'USD' } }),
    'OpenRouter 额度已刷新：剩余 1.00 USD，已用 2.00 USD',
  )
})

test('subscription quota helpers keep Codex free plan window rules', () => {
  const primaryToken = {
    provider: 'openai',
    credentialType: 'codex_auth_json',
    usage: {
      planType: 'free',
      subscriptionQuotaAvailable: true,
      primaryRemainingPercent: 80,
    },
  }
  const weeklyToken = {
    provider: 'openai',
    credentialType: 'codex_auth_json',
    usage: {
      planType: 'free',
      subscriptionQuotaAvailable: true,
      secondaryRemainingPercent: 0,
      secondaryUsedPercent: 100,
    },
  }

  assert.equal(showPrimaryQuotaWindow(primaryToken), true)
  assert.equal(showSecondaryQuotaWindow(primaryToken), false)
  assert.equal(showPrimaryQuotaWindow(weeklyToken), false)
  assert.equal(showSecondaryQuotaWindow(weeklyToken), true)
  assert.equal(weeklyLimitReached(weeklyToken), true)
})

test('Codex team plan shows the 5h quota label', () => {
  assert.equal(
    quotaPrimaryLabel({
      provider: 'openai',
      credentialType: 'codex_auth_json',
      usage: { planType: 'team' },
    }),
    '5h额度',
  )
  assert.equal(
    quotaPrimaryLabel({
      provider: 'openai',
      credentialType: 'codex_auth_json',
      usage: { planType: 'plus' },
    }),
    '5h额度',
  )
})

test('Codex quota display follows the windows returned by the backend', () => {
  const weeklyOnlyToken = {
    provider: 'openai',
    credentialType: 'codex_auth_json',
    usage: {
      planType: 'plus',
      subscriptionQuotaAvailable: true,
      secondaryUsedPercent: 45,
      secondaryRemainingPercent: 55,
    },
  }
  const bothWindowsToken = {
    provider: 'openai',
    credentialType: 'codex_auth_json',
    usage: {
      planType: 'plus',
      subscriptionQuotaAvailable: true,
      primaryRemainingPercent: 80,
      secondaryRemainingPercent: 55,
    },
  }
  const missingWindowsToken = {
    provider: 'openai',
    credentialType: 'codex_auth_json',
    usage: {
      planType: 'plus',
      subscriptionQuotaAvailable: true,
    },
  }
  const pendingToken = {
    provider: 'openai',
    credentialType: 'codex_auth_json',
    usage: { planType: 'plus' },
  }

  assert.equal(showPrimaryQuotaWindow(weeklyOnlyToken), false)
  assert.equal(showSecondaryQuotaWindow(weeklyOnlyToken), true)
  assert.equal(quotaWindowCount(weeklyOnlyToken), 1)
  assert.equal(quotaOverviewLabel(weeklyOnlyToken), '1 周额度')
  assert.equal(quotaOverviewRemainingField(weeklyOnlyToken), 'secondaryRemainingPercent')

  assert.equal(showPrimaryQuotaWindow(bothWindowsToken), true)
  assert.equal(showSecondaryQuotaWindow(bothWindowsToken), true)
  assert.equal(quotaWindowCount(bothWindowsToken), 2)
  assert.equal(quotaOverviewLabel(bothWindowsToken), '5h额度')
  assert.equal(quotaOverviewRemainingField(bothWindowsToken), 'primaryRemainingPercent')

  assert.equal(showPrimaryQuotaWindow(missingWindowsToken), false)
  assert.equal(showSecondaryQuotaWindow(missingWindowsToken), false)
  assert.equal(quotaWindowCount(missingWindowsToken), 0)
  assert.equal(quotaOverviewLabel(missingWindowsToken), '额度待刷新')
  assert.equal(quotaOverviewRemainingField(missingWindowsToken), '')

  assert.equal(showPrimaryQuotaWindow(pendingToken), false)
  assert.equal(showSecondaryQuotaWindow(pendingToken), false)
  assert.equal(quotaOverviewLabel(pendingToken), '额度待刷新')
  assert.equal(quotaOverviewRemainingField(pendingToken), '')
})

test('Codex weekly quota estimate uses current weekly tokens and remaining percent', () => {
  const resetAt = Math.floor(Date.parse('2026-06-18T00:00:00+08:00') / 1000)
  const token = {
    provider: 'openai',
    credentialType: 'codex_auth_json',
    usage: {
      subscriptionQuotaAvailable: true,
      secondaryRemainingPercent: 80,
      secondaryResetAt: resetAt,
    },
    stats: {
      inputTokens: 900000,
      outputTokens: 900000,
      totalTokens: 1800000,
      daily: [
        { date: '2026-06-01', inputTokens: 900000, outputTokens: 900000, totalTokens: 1800000 },
        { date: '2026-06-12', inputTokens: 100000, outputTokens: 10000, totalTokens: 110000 },
      ],
    },
  }

  assert.equal(codexWeeklyQuotaEstimateText(token), '$4.00 / 周')
  assert.equal(codexWeeklyQuotaEstimateMeta(token), '按当前周窗口 110,000 Token、已用成本 $0.8000 和已用 20% 估算 · OpenAI GPT-5.6 Sol')
})

test('Codex weekly quota estimate prices cache tokens like sub2api', () => {
  const resetAt = Math.floor(Date.parse('2026-06-18T00:00:00+08:00') / 1000)
  const token = {
    provider: 'openai',
    credentialType: 'codex_auth_json',
    usage: {
      subscriptionQuotaAvailable: true,
      secondaryRemainingPercent: 80,
      secondaryResetAt: resetAt,
    },
    stats: {
      daily: [
        {
          date: '2026-06-12',
          inputTokens: 120000,
          outputTokens: 10000,
          totalTokens: 130000,
          cacheCreationTokens: 5000,
          cacheReadTokens: 100000,
        },
      ],
    },
  }

  assert.equal(codexWeeklyQuotaEstimateText(token), '$2.41 / 周')
  assert.equal(codexWeeklyQuotaEstimateMeta(token), '按当前周窗口 130,000 Token、已用成本 $0.4813 和已用 20% 估算 · OpenAI GPT-5.6 Sol')
})

test('Codex weekly quota estimate applies GPT-5.6 Sol long-context pricing', () => {
  const resetAt = Math.floor(Date.parse('2026-06-18T00:00:00+08:00') / 1000)
  const token = {
    provider: 'openai',
    credentialType: 'codex_auth_json',
    usage: {
      subscriptionQuotaAvailable: true,
      secondaryRemainingPercent: 80,
      secondaryResetAt: resetAt,
    },
    stats: {
      daily: [
        {
          date: '2026-06-12',
          inputTokens: 300000,
          outputTokens: 50000,
          totalTokens: 350000,
        },
      ],
    },
  }

  assert.equal(codexWeeklyQuotaEstimateText(token), '$26.25 / 周')
  assert.equal(codexWeeklyQuotaEstimateMeta(token), '按当前周窗口 350,000 Token、已用成本 $5.25 和已用 20% 估算 · OpenAI GPT-5.6 Sol')
})

test('Codex weekly quota estimate stays hidden without consumed weekly quota', () => {
  assert.equal(
    codexWeeklyQuotaEstimateText({
      provider: 'openai',
      credentialType: 'codex_auth_json',
      usage: {
        subscriptionQuotaAvailable: true,
        secondaryRemainingPercent: 100,
      },
      stats: {
        totalTokens: 1000,
      },
    }),
    '',
  )
})

test('Codex weekly quota estimate stays hidden without current weekly usage rows', () => {
  assert.equal(
    codexWeeklyQuotaEstimateText({
      provider: 'openai',
      credentialType: 'codex_auth_json',
      usage: {
        subscriptionQuotaAvailable: true,
        secondaryRemainingPercent: 80,
      },
      stats: {
        inputTokens: 900000,
        outputTokens: 900000,
        totalTokens: 1800000,
      },
    }),
    '',
  )
})

test('cooling tokens report warning status class', () => {
  assert.equal(displayStatusClass({ cooldownUntil: new Date(Date.now() + 60000).toISOString() }), 'warning')
})

test('Codex reset credit helpers normalize count, records, and status', () => {
  const item = {
    usage: {
      codexResetCreditsAvailable: 2.9,
      codexResetCredits: [{ id: 'credit-1', status: 'redeemed', resetType: 'five_hour' }],
    },
  }

  assert.equal(codexResetCreditsAvailable(item), 2)
  assert.deepEqual(codexResetCredits(item), item.usage.codexResetCredits)
  assert.deepEqual(codexResetCreditStatusMeta(item.usage.codexResetCredits[0]), { label: '已使用', type: 'info' })
  assert.equal(codexResetCreditTypeLabel(item.usage.codexResetCredits[0]), 'five hour')
  assert.deepEqual(codexResetCreditStatusMeta({ status: 'available', expiresAt: 1 }), { label: '已过期', type: 'warning' })
  assert.equal(codexResetCreditTypeLabel({ resetType: 'codex_rate_limits' }), 'Codex 限额刷新')
  assert.equal(codexResetCreditsAvailable({}), null)
  assert.deepEqual(codexResetCredits({}), [])
})
