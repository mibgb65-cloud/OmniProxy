import { activateCodexUsage } from '../services/api.js'
import { isCodexToken } from '../utils/tokenDisplay.js'

export function createCodexActivationActions(state, dataActions, replaceToken) {
  async function activateCodexUsageWindow(items) {
    const targets = (Array.isArray(items) ? items : [items])
      .filter((item) => item?.id && !item.disabled && isCodexToken(item))
    if (!targets.length) {
      state.errorMessage.value = '暂无启用的 Codex auth.json 账号可检测'
      return
    }
    if (targets.some((item) => state.activatingCodexUsageIds[item.id])) return

    state.errorMessage.value = ''
    state.successMessage.value = ''
    let activated = 0
    let singleMessage = 'Codex 额度窗口检查完成'
    let refreshError = ''
    const failures = []

    for (const item of targets) {
      state.activatingCodexUsageIds[item.id] = true
      try {
        const result = await activateCodexUsage(item.id)
        if (result?.token) replaceToken(result.token)
        if (result?.activated) activated += 1
        singleMessage = result?.message || singleMessage
      } catch (error) {
        failures.push(error.message)
      } finally {
        state.activatingCodexUsageIds[item.id] = false
      }
    }

    try {
      await dataActions.refreshRealtime()
    } catch (error) {
      refreshError = error.message
    }

    if (failures.length) {
      state.errorMessage.value = targets.length === 1
        ? failures[0]
        : `已检查 ${targets.length - failures.length} 个 Codex 账号，${failures.length} 个失败：${failures[0]}`
    } else if (refreshError) {
      state.errorMessage.value = refreshError
    } else if (targets.length === 1) {
      state.successMessage.value = singleMessage
    } else {
      state.successMessage.value = `已检查 ${targets.length} 个 Codex 账号，激活 ${activated} 个额度窗口`
    }
  }

  return { activateCodexUsageWindow }
}
