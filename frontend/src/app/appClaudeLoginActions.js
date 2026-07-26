import {
  completeClaudeOAuthLogin,
  getClaudeOAuthLoginStatus,
  openExternalURL,
  startClaudeOAuthLogin,
} from '../services/api.js'

const claudeLoginPollDelay = 1200

export function createClaudeLoginActions(state, dataActions, replaceToken) {
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
      state.claudeLoginDialog.visible &&
      state.claudeLoginDialog.loginId === loginId
    )
  }

  function scheduleStatusCheck(loginId, generation) {
    if (!isCurrentLogin(loginId, generation) || state.claudeLoginDialog.status !== 'ready') return
    clearPollTimer()
    pollTimer = setTimeout(() => {
      pollTimer = null
      void checkClaudeLoginStatus(loginId, generation)
    }, claudeLoginPollDelay)
  }

  async function startLogin(refresh = false) {
    if (state.claudeLoggingIn.value) return
    stopPolling()
    const generation = pollGeneration
    state.errorMessage.value = ''
    state.successMessage.value = ''
    state.activeProvider.value = 'anthropic'
    state.claudeLoggingIn.value = true
    Object.assign(state.claudeLoginDialog, {
      visible: true,
      loginId: '',
      authUrl: '',
      status: 'starting',
      message: refresh ? '正在刷新 Claude 安全登录链接…' : '正在生成 Claude 安全登录链接…',
    })
    try {
      const login = await startClaudeOAuthLogin(refresh)
      if (!login?.loginId || !login?.authUrl) {
        throw new Error('Claude 登录会话创建失败')
      }
      if (generation !== pollGeneration) return
      Object.assign(state.claudeLoginDialog, {
        loginId: login.loginId,
        authUrl: login.authUrl,
        status: 'ready',
        message: '正在自动检测授权结果；授权成功后会自动导入，无需关闭弹窗。',
      })
      state.successMessage.value = refresh ? 'Claude 登录链接已刷新' : 'Claude 登录链接已生成'
      scheduleStatusCheck(login.loginId, generation)
    } catch (error) {
      if (generation !== pollGeneration) return
      state.claudeLoginDialog.status = 'error'
      state.claudeLoginDialog.message = error.message
      state.errorMessage.value = error.message
    } finally {
      if (generation === pollGeneration) state.claudeLoggingIn.value = false
    }
  }

  async function importClaudeLogin(loginId, generation) {
    if (state.claudeLoggingIn.value || !isCurrentLogin(loginId, generation)) return
    clearPollTimer()
    state.errorMessage.value = ''
    state.successMessage.value = ''
    state.claudeLoggingIn.value = true
    state.claudeLoginDialog.status = 'completing'
    state.claudeLoginDialog.message = '已检测到授权结果，正在交换凭据并导入 Claude OAuth 账号…'
    try {
      const updated = await completeClaudeOAuthLogin(loginId)
      if (!isCurrentLogin(loginId, generation)) return
      replaceToken(updated)
      await dataActions.refreshRealtime()
      state.claudeLoginDialog.status = 'success'
      state.claudeLoginDialog.message = `${updated.name} 已自动导入账号池。`
      state.successMessage.value = `Claude 登录成功：${updated.name}`
      stopPolling()
    } catch (error) {
      if (!isCurrentLogin(loginId, generation)) return
      state.claudeLoginDialog.status = 'error'
      state.claudeLoginDialog.message = error.message
      state.errorMessage.value = error.message
      stopPolling()
    } finally {
      state.claudeLoggingIn.value = false
    }
  }

  async function checkClaudeLoginStatus(loginId, generation, manual = false) {
    if (!isCurrentLogin(loginId, generation) || state.claudeLoginDialog.status !== 'ready') return
    try {
      const result = await getClaudeOAuthLoginStatus(loginId)
      if (!isCurrentLogin(loginId, generation)) return
      if (result?.ready) {
        await importClaudeLogin(loginId, generation)
        return
      }
      if (manual) {
        state.claudeLoginDialog.message = '暂未检测到授权结果，将继续自动检查。完成浏览器授权后无需再次点击。'
      }
      scheduleStatusCheck(loginId, generation)
    } catch (error) {
      if (!isCurrentLogin(loginId, generation)) return
      state.claudeLoginDialog.status = 'error'
      state.claudeLoginDialog.message = error.message
      state.errorMessage.value = error.message
      stopPolling()
    }
  }

  function loginClaude() {
    return startLogin(false)
  }

  function refreshClaudeLoginLink() {
    return startLogin(true)
  }

  function completeClaudeLogin() {
    if (state.claudeLoggingIn.value || state.claudeLoginDialog.status !== 'ready') return
    clearPollTimer()
    return checkClaudeLoginStatus(state.claudeLoginDialog.loginId, pollGeneration, true)
  }

  function closeClaudeLoginDialog() {
    if (state.claudeLoggingIn.value) return
    stopPolling()
    state.claudeLoginDialog.visible = false
  }

  function openClaudeLoginURL() {
    const authUrl = String(state.claudeLoginDialog.authUrl || '').trim()
    if (authUrl) openExternalURL(authUrl)
  }

  return {
    loginClaude,
    refreshClaudeLoginLink,
    completeClaudeLogin,
    closeClaudeLoginDialog,
    openClaudeLoginURL,
  }
}
