import {
  completeCodexOAuthLogin,
  getCodexOAuthLoginStatus,
  openExternalURL,
  startCodexOAuthLogin,
} from '../services/api.js'

const codexLoginPollDelay = 1200

export function createCodexLoginActions(state, dataActions, replaceToken) {
  let pollTimer = null
  let pollGeneration = 0

  function clearPollTimer() {
    if (pollTimer !== null) {
      clearTimeout(pollTimer)
      pollTimer = null
    }
  }

  function stopPolling() {
    clearPollTimer()
    pollGeneration += 1
  }

  function isCurrentLogin(loginId, generation) {
    return (
      generation === pollGeneration &&
      state.codexLoginDialog.visible &&
      state.codexLoginDialog.loginId === loginId
    )
  }

  function scheduleStatusCheck(loginId, generation) {
    if (!isCurrentLogin(loginId, generation) || state.codexLoginDialog.status !== 'ready') return
    clearPollTimer()
    pollTimer = setTimeout(() => {
      pollTimer = null
      void checkCodexLoginStatus(loginId, generation)
    }, codexLoginPollDelay)
  }

  async function startLogin(refresh = false) {
    if (state.codexLoggingIn.value) return
    stopPolling()
    const generation = pollGeneration
    state.errorMessage.value = ''
    state.successMessage.value = ''
    state.activeProvider.value = 'openai'
    state.codexLoggingIn.value = true
    Object.assign(state.codexLoginDialog, {
      visible: true,
      loginId: '',
      authUrl: '',
      status: 'starting',
      message: refresh ? '正在刷新 Codex 安全登录链接…' : '正在生成 Codex 安全登录链接…',
    })
    try {
      const login = await startCodexOAuthLogin(refresh)
      if (!login?.loginId || !login?.authUrl) {
        throw new Error('Codex 登录会话创建失败')
      }
      if (generation !== pollGeneration) return
      Object.assign(state.codexLoginDialog, {
        loginId: login.loginId,
        authUrl: login.authUrl,
        status: 'ready',
        message: '正在自动检测授权结果；授权成功后会自动导入，无需关闭弹窗。',
      })
      state.successMessage.value = refresh ? 'Codex 登录链接已刷新' : 'Codex 登录链接已生成'
      scheduleStatusCheck(login.loginId, generation)
    } catch (error) {
      if (generation !== pollGeneration) return
      state.codexLoginDialog.status = 'error'
      state.codexLoginDialog.message = error.message
      state.errorMessage.value = error.message
    } finally {
      if (generation === pollGeneration) state.codexLoggingIn.value = false
    }
  }

  async function importCodexLogin(loginId, generation) {
    if (state.codexLoggingIn.value || !isCurrentLogin(loginId, generation)) return
    clearPollTimer()
    state.errorMessage.value = ''
    state.successMessage.value = ''
    state.codexLoggingIn.value = true
    state.codexLoginDialog.status = 'completing'
    state.codexLoginDialog.message = '已检测到授权结果，正在交换凭据并按 CPA 格式导入账号…'
    try {
      const updated = await completeCodexOAuthLogin(loginId)
      if (!isCurrentLogin(loginId, generation)) return
      replaceToken(updated)
      await dataActions.refreshRealtime()
      state.codexLoginDialog.status = 'success'
      state.codexLoginDialog.message = `${updated.name} 已按 CPA 格式自动导入账号池。`
      state.successMessage.value = `Codex 登录成功：${updated.name}`
      stopPolling()
    } catch (error) {
      if (!isCurrentLogin(loginId, generation)) return
      state.codexLoginDialog.status = 'error'
      state.codexLoginDialog.message = error.message
      state.errorMessage.value = error.message
      stopPolling()
    } finally {
      state.codexLoggingIn.value = false
    }
  }

  async function checkCodexLoginStatus(loginId, generation, manual = false) {
    if (!isCurrentLogin(loginId, generation) || state.codexLoginDialog.status !== 'ready') return
    try {
      const result = await getCodexOAuthLoginStatus(loginId)
      if (!isCurrentLogin(loginId, generation)) return
      if (result?.ready) {
        await importCodexLogin(loginId, generation)
        return
      }
      if (manual) {
        state.codexLoginDialog.message = '暂未检测到授权结果，将继续自动检查。完成浏览器授权后无需再次点击。'
      }
      scheduleStatusCheck(loginId, generation)
    } catch (error) {
      if (!isCurrentLogin(loginId, generation)) return
      state.codexLoginDialog.status = 'error'
      state.codexLoginDialog.message = error.message
      state.errorMessage.value = error.message
      stopPolling()
    }
  }

  function loginCodex() {
    return startLogin(false)
  }

  function refreshCodexLoginLink() {
    return startLogin(true)
  }

  function completeCodexLogin() {
    if (state.codexLoggingIn.value || state.codexLoginDialog.status !== 'ready') return
    clearPollTimer()
    return checkCodexLoginStatus(state.codexLoginDialog.loginId, pollGeneration, true)
  }

  function closeCodexLoginDialog() {
    if (state.codexLoggingIn.value) return
    stopPolling()
    state.codexLoginDialog.visible = false
  }

  function openCodexLoginURL() {
    const authUrl = String(state.codexLoginDialog.authUrl || '').trim()
    if (authUrl) openExternalURL(authUrl)
  }

  return {
    loginCodex,
    refreshCodexLoginLink,
    completeCodexLogin,
    closeCodexLoginDialog,
    openCodexLoginURL,
  }
}
