import {
  completeCodexOAuthLogin,
  openExternalURL,
  startCodexOAuthLogin,
} from '../services/api'

export function createCodexLoginActions(state, dataActions, replaceToken) {
  async function loginCodex() {
    if (state.codexLoggingIn.value) return
    state.errorMessage.value = ''
    state.successMessage.value = ''
    state.activeProvider.value = 'openai'
    state.codexLoggingIn.value = true
    Object.assign(state.codexLoginDialog, {
      visible: true,
      loginId: '',
      authUrl: '',
      status: 'starting',
      message: '正在生成 Codex 安全登录链接…',
    })
    try {
      const login = await startCodexOAuthLogin()
      if (!login?.loginId || !login?.authUrl) {
        throw new Error('Codex 登录会话创建失败')
      }
      Object.assign(state.codexLoginDialog, {
        loginId: login.loginId,
        authUrl: login.authUrl,
        status: 'ready',
        message: '复制链接或主动打开登录页面，授权完成后返回这里导入账号。',
      })
      state.successMessage.value = 'Codex 登录链接已生成'
    } catch (error) {
      state.codexLoginDialog.status = 'error'
      state.codexLoginDialog.message = error.message
      state.errorMessage.value = error.message
    } finally {
      state.codexLoggingIn.value = false
    }
  }

  async function completeCodexLogin() {
    if (state.codexLoggingIn.value || !state.codexLoginDialog.loginId) return
    state.errorMessage.value = ''
    state.successMessage.value = ''
    state.codexLoggingIn.value = true
    state.codexLoginDialog.status = 'completing'
    state.codexLoginDialog.message = '正在接收授权结果并按 CPA 格式导入账号…'
    try {
      const updated = await completeCodexOAuthLogin(state.codexLoginDialog.loginId)
      replaceToken(updated)
      await dataActions.refreshRealtime()
      state.codexLoginDialog.status = 'success'
      state.codexLoginDialog.message = `${updated.name} 已按 CPA 格式导入账号池。`
      state.successMessage.value = `Codex 登录成功：${updated.name}`
    } catch (error) {
      state.codexLoginDialog.status = 'error'
      state.codexLoginDialog.message = error.message
      state.errorMessage.value = error.message
    } finally {
      state.codexLoggingIn.value = false
    }
  }

  function closeCodexLoginDialog() {
    if (state.codexLoggingIn.value) return
    state.codexLoginDialog.visible = false
  }

  function openCodexLoginURL() {
    const authUrl = String(state.codexLoginDialog.authUrl || '').trim()
    if (authUrl) openExternalURL(authUrl)
  }

  return {
    loginCodex,
    completeCodexLogin,
    closeCodexLoginDialog,
    openCodexLoginURL,
  }
}
