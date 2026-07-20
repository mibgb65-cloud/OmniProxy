import { ElMessageBox } from 'element-plus'
import { applyLoadedConfig } from './appDataMerge.js'
import { localDateKey } from '../utils/format'
import {
  chooseDataDirectory as chooseDataDirectoryWithDialog,
  clearBillingUsage,
  clearRequestHistory,
  createConfigSnapshot,
  deleteConfigSnapshot,
  exportCodexAuthFiles,
  exportConfigBundle,
  exportDiagnosticsBundle,
  exportHistory,
  exportTokens,
  getActiveRequests,
  getAppInfo,
  getAutoStartStatus,
  getBillingDates,
  getBillingSummary,
  getBillingUsage,
  getClientConfigPreviews,
  getConfig,
  importConfigBundle,
  listConfigSnapshots,
  getDataDirectory,
  getHistory,
  getHistorySummary,
  getLogs,
  getOpenRouterModels,
  getProxyStatus,
  getTaskAutomationBrowserProfiles,
  getTokens,
  getUpdateDiagnostics,
  rebuildHistorySummaries,
  restoreConfigSnapshot,
  setAutoStart,
  startProxy,
  stopProxy,
} from '../services/api'

export function createAppDataActions(state, navigation) {
  async function refreshAll() {
    if (state.timers.refreshAllInFlight) return
    state.timers.refreshAllInFlight = true
    state.timers.realtimeRefreshSeq += 1
    state.loading.value = true
    state.errorMessage.value = ''
    try {
      const [
        loadedTokens,
        loadedConfig,
        loadedLogs,
        loadedStatus,
        loadedActiveRequests,
        loadedHistory,
        loadedBillingSummary,
        loadedDataDirectory,
        loadedAutoStart,
        loadedAppInfo,
        loadedUpdateDiagnostics,
        loadedConfigSnapshots,
        loadedClientConfigPreviews,
      ] = await Promise.all([
        getTokens(),
        getConfig(),
        getLogs(),
        getProxyStatus(),
        getActiveRequests(),
        getHistory({ limit: 200 }),
        getBillingSummary(30),
        getDataDirectory(),
        getAutoStartStatus(),
        getAppInfo(),
        getUpdateDiagnostics().catch(() => null),
        listConfigSnapshots().catch(() => []),
        getClientConfigPreviews().catch(() => []),
      ])
      state.tokens.value = loadedTokens
      state.logs.value = loadedLogs
      state.activeRequests.value = loadedActiveRequests
      state.requestHistory.value = loadedHistory
      state.billingSummary.value = loadedBillingSummary || state.emptyBillingSummary()
      applyLoadedConfig(state.config, loadedConfig, state.routeDraftsDirty.value)
      Object.assign(state.proxyStatus, loadedStatus)
      Object.assign(state.dataDirectory, loadedDataDirectory, {
        pendingDataDir: '',
        restartRequired: false,
      })
      state.autoStartEnabled.value = Boolean(loadedAutoStart?.enabled)
      Object.assign(state.appInfo, loadedAppInfo)
      state.updateDiagnostics.value = loadedUpdateDiagnostics || null
      state.configSnapshots.value = Array.isArray(loadedConfigSnapshots) ? loadedConfigSnapshots : []
      state.clientConfigPreviews.value = Array.isArray(loadedClientConfigPreviews) ? loadedClientConfigPreviews : []
      if (state.activeTab.value === 'history') {
        await refreshHistory()
      } else if (state.activeTab.value === 'billing') {
        await refreshBilling()
      }
    } catch (error) {
      state.errorMessage.value = error.message
    } finally {
      state.timers.refreshAllInFlight = false
      state.loading.value = false
    }
  }

  async function refreshRealtime() {
    if (state.timers.refreshAllInFlight) return
    const seq = ++state.timers.realtimeRefreshSeq
    try {
      const requests = [
        getLogs(),
        getProxyStatus(),
        getTokens(),
        getActiveRequests(),
        getBillingSummary(30),
      ]
      if (state.activeTab.value !== 'history') {
        requests.push(getHistory({ limit: 200 }))
      }
      const [loadedLogs, loadedStatus, loadedTokens, loadedActiveRequests, loadedBillingSummary, loadedHistory] =
        await Promise.all(requests)
      if (seq !== state.timers.realtimeRefreshSeq) return
      state.logs.value = loadedLogs
      state.tokens.value = loadedTokens
      state.activeRequests.value = loadedActiveRequests
      state.billingSummary.value = loadedBillingSummary || state.emptyBillingSummary()
      if (loadedHistory) {
        state.requestHistory.value = loadedHistory
      }
      Object.assign(state.proxyStatus, loadedStatus)
      if (state.activeTab.value === 'history') {
        await refreshHistory()
      } else if (state.activeTab.value === 'billing') {
        await refreshBilling()
      }
    } catch (error) {
      if (seq !== state.timers.realtimeRefreshSeq) return
      state.errorMessage.value = error.message
    }
  }

  async function refreshHistory(filters = state.requestHistoryFilters.value) {
    try {
      const seq = ++state.timers.historyRefreshSeq
      const normalizedFilters = { ...(filters || {}) }
      state.requestHistoryFilters.value = normalizedFilters
      const [entries, summary] = await Promise.all([
        getHistory(normalizedFilters),
        getHistorySummary(normalizedFilters, 14),
      ])
      if (seq !== state.timers.historyRefreshSeq) return
      state.requestHistory.value = entries
      state.requestHistorySummary.value = summary
    } catch (error) {
      state.errorMessage.value = error.message
    }
  }

  async function refreshBilling(date = state.selectedBillingDate.value) {
    const seq = ++state.timers.billingRefreshSeq
    try {
      const normalizedDate = String(date || localDateKey()).trim() || localDateKey()
      state.selectedBillingDate.value = normalizedDate
      const [usage, dates, summary] = await Promise.all([
        getBillingUsage(normalizedDate),
        getBillingDates(30),
        getBillingSummary(30),
      ])
      if (seq !== state.timers.billingRefreshSeq) return
      state.billingUsage.value = usage || []
      state.billingDates.value = dates || []
      state.billingSummary.value = summary || state.emptyBillingSummary()
    } catch (error) {
      if (seq !== state.timers.billingRefreshSeq) return
      state.errorMessage.value = error.message
    }
  }

  async function refreshOpenRouterModels({ force = false } = {}) {
    if (state.openRouterModelsLoading.value) {
      return
    }
    state.openRouterModelsLoading.value = true
    state.openRouterModelsError.value = ''
    try {
      const result = await getOpenRouterModels(force)
      state.openRouterModels.value = result?.models || []
      state.openRouterModelsFetchedAt.value = result?.fetchedAt || ''
      state.openRouterModelsCached.value = Boolean(result?.cached)
    } catch (error) {
      state.openRouterModelsError.value = error.message
    } finally {
      state.openRouterModelsLoading.value = false
    }
  }

  async function refreshUpdateDiagnostics() {
    state.updateDiagnosticsLoading.value = true
    state.errorMessage.value = ''
    try {
      state.updateDiagnostics.value = await getUpdateDiagnostics()
    } catch (error) {
      state.errorMessage.value = error.message
    } finally {
      state.updateDiagnosticsLoading.value = false
    }
  }

  function openOpenRouterChat(model) {
    const modelId = typeof model === 'string' ? model : model?.id
    if (modelId) {
      state.selectedOpenRouterChatModel.value = modelId
    }
    navigation.selectTab('openrouter-chat')
    if (!state.openRouterModels.value.length && !state.openRouterModelsLoading.value) {
      refreshOpenRouterModels()
    }
  }

  function selectOpenRouterChatModel(modelId) {
    state.selectedOpenRouterChatModel.value = String(modelId || '').trim()
  }

  async function changeBillingDate(date) {
    await refreshBilling(date)
  }

  function isTaskAutomationLinuxDOMode() {
    const mode = String(state.config.taskAutomationLaunchMode || '').trim().toLowerCase()
    return mode === 'linuxdo' || mode === 'linux.do' || mode === 'linux-do' || mode === 'browser'
  }

  async function refreshTaskAutomationBrowserProfiles(browser = state.config.taskAutomationBrowser) {
    if (!isTaskAutomationLinuxDOMode()) {
      state.timers.taskAutomationBrowserProfileSeq += 1
      state.taskAutomationBrowserProfiles.value = []
      state.taskAutomationBrowserProfilesError.value = ''
      state.taskAutomationBrowserProfilesLoading.value = false
      return
    }
    const seq = ++state.timers.taskAutomationBrowserProfileSeq
    state.taskAutomationBrowserProfilesLoading.value = true
    state.taskAutomationBrowserProfilesError.value = ''
    try {
      const profiles = await getTaskAutomationBrowserProfiles(browser || 'default')
      if (seq !== state.timers.taskAutomationBrowserProfileSeq) return
      state.taskAutomationBrowserProfiles.value = Array.isArray(profiles) ? profiles : []
    } catch (error) {
      if (seq !== state.timers.taskAutomationBrowserProfileSeq) return
      state.taskAutomationBrowserProfiles.value = []
      state.taskAutomationBrowserProfilesError.value = error.message
    } finally {
      if (seq === state.timers.taskAutomationBrowserProfileSeq) {
        state.taskAutomationBrowserProfilesLoading.value = false
      }
    }
  }

  function openBillingView() {
    if (state.activeTab.value === 'billing') {
      refreshBilling()
      return
    }
    navigation.selectTab('billing')
  }

  async function clearBillingUsageData() {
    try {
      await ElMessageBox.confirm(
        '将删除本地账单汇总和累计代理 Token 统计，详细请求历史不会删除。此操作无法撤销。',
        '清空账单汇总',
        {
          confirmButtonText: '清空汇总',
          cancelButtonText: '取消',
          type: 'warning',
        },
      )
      state.clearingBillingUsage.value = true
      state.errorMessage.value = ''
      await clearBillingUsage()
      state.billingUsage.value = []
      state.billingDates.value = []
      state.billingSummary.value = state.emptyBillingSummary()
      state.successMessage.value = '账单汇总已清空'
    } catch (action) {
      if (action instanceof Error) {
        state.errorMessage.value = action.message
      }
    } finally {
      state.clearingBillingUsage.value = false
    }
  }

  async function clearRequestHistoryData() {
    try {
      await ElMessageBox.confirm(
        '将删除本地请求历史明细，已保存的每日汇总会保留。此操作无法撤销。',
        '清空请求历史',
        {
          confirmButtonText: '清空历史',
          cancelButtonText: '取消',
          type: 'warning',
        },
      )
      state.clearingRequestHistory.value = true
      state.errorMessage.value = ''
      await clearRequestHistory()
      await Promise.all([refreshHistory(), refreshBilling()])
      state.successMessage.value = '请求历史已清空'
    } catch (action) {
      if (action instanceof Error) {
        state.errorMessage.value = action.message
      }
    } finally {
      state.clearingRequestHistory.value = false
    }
  }

  async function rebuildHistorySummaryData() {
    try {
      await ElMessageBox.confirm(
        '将根据当前保留的请求历史重新生成账单汇总、每日汇总和账号/工作区统计。请求历史明细不会删除。',
        '重建历史汇总',
        {
          confirmButtonText: '重建汇总',
          cancelButtonText: '取消',
          type: 'warning',
        },
      )
      state.rebuildingHistorySummaries.value = true
      state.errorMessage.value = ''
      state.successMessage.value = ''
      await rebuildHistorySummaries()
      await Promise.all([refreshHistory(), refreshBilling()])
      state.successMessage.value = '历史与账单汇总已重建'
    } catch (action) {
      if (action instanceof Error) {
        state.errorMessage.value = action.message
      }
    } finally {
      state.rebuildingHistorySummaries.value = false
    }
  }

  async function chooseDataDirectory() {
    state.dataDirChanging.value = true
    state.errorMessage.value = ''
    try {
      const result = await chooseDataDirectoryWithDialog(true)
      if (result.cancelled) {
        return
      }
      Object.assign(state.dataDirectory, {
        bootstrapPath: result.bootstrapPath,
        envOverride: result.envOverride,
        pendingDataDir: result.dataDir,
        restartRequired: result.restartRequired,
      })
      if (!result.restartRequired) {
        state.dataDirectory.dataDir = result.dataDir
        state.successMessage.value = '数据目录已保存'
        return
      }
      const copied = result.migratedFiles?.length ? `，已迁移 ${result.migratedFiles.join('、')}` : ''
      const skipped = result.skippedFiles?.length ? `，目标目录已有 ${result.skippedFiles.join('、')} 未覆盖` : ''
      state.successMessage.value = `数据目录已保存，重启 OmniProxy 后生效${copied}${skipped}`
    } catch (error) {
      state.errorMessage.value = error.message
    } finally {
      state.dataDirChanging.value = false
    }
  }

  async function toggleProxy() {
    try {
      if (state.proxyStatus.running) {
        await stopProxy()
      } else {
        await startProxy()
      }
      await refreshRealtime()
      state.successMessage.value = state.proxyStatus.running ? '代理已启动' : '代理已停止'
    } catch (error) {
      state.errorMessage.value = error.message
    }
  }

  async function toggleAutoStart() {
    state.autoStartChanging.value = true
    state.errorMessage.value = ''
    state.successMessage.value = ''
    try {
      const next = !state.autoStartEnabled.value
      const result = await setAutoStart(next)
      state.autoStartEnabled.value = Boolean(result?.enabled)
      state.successMessage.value = state.autoStartEnabled.value ? '已启用开机自启' : '已关闭开机自启'
    } catch (error) {
      state.errorMessage.value = error.message
    } finally {
      state.autoStartChanging.value = false
    }
  }

  async function exportRequestHistory(payload) {
    const format = payload?.format
    const filters = payload?.filters || {}
    const entries = payload?.entries || []
    if (!entries.length) {
      state.errorMessage.value = '当前筛选条件下没有可导出的请求历史'
      return
    }
    state.exportingHistory.value = format
    state.errorMessage.value = ''
    state.successMessage.value = ''
    try {
      const path = await exportHistory(format, filters, entries)
      if (path) {
        state.successMessage.value = `请求历史已导出为 ${format.toUpperCase()}`
      }
    } catch (error) {
      state.errorMessage.value = error.message
    } finally {
      state.exportingHistory.value = ''
    }
  }

  async function exportTokenBackup() {
    state.exportingTokens.value = true
    state.errorMessage.value = ''
    state.successMessage.value = ''
    try {
      const result = await exportTokens()
      if (result?.path) {
        state.successMessage.value = result.message || `账号池已导出：${result.count || 0} 个账号`
      }
    } catch (error) {
      state.errorMessage.value = error.message
    } finally {
      state.exportingTokens.value = false
    }
  }

  async function exportCodexAuthBackups() {
    state.exportingCodexAuth.value = true
    state.errorMessage.value = ''
    state.successMessage.value = ''
    try {
      const result = await exportCodexAuthFiles()
      if (result?.directory) {
        state.successMessage.value = result.message || `Codex auth.json 已导出：${result.count || 0} 个文件`
      }
    } catch (error) {
      state.errorMessage.value = error.message
    } finally {
      state.exportingCodexAuth.value = false
    }
  }

  async function exportDiagnostics() {
    state.exportingDiagnostics.value = true
    state.errorMessage.value = ''
    state.successMessage.value = ''
    try {
      const result = await exportDiagnosticsBundle()
      if (result?.path || result?.fileName) {
        state.successMessage.value = `诊断包已导出：${result.fileName || result.path}`
      }
    } catch (error) {
      state.errorMessage.value = error.message
    } finally {
      state.exportingDiagnostics.value = false
    }
  }

  async function refreshConfigSnapshots() {
    try {
      state.configSnapshots.value = await listConfigSnapshots()
    } catch (error) {
      state.errorMessage.value = error.message
    }
  }

  async function createCurrentConfigSnapshot() {
    state.configSnapshotBusy.value = 'create'
    state.errorMessage.value = ''
    state.successMessage.value = ''
    try {
      await createConfigSnapshot('')
      await refreshConfigSnapshots()
      state.successMessage.value = '配置快照已创建'
    } catch (error) {
      state.errorMessage.value = error.message
    } finally {
      state.configSnapshotBusy.value = ''
    }
  }

  async function restoreConfigSnapshotById(id) {
    try {
      await ElMessageBox.confirm('将用该快照覆盖当前应用设置。账号池不会被改动。', '恢复配置快照', {
        confirmButtonText: '恢复快照',
        cancelButtonText: '取消',
        type: 'warning',
      })
      state.configSnapshotBusy.value = id
      state.errorMessage.value = ''
      state.successMessage.value = ''
      const saved = await restoreConfigSnapshot(id)
      applyLoadedConfig(state.config, saved, false)
      state.routeDraftsDirty.value = false
      await Promise.all([refreshConfigSnapshots(), refreshRealtime()])
      state.successMessage.value = '配置快照已恢复'
    } catch (action) {
      if (action instanceof Error) {
        state.errorMessage.value = action.message
      }
    } finally {
      state.configSnapshotBusy.value = ''
    }
  }

  async function deleteConfigSnapshotById(id) {
    try {
      await ElMessageBox.confirm('将删除这个本地配置快照。此操作不会影响当前设置。', '删除配置快照', {
        confirmButtonText: '删除快照',
        cancelButtonText: '取消',
        type: 'warning',
      })
      state.configSnapshotBusy.value = id
      state.errorMessage.value = ''
      await deleteConfigSnapshot(id)
      await refreshConfigSnapshots()
      state.successMessage.value = '配置快照已删除'
    } catch (action) {
      if (action instanceof Error) {
        state.errorMessage.value = action.message
      }
    } finally {
      state.configSnapshotBusy.value = ''
    }
  }

  async function exportCurrentConfig() {
    state.exportingConfig.value = true
    state.errorMessage.value = ''
    state.successMessage.value = ''
    try {
      const result = await exportConfigBundle()
      if (result?.path || result?.fileName) {
        state.successMessage.value = `配置已导出：${result.fileName || result.path}`
      }
    } catch (error) {
      state.errorMessage.value = error.message
    } finally {
      state.exportingConfig.value = false
    }
  }

  async function importConfigFromFile(file = null) {
    try {
      await ElMessageBox.confirm('将用导入文件覆盖当前应用设置。账号池不会被导入或覆盖。', '导入配置', {
        confirmButtonText: '导入配置',
        cancelButtonText: '取消',
        type: 'warning',
      })
      state.importingConfig.value = true
      state.errorMessage.value = ''
      state.successMessage.value = ''
      const result = await importConfigBundle(file)
      if (result?.config) {
        applyLoadedConfig(state.config, result.config, false)
        state.routeDraftsDirty.value = false
      }
      await Promise.all([refreshConfigSnapshots(), refreshRealtime()])
      state.successMessage.value = result?.message || '配置已导入'
    } catch (action) {
      if (action instanceof Error) {
        state.errorMessage.value = action.message
      }
    } finally {
      state.importingConfig.value = false
    }
  }

  function closeHistoryDiagnosis() {
    state.selectedHistoryEntry.value = null
  }

  return {
    refreshAll,
    refreshRealtime,
    refreshHistory,
    refreshBilling,
    refreshOpenRouterModels,
    refreshUpdateDiagnostics,
    openOpenRouterChat,
    selectOpenRouterChatModel,
    changeBillingDate,
    isTaskAutomationLinuxDOMode,
    refreshTaskAutomationBrowserProfiles,
    openBillingView,
    clearBillingUsageData,
    clearRequestHistoryData,
    rebuildHistorySummaryData,
    chooseDataDirectory,
    toggleProxy,
    toggleAutoStart,
    exportRequestHistory,
    exportTokenBackup,
    exportCodexAuthBackups,
    exportDiagnostics,
    refreshConfigSnapshots,
    createCurrentConfigSnapshot,
    restoreConfigSnapshotById,
    deleteConfigSnapshotById,
    exportCurrentConfig,
    importConfigFromFile,
    closeHistoryDiagnosis,
  }
}
