import { strict as assert } from 'node:assert'
import test from 'node:test'

function jsonResponse(status, payload) {
  return {
    ok: status >= 200 && status < 300,
    status,
    async json() {
      return payload
    },
  }
}

async function waitFor(predicate, timeoutMs = 2500) {
  const startedAt = Date.now()
  while (!predicate()) {
    if (Date.now() - startedAt > timeoutMs) throw new Error('timed out waiting for Claude login state')
    await new Promise((resolve) => setTimeout(resolve, 25))
  }
}

test('Claude login refreshes the link and automatically imports a completed authorization', async () => {
  globalThis.__OMNIPROXY_CONTROL_TOKEN__ = 'claude-login-actions-token'
  const calls = []
  let startCount = 0
  globalThis.fetch = async (url, options = {}) => {
    const parsed = new URL(String(url))
    calls.push({ url: parsed, options })
    if (parsed.pathname.endsWith('/claude/login/start')) {
      startCount += 1
      return jsonResponse(200, {
        loginId: `login-${startCount}`,
        authUrl: `https://claude.com/cai/oauth/authorize?session=${startCount}`,
      })
    }
    if (parsed.pathname.endsWith('/claude/login/status')) {
      assert.equal(parsed.searchParams.get('loginId'), 'login-2')
      return jsonResponse(200, { ready: true })
    }
    if (parsed.pathname.endsWith('/claude/login/complete')) {
      assert.deepEqual(JSON.parse(options.body), { loginId: 'login-2' })
      return jsonResponse(200, { id: 'account-1', name: 'claude@example.com' })
    }
    throw new Error(`unexpected fetch: ${url}`)
  }

  const state = {
    claudeLoggingIn: { value: false },
    errorMessage: { value: '' },
    successMessage: { value: '' },
    activeProvider: { value: '' },
    claudeLoginDialog: {
      visible: false,
      loginId: '',
      authUrl: '',
      status: 'idle',
      message: '',
    },
  }
  let imported = null
  let realtimeRefreshes = 0
  const { createClaudeLoginActions } = await import(`./appClaudeLoginActions.js?test=${Date.now()}`)
  const actions = createClaudeLoginActions(
    state,
    { refreshRealtime: async () => { realtimeRefreshes += 1 } },
    (token) => { imported = token },
  )

  await actions.loginClaude()
  assert.equal(state.claudeLoginDialog.loginId, 'login-1')
  await actions.refreshClaudeLoginLink()
  assert.equal(state.claudeLoginDialog.loginId, 'login-2')
  assert.equal(calls[0].url.searchParams.get('refresh'), 'false')
  assert.equal(calls[1].url.searchParams.get('refresh'), 'true')

  await waitFor(() => state.claudeLoginDialog.status === 'success')
  assert.equal(imported?.id, 'account-1')
  assert.equal(realtimeRefreshes, 1)
  assert.match(state.claudeLoginDialog.message, /自动导入账号池/)

  delete globalThis.__OMNIPROXY_CONTROL_TOKEN__
  delete globalThis.fetch
})
