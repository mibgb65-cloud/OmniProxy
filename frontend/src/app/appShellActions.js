import { onBeforeUnmount, onMounted, watch } from 'vue'
import { claudeModelSelectionLimit, codexModelSelectionLimit, defaultCodexModels } from '../constants/claudeModels'
import {
  WindowHide,
  WindowIsMaximised,
  WindowMinimise,
  WindowToggleMaximise,
} from '../../wailsjs/runtime/runtime'

export function createAppShellActions({
  state,
  derived,
  workspace,
  update,
  dataActions,
  tokenActions,
  navigation,
}) {
  function hasWailsRuntime() {
    return typeof window !== 'undefined' && Boolean(window.runtime)
  }

  async function refreshWindowState() {
    if (!hasWailsRuntime()) return
    try {
      state.windowMaximised.value = await WindowIsMaximised()
    } catch {
      state.windowMaximised.value = false
    }
  }

  function minimiseWindow() {
    if (hasWailsRuntime()) {
      WindowMinimise()
    }
  }

  function toggleWindowMaximise() {
    if (!hasWailsRuntime()) return
    WindowToggleMaximise()
    window.setTimeout(refreshWindowState, 120)
  }

  function startWindowResize(edge) {
    if (state.windowMaximised.value || typeof window === 'undefined' || !window.WailsInvoke) {
      return
    }
    window.WailsInvoke(`resize:${edge}`)
  }

  function closeWindow() {
    if (hasWailsRuntime()) {
      WindowHide()
    }
  }

  function toggleAppTheme() {
    state.isDark.value = !state.isDark.value
  }

  function syncDocumentTheme(value) {
    if (typeof document === 'undefined') {
      return
    }
    document.documentElement.classList.toggle('dark', value)
    document.body.classList.toggle('dark', value)
  }

  function openFirstTokenForm() {
    navigation.selectTab('tokens')
    tokenActions.openCreateForm('openai')
  }

  function closeFirstUseGuide() {
    state.firstUseGuideVisible.value = false
    window.localStorage?.setItem(state.firstUseGuideStorageKey, '1')
  }

  function previousFirstUseGuideStep() {
    state.firstUseGuideStepIndex.value = Math.max(0, state.firstUseGuideStepIndex.value - 1)
  }

  function nextFirstUseGuideStep() {
    if (state.firstUseGuideStepIndex.value >= state.firstUseGuideSteps.length - 1) {
      closeFirstUseGuide()
      return
    }
    state.firstUseGuideStepIndex.value += 1
  }

  function runFirstUseGuideAction() {
    const step = derived.currentFirstUseGuideStep.value
    closeFirstUseGuide()
    if (step.actionKey === 'tokens') {
      openFirstTokenForm()
    } else if (step.actionKey === 'proxy') {
      if (!state.proxyStatus.running) {
        dataActions.toggleProxy()
      }
    } else if (step.actionKey === 'quickstart') {
      navigation.selectTab('quickstart')
    }
  }

  async function copyEndpointValue(value, label) {
    const text = String(value || '').trim()
    if (!text) return
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(text)
      } else {
        const textarea = document.createElement('textarea')
        textarea.value = text
        textarea.setAttribute('readonly', '')
        textarea.style.position = 'fixed'
        textarea.style.opacity = '0'
        document.body.appendChild(textarea)
        textarea.select()
        document.execCommand('copy')
        document.body.removeChild(textarea)
      }
      state.successMessage.value = `${label || '内容'}已复制`
    } catch (error) {
      state.errorMessage.value = `复制失败：${error.message}`
    }
  }

  function startRealtimeTimer() {
    if (state.timers.realtimeTimer) return
    state.timers.realtimeTimer = window.setInterval(dataActions.refreshRealtime, 3000)
  }

  function stopRealtimeTimer() {
    if (!state.timers.realtimeTimer) return
    window.clearInterval(state.timers.realtimeTimer)
    state.timers.realtimeTimer = null
  }

  // Closing the window only hides it to the tray, so polling would otherwise
  // keep running against a UI nobody can see.
  function handleVisibilityChange() {
    if (document.hidden) {
      stopRealtimeTimer()
      return
    }
    startRealtimeTimer()
    void dataActions.refreshRealtime()
  }

  onMounted(async () => {
    const savedAppTheme = window.localStorage?.getItem(state.appThemeStorageKey)
    if (savedAppTheme === 'dark' || savedAppTheme === 'light') {
      state.isDark.value = savedAppTheme === 'dark'
    } else if (window.matchMedia?.('(prefers-color-scheme: dark)').matches) {
      state.isDark.value = true
    }
    syncDocumentTheme(state.isDark.value)
    await refreshWindowState()
    window.addEventListener('resize', refreshWindowState)
    window.addEventListener('keydown', update.handleTitlebarUpdateKeydown)
    document.addEventListener('pointerdown', update.handleTitlebarUpdateOutsidePointer)
    await dataActions.refreshAll()
    update.notifyCompletedUpdateIfNeeded()
    await update.refreshUpdateDownloadStatus()
    update.scheduleUpdateChecks()
    startRealtimeTimer()
    document.addEventListener('visibilitychange', handleVisibilityChange)
  })

  onBeforeUnmount(() => {
    window.removeEventListener('resize', refreshWindowState)
    window.removeEventListener('keydown', update.handleTitlebarUpdateKeydown)
    document.removeEventListener('pointerdown', update.handleTitlebarUpdateOutsidePointer)
    document.removeEventListener('visibilitychange', handleVisibilityChange)
    stopRealtimeTimer()
    update.stopAppUpdateTimers()
    if (state.timers.toastTimer) {
      window.clearTimeout(state.timers.toastTimer)
      state.timers.toastTimer = null
    }
    workspace.disposeWorkspaceScroll()
  })

  watch([state.errorMessage, state.successMessage], ([error, success]) => {
    if (state.timers.toastTimer) {
      window.clearTimeout(state.timers.toastTimer)
      state.timers.toastTimer = null
    }
    if (!error && !success) {
      return
    }
    state.timers.toastTimer = window.setTimeout(() => {
      state.errorMessage.value = ''
      state.successMessage.value = ''
      state.timers.toastTimer = null
    }, state.toastAutoCloseMs)
  })

  watch(state.isDark, (value) => {
    window.localStorage?.setItem(state.appThemeStorageKey, value ? 'dark' : 'light')
    syncDocumentTheme(value)
  })

  watch(state.activeTab, (tab, previousTab) => {
    workspace.saveWorkspaceScrollPosition(previousTab)
    workspace.pauseWorkspaceScrollSaving()
    workspace.hideWorkspaceScrollbar()
    if (tab === 'history') {
      dataActions.refreshHistory()
    } else if (tab === 'billing') {
      dataActions.refreshBilling()
    } else if (tab === 'tokens' && state.activeProvider.value === 'openrouter') {
      dataActions.refreshOpenRouterModels()
    } else if (tab === 'openrouter-chat') {
      dataActions.refreshOpenRouterModels()
    } else if (tab === 'settings') {
      dataActions.refreshTaskAutomationBrowserProfiles()
    } else if (tab === 'about') {
      dataActions.refreshUpdateDiagnostics()
    }
  })

  watch(
    () => [state.config.taskAutomationLaunchMode, state.config.taskAutomationBrowser],
    () => {
      if (state.activeTab.value === 'settings') {
        dataActions.refreshTaskAutomationBrowserProfiles()
      }
    },
  )

  watch(state.activeProvider, (provider) => {
    if (state.activeTab.value === 'tokens' && provider === 'openrouter') {
      dataActions.refreshOpenRouterModels()
    }
  })

  watch(state.openRouterModels, (models) => {
    if (!state.selectedOpenRouterChatModel.value && models.length) {
      state.selectedOpenRouterChatModel.value = models[0].id
    }
  })

  watch(state.selectedCodexModels, (models) => {
    const normalized = []
    const sourceModels = Array.isArray(models) ? models : [...defaultCodexModels]
    for (const model of sourceModels) {
      if (!model || normalized.includes(model)) continue
      normalized.push(model)
      if (normalized.length >= codexModelSelectionLimit) break
    }
    if (normalized.length !== sourceModels.length || normalized.some((model, index) => model !== sourceModels[index])) {
      state.selectedCodexModels.value = normalized
    }
  })

  watch(state.selectedClaudeModels, (models) => {
    const normalized = []
    for (const model of models) {
      if (!model || normalized.includes(model)) continue
      normalized.push(model)
      if (normalized.length >= claudeModelSelectionLimit) break
    }
    if (normalized.length !== models.length || normalized.some((model, index) => model !== models[index])) {
      state.selectedClaudeModels.value = normalized
    }
  })

  watch(derived.subscriptionOverviewPageCount, (count) => {
    state.subscriptionQuotaPage.value = derived.clampQuotaOverviewPage(state.subscriptionQuotaPage.value, count)
  })

  watch(derived.apiOverviewPageCount, (count) => {
    state.apiQuotaPage.value = derived.clampQuotaOverviewPage(state.apiQuotaPage.value, count)
  })

  return {
    hasWailsRuntime,
    refreshWindowState,
    minimiseWindow,
    toggleWindowMaximise,
    startWindowResize,
    closeWindow,
    toggleAppTheme,
    selectTab: navigation.selectTab,
    syncDocumentTheme,
    openFirstTokenForm,
    closeFirstUseGuide,
    previousFirstUseGuideStep,
    nextFirstUseGuideStep,
    runFirstUseGuideAction,
    copyEndpointValue,
  }
}
