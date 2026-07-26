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

test('HTTP API fallback fetches and sends the control token', async () => {
  globalThis.__OMNIPROXY_CONTROL_TOKEN__ = 'test-control-token'
  const calls = []
  globalThis.fetch = async (url, options = {}) => {
    calls.push({ url: String(url), options })
    if (String(url).endsWith('/logs')) {
      assert.equal(options.headers['X-OmniProxy-Control-Token'], 'test-control-token')
      return jsonResponse(200, [])
    }
    throw new Error(`unexpected fetch: ${url}`)
  }

  const { getLogs } = await import(`./api.js?control-token-test=${Date.now()}`)
  assert.deepEqual(await getLogs(), [])
  assert.deepEqual(
    calls.map((call) => call.url),
    ['http://127.0.0.1:3890/api/logs'],
  )
  delete globalThis.__OMNIPROXY_CONTROL_TOKEN__
})

test('Wails API calls generated desktop bindings when runtime is available', async () => {
  const previousWindow = globalThis.window
  try {
    globalThis.window = {
      go: {
        main: {
          DesktopApp: {
            Logs: async () => [{ message: 'from desktop' }],
          },
        },
      },
    }
    globalThis.fetch = async (url) => {
      throw new Error(`unexpected fetch: ${url}`)
    }

    const { getLogs } = await import(`./api.js?wails-runtime-test=${Date.now()}`)
    assert.deepEqual(await getLogs(), [{ message: 'from desktop' }])
  } finally {
    if (previousWindow === undefined) {
      delete globalThis.window
    } else {
      globalThis.window = previousWindow
    }
  }
})

test('HTTP API fallback refreshes token auth through the control API', async () => {
  globalThis.__OMNIPROXY_CONTROL_TOKEN__ = 'refresh-control-token'
  const calls = []
  globalThis.fetch = async (url, options = {}) => {
    calls.push({ url: String(url), options })
    if (String(url).endsWith('/tokens/openai-token/refresh')) {
      assert.equal(options.method, 'POST')
      assert.equal(options.headers['X-OmniProxy-Control-Token'], 'refresh-control-token')
      return jsonResponse(200, { id: 'openai-token', name: 'coder@example.com' })
    }
    throw new Error(`unexpected fetch: ${url}`)
  }

  const { refreshTokenAuth } = await import(`./api.js?refresh-token-test=${Date.now()}`)
  assert.deepEqual(await refreshTokenAuth('openai-token'), {
    id: 'openai-token',
    name: 'coder@example.com',
  })
  assert.deepEqual(
    calls.map((call) => call.url),
    ['http://127.0.0.1:3890/api/tokens/openai-token/refresh'],
  )
  delete globalThis.__OMNIPROXY_CONTROL_TOKEN__
})

test('HTTP API fallback supports browser logins and reset credits', async () => {
  globalThis.__OMNIPROXY_CONTROL_TOKEN__ = 'codex-action-token'
  const calls = []
  globalThis.fetch = async (url, options = {}) => {
    calls.push({ url: String(url), options })
    const path = new URL(String(url)).pathname
    assert.equal(options.headers['X-OmniProxy-Control-Token'], 'codex-action-token')
    if (path.endsWith('/codex/login/start')) {
      assert.equal(new URL(String(url)).searchParams.get('refresh'), 'true')
      return jsonResponse(200, { loginId: 'login-1', authUrl: 'https://auth.openai.com/oauth/authorize' })
    }
    if (path.endsWith('/codex/login/status')) {
      assert.equal(new URL(String(url)).searchParams.get('loginId'), 'login-1')
      return jsonResponse(200, { ready: true })
    }
    if (path.endsWith('/codex/login/complete')) {
      assert.deepEqual(JSON.parse(options.body), { loginId: 'login-1' })
      return jsonResponse(200, { id: 'account-1', name: 'coder@example.com' })
    }
    if (path.endsWith('/claude/login/start')) {
      assert.equal(new URL(String(url)).searchParams.get('refresh'), 'false')
      return jsonResponse(200, { loginId: 'claude-login-1', authUrl: 'https://claude.com/cai/oauth/authorize' })
    }
    if (path.endsWith('/claude/login/status')) {
      assert.equal(new URL(String(url)).searchParams.get('loginId'), 'claude-login-1')
      return jsonResponse(200, { ready: true })
    }
    if (path.endsWith('/claude/login/complete')) {
      assert.deepEqual(JSON.parse(options.body), { loginId: 'claude-login-1' })
      return jsonResponse(200, { id: 'claude-account-1', name: 'claude@example.com' })
    }
    if (path.endsWith('/tokens/account-1/reset-credit')) {
      return jsonResponse(200, { consumed: true, token: { id: 'account-1' } })
    }
    throw new Error(`unexpected fetch: ${url}`)
  }

  const api = await import(`./api.js?codex-actions-test=${Date.now()}`)
  assert.equal((await api.startCodexOAuthLogin(true)).loginId, 'login-1')
  assert.equal((await api.getCodexOAuthLoginStatus('login-1')).ready, true)
  assert.equal((await api.completeCodexOAuthLogin('login-1')).id, 'account-1')
  assert.equal((await api.startClaudeOAuthLogin(false)).loginId, 'claude-login-1')
  assert.equal((await api.getClaudeOAuthLoginStatus('claude-login-1')).ready, true)
  assert.equal((await api.completeClaudeOAuthLogin('claude-login-1')).id, 'claude-account-1')
  assert.equal((await api.consumeCodexResetCredit('account-1')).consumed, true)
  assert.deepEqual(
    calls.map((call) => new URL(call.url).pathname),
    [
      '/api/codex/login/start',
      '/api/codex/login/status',
      '/api/codex/login/complete',
      '/api/claude/login/start',
      '/api/claude/login/status',
      '/api/claude/login/complete',
      '/api/tokens/account-1/reset-credit',
    ],
  )
  delete globalThis.__OMNIPROXY_CONTROL_TOKEN__
})

test('HTTP API fallback imports API keys through the control API', async () => {
  globalThis.__OMNIPROXY_CONTROL_TOKEN__ = 'import-control-token'
  const payload = { provider: 'openai', tokenText: 'sk-test-token-value' }
  const calls = []
  globalThis.fetch = async (url, options = {}) => {
    calls.push({ url: String(url), options })
    if (String(url).endsWith('/tokens/import-api-keys')) {
      assert.equal(options.method, 'POST')
      assert.equal(options.headers['X-OmniProxy-Control-Token'], 'import-control-token')
      assert.deepEqual(JSON.parse(options.body), payload)
      return jsonResponse(201, { createdCount: 1, skipped: [] })
    }
    throw new Error(`unexpected fetch: ${url}`)
  }

  const { importAPIKeys } = await import(`./api.js?import-api-keys-test=${Date.now()}`)
  assert.deepEqual(await importAPIKeys(payload), {
    createdCount: 1,
    skipped: [],
  })
  assert.deepEqual(
    calls.map((call) => call.url),
    ['http://127.0.0.1:3890/api/tokens/import-api-keys'],
  )
  delete globalThis.__OMNIPROXY_CONTROL_TOKEN__
})

test('HTTP API fallback fetches history summary with filters', async () => {
  globalThis.__OMNIPROXY_CONTROL_TOKEN__ = 'summary-control-token'
  const calls = []
  globalThis.fetch = async (url, options = {}) => {
    calls.push({ url: String(url), options })
    const parsed = new URL(String(url))
    if (parsed.pathname.endsWith('/history/summary')) {
      assert.equal(options.headers['X-OmniProxy-Control-Token'], 'summary-control-token')
      assert.equal(parsed.searchParams.get('provider'), 'openai')
      assert.equal(parsed.searchParams.get('model'), 'gpt-5.5')
      assert.equal(parsed.searchParams.get('tokenId'), 'workspace-a')
      assert.equal(parsed.searchParams.get('days'), '14')
      return jsonResponse(200, { total: 42, dailyRows: [] })
    }
    throw new Error(`unexpected fetch: ${url}`)
  }

  const { getHistorySummary } = await import(`./api.js?history-summary-test=${Date.now()}`)
  assert.deepEqual(await getHistorySummary({ provider: 'openai', model: 'gpt-5.5', tokenId: 'workspace-a' }, 14), {
    total: 42,
    dailyRows: [],
  })
  assert.deepEqual(
    calls.map((call) => new URL(call.url).pathname),
    ['/api/history/summary'],
  )
  delete globalThis.__OMNIPROXY_CONTROL_TOKEN__
})

test('HTTP API fallback rebuilds history summaries', async () => {
  globalThis.__OMNIPROXY_CONTROL_TOKEN__ = 'rebuild-history-token'
  const calls = []
  globalThis.fetch = async (url, options = {}) => {
    calls.push({ url: String(url), options })
    if (String(url).endsWith('/history/rebuild-summaries')) {
      assert.equal(options.method, 'POST')
      assert.equal(options.headers['X-OmniProxy-Control-Token'], 'rebuild-history-token')
      return {
        ok: true,
        status: 204,
        async json() {
          return null
        },
      }
    }
    throw new Error(`unexpected fetch: ${url}`)
  }

  const { rebuildHistorySummaries } = await import(`./api.js?history-rebuild-test=${Date.now()}`)
  assert.equal(await rebuildHistorySummaries(), null)
  assert.deepEqual(
    calls.map((call) => call.url),
    ['http://127.0.0.1:3890/api/history/rebuild-summaries'],
  )
  delete globalThis.__OMNIPROXY_CONTROL_TOKEN__
})

test('HTTP API fallback fetches billing summary', async () => {
  globalThis.__OMNIPROXY_CONTROL_TOKEN__ = 'billing-summary-token'
  const calls = []
  globalThis.fetch = async (url, options = {}) => {
    calls.push({ url: String(url), options })
    const parsed = new URL(String(url))
    if (parsed.pathname.endsWith('/billing/summary')) {
      assert.equal(options.headers['X-OmniProxy-Control-Token'], 'billing-summary-token')
      assert.equal(parsed.searchParams.get('days'), '30')
      return jsonResponse(200, { requestCount: 3, totalTokens: 2048, dailyRows: [] })
    }
    throw new Error(`unexpected fetch: ${url}`)
  }

  const { getBillingSummary } = await import(`./api.js?billing-summary-test=${Date.now()}`)
  assert.deepEqual(await getBillingSummary(30), {
    requestCount: 3,
    totalTokens: 2048,
    dailyRows: [],
  })
  assert.deepEqual(
    calls.map((call) => new URL(call.url).pathname),
    ['/api/billing/summary'],
  )
  delete globalThis.__OMNIPROXY_CONTROL_TOKEN__
})

test('HTTP API fallback fetches update diagnostics', async () => {
  globalThis.__OMNIPROXY_CONTROL_TOKEN__ = 'update-diagnostics-token'
  const calls = []
  globalThis.fetch = async (url, options = {}) => {
    calls.push({ url: String(url), options })
    if (String(url).endsWith('/update/diagnostics')) {
      assert.equal(options.headers['X-OmniProxy-Control-Token'], 'update-diagnostics-token')
      return jsonResponse(200, {
        directory: 'C:\\Temp\\OmniProxy\\updates',
        status: { state: 'idle' },
        logTail: 'ready',
      })
    }
    throw new Error(`unexpected fetch: ${url}`)
  }

  const { getUpdateDiagnostics } = await import(`./api.js?update-diagnostics-test=${Date.now()}`)
  assert.deepEqual(await getUpdateDiagnostics(), {
    directory: 'C:\\Temp\\OmniProxy\\updates',
    status: { state: 'idle' },
    logTail: 'ready',
  })
  assert.deepEqual(
    calls.map((call) => call.url),
    ['http://127.0.0.1:3890/api/update/diagnostics'],
  )
  delete globalThis.__OMNIPROXY_CONTROL_TOKEN__
})

test('HTTP API fallback diagnoses gateway routes through the control API', async () => {
  globalThis.__OMNIPROXY_CONTROL_TOKEN__ = 'gateway-diagnostic-token'
  const payload = { client: 'claude', model: 'sonnet' }
  const calls = []
  globalThis.fetch = async (url, options = {}) => {
    calls.push({ url: String(url), options })
    if (String(url).endsWith('/gateway/diagnose')) {
      assert.equal(options.method, 'POST')
      assert.equal(options.headers['X-OmniProxy-Control-Token'], 'gateway-diagnostic-token')
      assert.deepEqual(JSON.parse(options.body), payload)
      return jsonResponse(200, { ok: true, selectedIndex: 0, chain: [] })
    }
    throw new Error(`unexpected fetch: ${url}`)
  }

  const { diagnoseGatewayRoute } = await import(`./api.js?gateway-diagnose-test=${Date.now()}`)
  assert.deepEqual(await diagnoseGatewayRoute(payload), { ok: true, selectedIndex: 0, chain: [] })
  assert.deepEqual(
    calls.map((call) => call.url),
    ['http://127.0.0.1:3890/api/gateway/diagnose'],
  )
  delete globalThis.__OMNIPROXY_CONTROL_TOKEN__
})

test('HTTP API fallback manages config snapshots through the control API', async () => {
  globalThis.__OMNIPROXY_CONTROL_TOKEN__ = 'config-snapshot-token'
  const calls = []
  globalThis.fetch = async (url, options = {}) => {
    calls.push({ url: String(url), options })
    const parsed = new URL(String(url))
    if (parsed.pathname.endsWith('/config/snapshots') && !options.method) {
      return jsonResponse(200, [{ id: 'snap-1', name: 'before' }])
    }
    if (parsed.pathname.endsWith('/config/snapshots') && options.method === 'POST') {
      assert.deepEqual(JSON.parse(options.body), { name: 'manual' })
      return jsonResponse(201, { id: 'snap-2', name: 'manual' })
    }
    if (parsed.pathname.endsWith('/config/snapshots/snap-1') && options.method === 'PUT') {
      return jsonResponse(200, { proxyPort: 3000 })
    }
    throw new Error(`unexpected fetch: ${url}`)
  }

  const mod = await import(`./api.js?config-snapshot-test=${Date.now()}`)
  assert.deepEqual(await mod.listConfigSnapshots(), [{ id: 'snap-1', name: 'before' }])
  assert.deepEqual(await mod.createConfigSnapshot('manual'), { id: 'snap-2', name: 'manual' })
  assert.deepEqual(await mod.restoreConfigSnapshot('snap-1'), { proxyPort: 3000 })
  assert.deepEqual(
    calls.map((call) => new URL(call.url).pathname),
    ['/api/config/snapshots', '/api/config/snapshots', '/api/config/snapshots/snap-1'],
  )
  delete globalThis.__OMNIPROXY_CONTROL_TOKEN__
})

test('HTTP API fallback syncs provider models through the control API', async () => {
  globalThis.__OMNIPROXY_CONTROL_TOKEN__ = 'model-sync-token'
  const calls = []
  globalThis.fetch = async (url, options = {}) => {
    calls.push({ url: String(url), options })
    if (String(url).endsWith('/models/sync')) {
      assert.equal(options.method, 'POST')
      assert.equal(options.headers['X-OmniProxy-Control-Token'], 'model-sync-token')
      assert.deepEqual(JSON.parse(options.body), { provider: 'deepseek' })
      return jsonResponse(200, { provider: 'deepseek', models: [{ id: 'deepseek-test' }] })
    }
    throw new Error(`unexpected fetch: ${url}`)
  }

  const { syncProviderModels } = await import(`./api.js?provider-model-sync-test=${Date.now()}`)
  assert.deepEqual(await syncProviderModels('deepseek'), {
    provider: 'deepseek',
    models: [{ id: 'deepseek-test' }],
  })
  assert.deepEqual(
    calls.map((call) => call.url),
    ['http://127.0.0.1:3890/api/models/sync'],
  )
  delete globalThis.__OMNIPROXY_CONTROL_TOKEN__
})

test('HTTP API fallback keeps the default history page size bounded', async () => {
  globalThis.__OMNIPROXY_CONTROL_TOKEN__ = 'history-limit-token'
  const calls = []
  globalThis.fetch = async (url, options = {}) => {
    calls.push({ url: String(url), options })
    if (new URL(String(url)).pathname.endsWith('/history')) {
      return jsonResponse(200, [])
    }
    throw new Error(`unexpected fetch: ${url}`)
  }

  const { getHistory } = await import(`./api.js?history-limit-test=${Date.now()}`)

  // The history tab re-fetches on every realtime tick, so an unbounded default
  // would pull the whole table a few times a minute.
  await getHistory()
  const fallbackLimit = Number(new URL(calls[0].url).searchParams.get('limit'))
  assert.ok(
    fallbackLimit > 0 && fallbackLimit <= 1000,
    `expected a bounded default history limit, got ${fallbackLimit}`,
  )

  await getHistory({ limit: 25 })
  assert.equal(new URL(calls[1].url).searchParams.get('limit'), '25')
  delete globalThis.__OMNIPROXY_CONTROL_TOKEN__
})
