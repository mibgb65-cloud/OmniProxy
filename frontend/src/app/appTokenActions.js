import { ElMessageBox } from 'element-plus'
import {
  consumeCodexResetCredit,
  createToken,
  deleteToken,
  importAPIKeys,
  refreshTokenAuth,
  setTokenDisabled,
  setTokenSelected,
  updateToken,
  validateToken,
} from '../services/api'
import { codexResetCreditsAvailable, isCodexToken, validationSuccessMessage } from '../utils/tokenDisplay'
import { createCodexLoginActions } from './appCodexLoginActions'
import { codexIdentityFromAuthJSON } from './codexAuth'

function codexIdentityKey(email, accountId = '') {
  const normalizedEmail = String(email || '').trim().toLowerCase()
  const normalizedAccountId = String(accountId || '').trim()
  return normalizedAccountId ? `${normalizedEmail}::${normalizedAccountId}` : normalizedEmail
}
function rememberCodexToken(map, item, identity = {}) {
  if (!item) return
  const email = item.name || identity.email
  const accountId = item.accountId || identity.accountId || ''
  const key = codexIdentityKey(email, accountId)
  if (key) map.set(key, item)
  if (!accountId && email) map.set(codexIdentityKey(email), item)
}

export function createTokenActions(state, derived, tokenHelpers, dataActions) {
  function openCreateForm(provider = 'openai') {
    Object.assign(state.form, {
      visible: true,
      editingId: '',
      name: '',
      provider,
      originalProvider: provider,
      credentialType: 'api_key',
      originalCredentialType: 'api_key',
      region: 'cn',
      baseUrl: tokenHelpers.providerDefaultBaseUrl(provider),
      tokenValue: '',
    })
  }

  function openEditForm(token) {
    Object.assign(state.form, {
      visible: true,
      editingId: token.id,
      name: token.name,
      provider: token.provider,
      originalProvider: token.provider,
      credentialType: token.credentialType || 'api_key',
      originalCredentialType: token.credentialType || 'api_key',
      region: token.region || 'cn',
      baseUrl: token.baseUrl || '',
      tokenValue: '',
    })
  }

  function closeForm() {
    state.form.visible = false
  }

  async function submitForm() {
    state.errorMessage.value = ''
    state.successMessage.value = ''
    const name = derived.isAutoNameForm.value ? '' : state.form.name.trim()
    const tokenValue = state.form.tokenValue.trim()
    const provider = state.form.provider.trim() || 'openai'
    const credentialType = tokenHelpers.normalizedCredentialType(provider, state.form.credentialType)
    const region = provider === 'xiaomi' && credentialType === 'mimo_token_plan' ? state.form.region || 'cn' : ''
    const baseUrl = tokenHelpers.providerRequiresBaseUrl(provider) ? state.form.baseUrl.trim() : ''
    const isEditing = Boolean(state.form.editingId)
    const replacingCredential = tokenValue !== ''

    if (!derived.isAutoNameForm.value && !name) {
      state.errorMessage.value = '账号名称不能为空'
      return
    }
    const duplicate = state.tokens.value.some(
      (item) =>
        !derived.isAutoNameForm.value &&
        item.id !== state.form.editingId &&
        item.provider === provider &&
        item.name.toLowerCase() === name.toLowerCase(),
    )
    if (duplicate) {
      state.errorMessage.value = '同一厂商下账号名称不可重复'
      return
    }
    if (
      isEditing &&
      !replacingCredential &&
      (provider !== state.form.originalProvider || credentialType !== state.form.originalCredentialType)
    ) {
      state.errorMessage.value = '更改厂商或凭据类型时需要重新填写凭据'
      return
    }
    if (!tokenHelpers.validateProviderBaseUrl(provider, baseUrl)) {
      return
    }
    if ((credentialType === 'codex_auth_json' || credentialType === 'claude_oauth_json') && (!isEditing || replacingCredential)) {
      try {
        const parsed = JSON.parse(tokenValue)
        if (
          credentialType === 'claude_oauth_json' &&
          !parsed.access_token &&
          !parsed.accessToken &&
          !parsed.refresh_token &&
          !parsed.refreshToken &&
          !parsed.claudeAiOauth
        ) {
          state.errorMessage.value = 'Claude OAuth JSON 需要包含 access_token 或 refresh_token'
          return
        }
      } catch {
        state.errorMessage.value = credentialType === 'claude_oauth_json' ? 'Claude OAuth JSON 不是有效 JSON' : 'Codex auth.json 内容不是有效 JSON'
        return
      }
    } else if (replacingCredential && provider === 'xiaomi' && credentialType === 'mimo_token_plan' && !tokenValue.startsWith('tp-')) {
      state.errorMessage.value = 'MiMo Token Plan Key 必须以 tp- 开头'
      return
    } else if (replacingCredential && provider === 'xiaomi' && credentialType === 'api_key' && !tokenValue.startsWith('sk-')) {
      state.errorMessage.value = 'MiMo 按量 API Key 必须以 sk- 开头'
      return
    } else if (replacingCredential && provider === 'tokenrouter' && !tokenValue.startsWith('tr_')) {
      state.errorMessage.value = 'TokenRouter API Key 必须以 tr_ 开头'
      return
    } else if ((!isEditing || replacingCredential) && tokenValue.length < 12) {
      state.errorMessage.value = 'Token 长度过短'
      return
    }

    const payload = {
      name,
      provider,
      credentialType,
      region,
      baseUrl,
      tokenValue,
    }

    try {
      if (state.form.editingId) {
        await updateToken(state.form.editingId, payload)
      } else {
        await createToken(payload)
      }
      closeForm()
      await dataActions.refreshAll()
      if (provider === 'openrouter') {
        await dataActions.refreshOpenRouterModels({ force: true })
      }
      state.successMessage.value = state.form.editingId ? '账号已更新' : '账号已添加'
    } catch (error) {
      state.errorMessage.value = error.message
    }
  }

  function openBatchImport(provider = 'openai') {
    Object.assign(state.batchImportForm, {
      visible: true,
      provider,
      baseUrl: tokenHelpers.providerDefaultBaseUrl(provider),
      tokenText: '',
    })
  }

  function closeBatchImport() {
    if (state.batchImporting.value) return
    state.batchImportForm.visible = false
  }

  function onBatchImportProviderChange() {
    if (tokenHelpers.providerRequiresBaseUrl(state.batchImportForm.provider) && !state.batchImportForm.baseUrl) {
      state.batchImportForm.baseUrl = tokenHelpers.providerDefaultBaseUrl(state.batchImportForm.provider)
    } else if (!tokenHelpers.providerRequiresBaseUrl(state.batchImportForm.provider)) {
      state.batchImportForm.baseUrl = ''
    }
  }

  function batchImportPlaceholder() {
    if (state.batchImportForm.provider === 'zo') {
      return [
        'zo_sk_xxxxxxxxxxxxxxxxxxxxxxxx',
        'zo_sk_yyyyyyyyyyyyyyyyyyyyyyyy',
      ].join('\n')
    }
    if (state.batchImportForm.provider === 'prem') {
      return [
        'prem-key-xxxxxxxxxxxxxxxxxxxxxxxx',
        'prem-key-yyyyyyyyyyyyyyyyyyyyyyyy',
      ].join('\n')
    }
    return [
      'sk-aa0aeaf480484648a8a93d672d76334d  # balance: 10.14 CNY',
      'sk-460d28e38c7e4b05a13fa2bebd27159c  # balance: 0.24 USD',
      'sk-3d7acb8511ad4da18e8b0c89733f472b  # balance: 7.18 USD',
    ].join('\n')
  }

  async function submitBatchImport() {
    state.errorMessage.value = ''
    state.successMessage.value = ''
    const provider = state.batchImportForm.provider.trim() || 'openai'
    const baseUrl = tokenHelpers.providerRequiresBaseUrl(provider) ? state.batchImportForm.baseUrl.trim() : ''
    const tokenText = state.batchImportForm.tokenText.trim()

    if (!tokenText) {
      state.errorMessage.value = '请先粘贴要导入的 API Key'
      return
    }
    if (!tokenHelpers.validateProviderBaseUrl(provider, baseUrl)) {
      return
    }

    state.batchImporting.value = true
    try {
      const result = await importAPIKeys({
        provider,
        credentialType: 'api_key',
        region: '',
        baseUrl,
        tokenText,
      })
      state.batchImportForm.visible = false
      state.activeProvider.value = provider
      await dataActions.refreshAll()
      if (provider === 'openrouter' && result.createdCount) {
        await dataActions.refreshOpenRouterModels({ force: true })
      }

      const created = result.createdCount || 0
      const skipped = result.skipped?.length || 0
      if (created > 0) {
        state.successMessage.value = `已导入 ${created} 个 API Key${skipped ? `，跳过 ${skipped} 行` : ''}`
      } else {
        state.errorMessage.value = skipped ? `没有导入新的 API Key，已跳过 ${skipped} 行` : '没有导入新的 API Key'
      }
    } catch (error) {
      state.errorMessage.value = error.message
    } finally {
      state.batchImporting.value = false
    }
  }

  async function removeToken(token) {
    state.deleteCandidate.value = token
  }

  function closeDeleteConfirm() {
    if (state.deleteBusy.value) return
    state.deleteCandidate.value = null
  }

  async function confirmRemoveToken() {
    if (!state.deleteCandidate.value?.id) return
    const target = state.deleteCandidate.value
    state.deleteBusy.value = true
    try {
      await deleteToken(target.id)
      await dataActions.refreshAll()
      state.successMessage.value = `账号已删除：${target.name}`
      state.deleteCandidate.value = null
    } catch (error) {
      state.errorMessage.value = error.message
    } finally {
      state.deleteBusy.value = false
    }
  }

  function replaceToken(updated) {
    if (!updated?.id) return
    state.tokens.value = state.tokens.value.map((item) => (item.id === updated.id ? updated : item))
  }

  async function toggleTokenEnabled(token, enabled = Boolean(token.disabled)) {
    state.errorMessage.value = ''
    state.successMessage.value = ''
    state.togglingTokenIds[token.id] = true
    try {
      const nextEnabled = Boolean(enabled)
      const updated = await setTokenDisabled(token.id, !nextEnabled)
      replaceToken(updated)
      state.successMessage.value = nextEnabled ? `已启用账号：${updated.name}` : `已停用账号：${updated.name}`
    } catch (error) {
      state.errorMessage.value = error.message
      await dataActions.refreshRealtime()
    } finally {
      state.togglingTokenIds[token.id] = false
    }
  }

  function providerSelectedTokens(provider) {
    return tokenHelpers.providerTokens(provider).filter((item) => item.selected)
  }

  async function toggleTokenSelected(token) {
    if (!token?.id) return
    if (token.disabled) {
      state.errorMessage.value = '已停用账号需要先在账号管理中启用'
      return
    }
    state.errorMessage.value = ''
    state.successMessage.value = ''
    state.switchingOnlyTokenIds[token.id] = true
    try {
      const nextSelected = !token.selected
      const updatedTokens = await setTokenSelected(token.id, nextSelected)
      if (Array.isArray(updatedTokens)) {
        state.tokens.value = updatedTokens
      } else {
        await dataActions.refreshRealtime()
      }
      const selectedCount = Array.isArray(updatedTokens)
        ? updatedTokens.filter((item) => item.provider === token.provider && item.selected).length
        : providerSelectedTokens(token.provider).length
      if (nextSelected) {
        state.successMessage.value = `已选择 ${tokenHelpers.providerLabel(token.provider)} 账号：${token.name}，低额度时会切到其他可用账号`
      } else if (selectedCount > 0) {
        state.successMessage.value = `已取消选择 ${token.name}，${tokenHelpers.providerLabel(token.provider)} 会优先使用仍已选账号`
      } else {
        state.successMessage.value = `已恢复 ${tokenHelpers.providerLabel(token.provider)} 默认轮换`
      }
    } catch (error) {
      state.errorMessage.value = error.message
      await dataActions.refreshRealtime()
    } finally {
      state.switchingOnlyTokenIds[token.id] = false
    }
  }

  async function selectTokenGroup(tokens) {
    const items = Array.isArray(tokens) ? tokens.filter((item) => item?.id && !item.disabled && !item.selected) : []
    if (!items.length) {
      state.successMessage.value = '该账号下的工作区已全部选中'
      return
    }
    state.errorMessage.value = ''
    state.successMessage.value = ''
    try {
      for (const item of items) {
        state.switchingOnlyTokenIds[item.id] = true
        const updatedTokens = await setTokenSelected(item.id, true)
        if (Array.isArray(updatedTokens)) {
          state.tokens.value = updatedTokens
        } else {
          await dataActions.refreshRealtime()
        }
      }
      state.successMessage.value = `已全选 ${items[0].name} 的 ${items.length} 个工作区`
    } catch (error) {
      state.errorMessage.value = error.message
      await dataActions.refreshRealtime()
    } finally {
      for (const item of items) {
        state.switchingOnlyTokenIds[item.id] = false
      }
    }
  }

  async function verifyToken(token) {
    state.errorMessage.value = ''
    state.successMessage.value = ''
    state.validatingIds[token.id] = true
    try {
      const result = await validateToken(token.id)
      await dataActions.refreshRealtime()
      if (result.ok) {
        state.successMessage.value = validationSuccessMessage(token, result)
      } else {
        state.errorMessage.value = `验证未通过：${result.status || '-'} ${result.message || ''}`
      }
    } catch (error) {
      state.errorMessage.value = error.message
    } finally {
      state.validatingIds[token.id] = false
    }
  }

  async function refreshAuthToken(token) {
    if (!isCodexToken(token)) {
      state.errorMessage.value = '当前账号不支持刷新令牌'
      return
    }
    state.errorMessage.value = ''
    state.successMessage.value = ''
    state.refreshingTokenIds[token.id] = true
    try {
      const updated = await refreshTokenAuth(token.id)
      replaceToken(updated)
      await dataActions.refreshRealtime()
      state.successMessage.value = `令牌已刷新：${updated.name}`
    } catch (error) {
      state.errorMessage.value = error.message
    } finally {
      state.refreshingTokenIds[token.id] = false
    }
  }
  async function useCodexResetCredit(item) {
    if (!item?.id || state.consumingResetCreditIds[item.id]) return
    const available = codexResetCreditsAvailable(item)
    if (!available) {
      state.errorMessage.value = '当前账号没有可用的额度刷新卡'
      return
    }
    try {
      await ElMessageBox.confirm(
        `将消耗 1 张额度刷新卡，立即重置当前 5 小时额度。当前可用 ${available} 张。`,
        '使用额度刷新卡',
        {
          confirmButtonText: '确认使用',
          cancelButtonText: '取消',
          type: 'warning',
        },
      )
    } catch (action) {
      if (action instanceof Error) state.errorMessage.value = action.message
      return
    }
    state.errorMessage.value = ''
    state.successMessage.value = ''
    state.consumingResetCreditIds[item.id] = true
    try {
      const result = await consumeCodexResetCredit(item.id)
      replaceToken(result.token)
      await dataActions.refreshRealtime()
      state.successMessage.value = result.refreshError
        ? `${result.message}：${result.refreshError}`
        : result.message || '额度刷新卡已使用'
    } catch (error) {
      state.errorMessage.value = error.message
    } finally {
      state.consumingResetCreditIds[item.id] = false
    }
  }
  function openCodexAuthFilePicker() {
    state.errorMessage.value = ''
    state.successMessage.value = ''
  }

  async function importCodexAuthFiles(event) {
    const fileInput = event.target
    const files = Array.from(fileInput.files || [])
    fileInput.value = ''
    if (!files.length) {
      return
    }

    state.activeProvider.value = 'openai'
    state.errorMessage.value = ''
    state.successMessage.value = ''
    state.codexAuthImporting.value = true

    const knownCodexTokens = new Map()
    const knownOpenAINonCodexTokens = new Map()
    state.tokens.value.forEach((item) => {
      if (item.provider !== 'openai') return
      if (isCodexToken(item)) {
        rememberCodexToken(knownCodexTokens, item)
      } else {
        knownOpenAINonCodexTokens.set(item.name.toLowerCase(), item)
      }
    })

    const summary = {
      created: 0,
      updated: 0,
      failed: [],
    }

    try {
      for (const file of files) {
        try {
          const tokenValue = (await file.text()).trim()
          const identity = codexIdentityFromAuthJSON(tokenValue)
          const emailKey = identity.email.toLowerCase()
          if (knownOpenAINonCodexTokens.has(emailKey)) {
            throw new Error(`同名 OpenAI 账号已存在，且不是 Codex auth.json`)
          }

          const payload = {
            name: '',
            provider: 'openai',
            credentialType: 'codex_auth_json',
            tokenValue,
          }
          const key = codexIdentityKey(identity.email, identity.accountId)
          const fallbackKey = identity.accountId ? codexIdentityKey(identity.email) : ''
          const existing = knownCodexTokens.get(key) || (fallbackKey ? knownCodexTokens.get(fallbackKey) : null)
          if (existing) {
            const updated = await updateToken(existing.id, payload)
            rememberCodexToken(knownCodexTokens, updated, identity)
            summary.updated += 1
          } else {
            const created = await createToken(payload)
            rememberCodexToken(knownCodexTokens, created, identity)
            summary.created += 1
          }
        } catch (error) {
          summary.failed.push(`${file.name}: ${error.message}`)
        }
      }

      await dataActions.refreshAll()
      const importedCount = summary.created + summary.updated
      if (importedCount) {
        const parts = []
        if (summary.created) parts.push(`新增 ${summary.created} 个`)
        if (summary.updated) parts.push(`更新 ${summary.updated} 个`)
        state.successMessage.value = `Codex auth 文件导入完成：${parts.join('，')}`
      }
      if (summary.failed.length) {
        state.errorMessage.value = `导入失败 ${summary.failed.length} 个：${summary.failed.slice(0, 3).join('；')}`
      }
      if (!importedCount && !summary.failed.length) {
        state.successMessage.value = '没有可导入的 auth 文件'
      }
    } finally {
      state.codexAuthImporting.value = false
    }
  }

  function onProviderChange() {
    state.form.credentialType = tokenHelpers.normalizedCredentialType(state.form.provider, state.form.credentialType)
    if (tokenHelpers.providerRequiresBaseUrl(state.form.provider) && !state.form.baseUrl) {
      state.form.baseUrl = tokenHelpers.providerDefaultBaseUrl(state.form.provider)
    } else if (!tokenHelpers.providerRequiresBaseUrl(state.form.provider)) {
      state.form.baseUrl = ''
    }
    if (state.form.editingId && state.form.provider !== state.form.originalProvider) {
      state.form.tokenValue = ''
    }
  }

  function resetQuotaRefreshProgress() {
    Object.assign(state.quotaRefreshProgress, {
      visible: false,
      percent: 0,
      total: 0,
      completed: 0,
      failed: 0,
      providerLabel: '',
      currentName: '',
    })
  }

  function wait(ms) {
    return new Promise((resolve) => {
      window.setTimeout(resolve, ms)
    })
  }

  function startQuotaRefreshProgress(items) {
    Object.assign(state.quotaRefreshProgress, {
      visible: true,
      percent: 10,
      total: items.length,
      completed: 0,
      failed: 0,
      providerLabel: derived.activeProviderInfo.value.label,
      currentName: items[0]?.name || '',
    })
  }

  function updateQuotaRefreshProgress({ completed, failed, currentName, done = false }) {
    state.quotaRefreshProgress.completed = completed
    state.quotaRefreshProgress.failed = failed
    state.quotaRefreshProgress.currentName = currentName || state.quotaRefreshProgress.currentName
    state.quotaRefreshProgress.percent = done
      ? 100
      : Math.min(96, Math.max(10, Math.round(10 + (completed / Math.max(state.quotaRefreshProgress.total, 1)) * 84)))
  }

  async function refreshProviderQuotas() {
    if (state.refreshingProvider.value) return

    const items = derived.activeProviderTokens.value.filter((item) => !item.disabled)
    if (!items.length) {
      state.successMessage.value = `暂无启用的 ${derived.activeProviderInfo.value.label} 账号可刷新`
      return
    }

    state.errorMessage.value = ''
    state.successMessage.value = ''
    state.refreshingProvider.value = true
    let failed = 0
    let completed = 0
    let finalErrorMessage = ''
    let finalSuccessMessage = ''
    startQuotaRefreshProgress(items)

    try {
      for (const item of items) {
        state.quotaRefreshProgress.currentName = item.name
        state.validatingIds[item.id] = true
        try {
          const result = await validateToken(item.id)
          if (!result?.ok) {
            failed += 1
          }
        } catch {
          failed += 1
        } finally {
          state.validatingIds[item.id] = false
          completed += 1
          updateQuotaRefreshProgress({ completed, failed, currentName: item.name })
        }
      }
      updateQuotaRefreshProgress({ completed, failed, currentName: '同步最新额度状态' })
      await dataActions.refreshRealtime()
      updateQuotaRefreshProgress({
        completed,
        failed,
        currentName: failed ? '部分账号刷新失败' : '刷新完成',
        done: true,
      })
      await wait(260)
      if (failed) {
        finalErrorMessage = `已刷新 ${items.length - failed} 个账号，${failed} 个失败`
      } else {
        finalSuccessMessage = `已刷新 ${items.length} 个 ${derived.activeProviderInfo.value.label} 账号`
      }
    } catch (error) {
      finalErrorMessage = error.message
    } finally {
      resetQuotaRefreshProgress()
      state.refreshingProvider.value = false
      if (finalErrorMessage) {
        state.errorMessage.value = finalErrorMessage
      } else if (finalSuccessMessage) {
        state.successMessage.value = finalSuccessMessage
      }
    }
  }

  async function refreshQuota(item) {
    await verifyToken(item)
  }

  return {
    openCreateForm,
    openEditForm,
    closeForm,
    submitForm,
    openBatchImport,
    closeBatchImport,
    onBatchImportProviderChange,
    batchImportPlaceholder,
    submitBatchImport,
    removeToken,
    closeDeleteConfirm,
    confirmRemoveToken,
    replaceToken,
    toggleTokenEnabled,
    providerSelectedTokens,
    toggleTokenSelected,
    selectTokenGroup,
    verifyToken,
    refreshAuthToken,
    ...createCodexLoginActions(state, dataActions, replaceToken),
    useCodexResetCredit,
    openCodexAuthFilePicker,
    importCodexAuthFiles,
    onProviderChange,
    resetQuotaRefreshProgress,
    wait,
    startQuotaRefreshProgress,
    updateQuotaRefreshProgress,
    refreshProviderQuotas,
    refreshQuota,
  }
}
