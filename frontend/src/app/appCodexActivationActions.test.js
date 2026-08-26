import { strict as assert } from 'node:assert'
import test from 'node:test'

function jsonResponse(status, payload) {
  return {
    ok: status >= 200 && status < 300,
    status,
    async json() { return payload },
  }
}

function actionState() {
  return {
    activatingCodexUsageIds: {},
    errorMessage: { value: '' },
    successMessage: { value: '' },
  }
}

test('consolidated Codex activation processes enabled accounts sequentially', async () => {
  globalThis.__OMNIPROXY_CONTROL_TOKEN__ = 'activation-control-token'
  const requests = []
  let inFlight = 0
  let maxInFlight = 0
  globalThis.fetch = async (url) => {
    const id = String(url).match(/tokens\/(.+)\/codex-usage-activation/)?.[1]
    assert.ok(id)
    requests.push(id)
    inFlight += 1
    maxInFlight = Math.max(maxInFlight, inFlight)
    await new Promise((resolve) => setTimeout(resolve, 5))
    inFlight -= 1
    return jsonResponse(200, { activated: id === 'codex-1', token: { id } })
  }

  const state = actionState()
  const replaced = []
  let refreshes = 0
  const { createCodexActivationActions } = await import(`./appCodexActivationActions.js?success=${Date.now()}`)
  const { activateCodexUsageWindow } = createCodexActivationActions(
    state,
    { async refreshRealtime() { refreshes += 1 } },
    (token) => replaced.push(token.id),
  )
  await activateCodexUsageWindow([
    { id: 'codex-1', provider: 'openai', credentialType: 'codex_auth_json' },
    { id: 'disabled', provider: 'openai', credentialType: 'codex_auth_json', disabled: true },
    { id: 'api-key', provider: 'openai', credentialType: 'api_key' },
    { id: 'codex-2', provider: 'openai', credentialType: 'codex_auth_json' },
  ])

  assert.deepEqual(requests, ['codex-1', 'codex-2'])
  assert.equal(maxInFlight, 1)
  assert.deepEqual(replaced, ['codex-1', 'codex-2'])
  assert.equal(refreshes, 1)
  assert.equal(state.errorMessage.value, '')
  assert.equal(state.successMessage.value, '已检查 2 个 Codex 账号，激活 1 个额度窗口')
  assert.deepEqual(state.activatingCodexUsageIds, { 'codex-1': false, 'codex-2': false })
  delete globalThis.__OMNIPROXY_CONTROL_TOKEN__
})

test('consolidated Codex activation continues after an account failure', async () => {
  globalThis.__OMNIPROXY_CONTROL_TOKEN__ = 'activation-failure-token'
  const requests = []
  globalThis.fetch = async (url) => {
    const id = String(url).match(/tokens\/(.+)\/codex-usage-activation/)?.[1]
    requests.push(id)
    return id === 'codex-1'
      ? jsonResponse(500, { error: '上游失败' })
      : jsonResponse(200, { activated: false, token: { id } })
  }

  const state = actionState()
  let refreshes = 0
  const { createCodexActivationActions } = await import(`./appCodexActivationActions.js?failure=${Date.now()}`)
  const { activateCodexUsageWindow } = createCodexActivationActions(
    state,
    { async refreshRealtime() { refreshes += 1 } },
    () => {},
  )
  await activateCodexUsageWindow([
    { id: 'codex-1', provider: 'openai', credentialType: 'codex_auth_json' },
    { id: 'codex-2', provider: 'openai', credentialType: 'codex_auth_json' },
  ])

  assert.deepEqual(requests, ['codex-1', 'codex-2'])
  assert.equal(refreshes, 1)
  assert.equal(state.successMessage.value, '')
  assert.equal(state.errorMessage.value, '已检查 1 个 Codex 账号，1 个失败：上游失败')
  assert.deepEqual(state.activatingCodexUsageIds, { 'codex-1': false, 'codex-2': false })
  delete globalThis.__OMNIPROXY_CONTROL_TOKEN__
})
