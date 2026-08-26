import { strict as assert } from 'node:assert'
import { readFileSync } from 'node:fs'
import test from 'node:test'

import { configPayload, gatewayRoutesPayload, modelRoutesPayload } from './appConfigActions.js'

test('gatewayRoutesPayload preserves ordered fallback route settings', () => {
  const payload = gatewayRoutesPayload({
    openai: {
      provider: ' openai ',
      credentialType: ' api_key ',
      model: ' gpt-5.4 ',
      fallbacks: [
        { provider: ' deepseek ', credentialType: '', model: ' deepseek-v4-pro[1m] ' },
        { provider: ' zhipu ', credentialType: ' coding_plan ', model: '' },
        { provider: ' ', credentialType: 'api_key', model: 'ignored' },
      ],
    },
  })

  assert.deepEqual(payload.openai, {
    provider: 'openai',
    credentialType: 'api_key',
    model: 'gpt-5.4',
    fallbacks: [
      { provider: 'deepseek', credentialType: '', model: 'deepseek-v4-pro[1m]' },
      { provider: 'zhipu', credentialType: 'coding_plan', model: '' },
    ],
  })
})

test('gatewayRoutesPayload keeps legacy routes compatible when fallbacks are omitted', () => {
  const payload = gatewayRoutesPayload({
    openai: {
      provider: ' deepseek ',
      model: ' deepseek-v4-pro[1m] ',
    },
  })

  assert.deepEqual(payload.openai, {
    provider: 'deepseek',
    credentialType: '',
    model: 'deepseek-v4-pro[1m]',
    fallbacks: [],
  })
})

test('gatewayRoutesPayload skips malformed fallback entries', () => {
  const payload = gatewayRoutesPayload({
    openai: {
      provider: 'openai',
      fallbacks: [
        null,
        'deepseek',
        { provider: ' kimi ', credentialType: ' api_key ', model: ' kimi-for-coding ' },
      ],
    },
  })

  assert.deepEqual(payload.openai.fallbacks, [
    { provider: 'kimi', credentialType: 'api_key', model: 'kimi-for-coding' },
  ])
})

test('modelRoutesPayload preserves ordered backend route settings', () => {
  const payload = modelRoutesPayload({
    ' deepseek-v4-pro ': {
      provider: ' deepseek ',
      model: ' deepseek-v4-pro ',
      fallbacks: [
        { provider: ' prem ', credentialType: ' api_key ', model: ' deepseek-v4-pro ' },
      ],
    },
  })

  assert.deepEqual(payload, {
    'deepseek-v4-pro': {
      provider: 'deepseek',
      credentialType: '',
      model: 'deepseek-v4-pro',
      fallbacks: [
        { provider: 'prem', credentialType: 'api_key', model: 'deepseek-v4-pro' },
      ],
    },
  })
})

test('configPayload preserves all top-level backend config fields', () => {
  const backendFields = topLevelBackendConfigFields()
  const payloadFields = Object.keys(configPayload({}))

  assert.deepEqual(payloadFields.toSorted(), backendFields.toSorted())
})

test('configPayload trims third-party URLs and preserves Prem autostart toggle', () => {
  const payload = configPayload({
    anyrouterBaseUrl: ' https://anyrouter.example ',
    premBaseUrl: ' http://127.0.0.1:3101 ',
    premAutoStartPcciProxy: false,
  })

  assert.equal(payload.anyrouterBaseUrl, 'https://anyrouter.example')
  assert.equal(payload.premBaseUrl, 'http://127.0.0.1:3101')
  assert.equal(payload.premAutoStartPcciProxy, false)
})

test('configPayload preserves the Codex usage auto-activation toggle', () => {
  assert.equal(configPayload({ codexAutoActivateUsage: true }).codexAutoActivateUsage, true)
  assert.equal(configPayload({ codexAutoActivateUsage: false }).codexAutoActivateUsage, false)
})

function topLevelBackendConfigFields() {
  const source = readFileSync(new URL('../../../OmniProxyBackend/internal/config/config.go', import.meta.url), 'utf8')
  const match = source.match(/type Config struct \{([\s\S]*?)\n\}/)
  assert.ok(match, 'backend Config struct should be readable')
  return [...match[1].matchAll(/`json:"([^"]+)"/g)]
    .map((item) => item[1].split(',')[0])
    .filter(Boolean)
}
