<script setup>
import { defineAsyncComponent } from 'vue'
import AppSidebar from './AppSidebar.vue'
import AppWindowTitlebar from './AppWindowTitlebar.vue'
import AppWorkspaceToasts from './AppWorkspaceToasts.vue'
import DiagnosticDrawer from '../DiagnosticDrawer.vue'
import appIconUrl from '../../assets/appicon.png'
import { providers } from '../../constants/app'
import { claudeModelOptions, claudeModelSelectionLimit, codexModelOptions, codexModelSelectionLimit } from '../../constants/claudeModels'
import { formatDuration, formatNumber, formatTime } from '../../utils/format'
import {
  apiQuotaDisplay,
  apiQuotaMeta,
  displayStatusClass,
  displayStatusLabel,
  displayStatusType,
  formatBalance,
  healthSummary,
  quotaDisplay,
  quotaPercentText,
  quotaPercentValue,
  quotaPrimaryLabel,
  weeklyLimitReached,
} from '../../utils/tokenDisplay'
import { openExternalURL } from '../../services/api'
import { useOmniProxyApp } from '../../app/useOmniProxyApp'

const DashboardView = defineAsyncComponent(() => import('../DashboardView.vue'))
const AboutView = defineAsyncComponent(() => import('../AboutView.vue'))
const BillingView = defineAsyncComponent(() => import('../../features/billing/BillingView.vue'))
const CodexLoginModal = defineAsyncComponent(() => import('../CodexLoginModal.vue'))
const FirstUseGuideModal = defineAsyncComponent(() => import('../FirstUseGuideModal.vue'))
const GatewayRoutesView = defineAsyncComponent(() => import('../../features/gateway-routing/GatewayRoutesView.vue'))
const HistoryView = defineAsyncComponent(() => import('../../features/history/HistoryView.vue'))
const HelpView = defineAsyncComponent(() => import('../HelpView.vue'))
const LogsView = defineAsyncComponent(() => import('../LogsView.vue'))
const OpenRouterChatView = defineAsyncComponent(() => import('../../features/openrouter-chat/OpenRouterChatView.vue'))
const QuickstartView = defineAsyncComponent(() => import('../QuickstartView.vue'))
const QuotasView = defineAsyncComponent(() => import('../../features/quotas/QuotasView.vue'))
const SettingsView = defineAsyncComponent(() => import('../../features/settings/SettingsView.vue'))
const TokenBatchImportModal = defineAsyncComponent(() => import('../TokenBatchImportModal.vue'))
const TokenEditorModal = defineAsyncComponent(() => import('../TokenEditorModal.vue'))
const TokenTrendView = defineAsyncComponent(() => import('../TokenTrendView.vue'))
const TokensView = defineAsyncComponent(() => import('../../features/tokens/TokensView.vue'))

const {
  activeProvider, activeProviderAPIBalanceSummaries, activeProviderEnabledCount, activeProviderInfo, activeProviderTokens, activeRequests, activeTab, activeTokenIds,
  activeTokens, afterPageEnter, apiOverviewPageCount, apiOverviewTokens, apiQuotaPage, apiQuotaPageText, appInfo, appThemeLabel,
  autoStartChanging, autoStartEnabled, batchImportForm, batchImportPlaceholder, batchImporting, billingDates, billingUsage, canConfigureClaudeModels, canConfigureCodexModels,
  changeBillingDate, changeQuotaOverviewPage, chooseDataDirectory, claudeCliRestoring, claudeDesktopConfiguring, claudeDesktopRestoring, claudeModelsConfiguring,
  clearBillingUsageData, clearRequestHistoryData, clearingBillingUsage, clearingRequestHistory, clientConfigPreviews, clientToolLabel, closeBatchImport, closeDeleteConfirm, closeFirstUseGuide,
  closeForm, closeHistoryDiagnosis, closeTitlebarUpdatePopover, closeWindow, closeCodexLoginDialog, codexAuthImporting, codexConfiguring, codexLoginDialog, codexLoggingIn, codexRestoring, config, configSnapshotBusy,
  configSnapshots, createCurrentConfigSnapshot,
  confirmRemoveToken, confirmTitlebarUpdatePopover, configureLocalClaudeDesktopModels, configureLocalClaudeModels, configureLocalCodex, configureLocalDeepSeekTUI,
  configureLocalGemini, configureLocalOpenCode, configureLocalPi, coolingTokens, copyEndpointValue, credentialDisplay, credentialLabel, credentialPlaceholder,
  currentFirstUseGuideStep, currentTabLabel, dailyUsageRows, dashboardDailyUsageRows, dashboardSignals, dashboardTrendRows, dataDirChanging, dataDirectory,
  consumingResetCreditIds, deepSeekTUIConfiguring, deepSeekTUIRestoring, deleteBusy, deleteCandidate, deleteConfigSnapshotById, disabledTokens, errorMessage, exhaustedTokens, exportCodexAuthBackups,
  exportCurrentConfig, exportDiagnostics, exportRequestHistory, exportTokenBackup, exportingCodexAuth, exportingConfig, exportingDiagnostics, exportingHistory, exportingTokens, firstUseGuideStepIndex, firstUseGuideSteps, firstUseGuideVisible,
  form, geminiConfiguring, geminiRestoring, hasWailsRuntime, hideWorkspaceScrollbar, importCodexAuthFiles, importConfigFromFile, importingConfig, installReadyUpdateFromUpdateSurface, invalidTokens,
  isAutoNameForm, isClaudeModelOptionDisabled, isCodexModelOptionDisabled, isDark, isTokenActiveNow, lastUpdateCheckedAt, lastUpdateInfo, loading, logs,
  completeCodexLogin, loginCodex, lowTokens, manualCheckForUpdates, minimiseWindow, mobileSidebarOpen, navSections, nextFirstUseGuideStep, onBatchImportProviderChange, onProviderChange,
  openBatchImport, openBillingView, openCodexAuthFilePicker, openCodexLoginURL, openCreateForm, openEditForm, openOpenRouterChat, openRouterModels, openRouterModelsCached,
  openRouterModelsError, openRouterModelsFetchedAt, openRouterModelsLoading, openRouterTokens, opencodeConfiguring, opencodeRestoring, pagedApiOverviewTokens,
  pagedSubscriptionOverviewTokens, piConfiguring, piRestoring, persistConfig, previousFirstUseGuideStep, providerLabel, providerTokens, proxyEndpoint,
  proxyStatus, quotaOverviewRangeText, quotaRefreshProgress, rebuildHistorySummaryData, rebuildingHistorySummaries, refreshAll, refreshAuthToken, refreshBilling, refreshHistory, refreshOpenRouterModels,
  refreshProviderQuotas, refreshQuota, refreshRealtime, refreshingProvider, refreshingTokenIds, refreshTaskAutomationBrowserProfiles, refreshUpdateDiagnostics, removeToken, requestHistory,
  requestHistorySummary, requestTrendWidth, restoreActiveWorkspaceScroll, restoreConfigSnapshotById, restoreLocalClaude, restoreLocalClaudeDesktop, restoreLocalCodex, restoreLocalDeepSeekTUI,
  restoreLocalGemini, restoreLocalOpenCode, restoreLocalPi, runFirstUseGuideAction, selectOpenRouterChatModel, selectProvider, selectTab, selectTokenGroup, selectedBillingDate,
  riskTokens, routeDraftsDirty, selectedClaudeModelLabels, selectedClaudeModels, selectedCodexModelLabels, selectedCodexModels, selectedHistoryEntry, selectedOpenRouterChatModel, skipCurrentUpdate, snoozeTitlebarUpdate, startUpdateDownload, startWindowResize, submitBatchImport,
  submitForm, subscriptionOverviewPageCount, subscriptionOverviewTokens, subscriptionQuotaPage, subscriptionQuotaPageText, successMessage, switchingOnlyTokenIds,
  tabIcons, taskAutomationBrowserProfiles, taskAutomationBrowserProfilesError, taskAutomationBrowserProfilesLoading, titlebarUpdatePopoverOpen, titlebarUpdatePrompt,
  titlebarUpdateVisible, todayProxyRequests, todayProxyTokens, toggleAppTheme, toggleAutoStart, toggleProxy, toggleTitlebarUpdatePopover,
  toggleTokenEnabled, toggleTokenSelected, toggleWindowMaximise, togglingTokenIds, tokens, toolUsageDuration, toolUsageMeta, toolUsageRows,
  totalProxyInputTokens, totalProxyOutputTokens, totalProxyRequests, totalProxyTokens, trendWidth, updateChecking, updateDiagnostics, updateDiagnosticsLoading, updateDownloadStatus, validatingIds,
  useCodexResetCredit, verifyToken, watchTokens, windowMaximised, workspaceRef, workspaceScrollbarVisible, handleWorkspacePointerMove, handleWorkspaceScroll,
} = useOmniProxyApp()
</script>

<template>
  <div class="shell" :class="{ dark: isDark }">
    <AppWindowTitlebar
      :app-icon-url="appIconUrl"
      :app-info="appInfo"
      :proxy-status="proxyStatus"
      :window-maximised="windowMaximised"
      :titlebar-update-visible="titlebarUpdateVisible"
      :titlebar-update-prompt="titlebarUpdatePrompt"
      :titlebar-update-popover-open="titlebarUpdatePopoverOpen"
      @toggle-window-maximise="toggleWindowMaximise"
      @toggle-titlebar-update-popover="toggleTitlebarUpdatePopover"
      @close-titlebar-update-popover="closeTitlebarUpdatePopover"
      @confirm-titlebar-update-popover="confirmTitlebarUpdatePopover"
      @snooze-titlebar-update="snoozeTitlebarUpdate"
      @skip-current-update="skipCurrentUpdate"
      @minimise-window="minimiseWindow"
      @close-window="closeWindow"
    />
    <div
      v-if="hasWailsRuntime()"
      class="window-resize-edge window-resize-edge-right"
      aria-hidden="true"
      @mousedown.prevent.stop="startWindowResize('e-resize')"
    ></div>

    <AppSidebar
      :app-icon-url="appIconUrl"
      :mobile-sidebar-open="mobileSidebarOpen"
      :proxy-status="proxyStatus"
      :proxy-endpoint="proxyEndpoint"
      :tokens-count="tokens.length"
      :active-tokens-count="activeTokens.length"
      :config-proxy-port="config.proxyPort"
      :nav-sections="navSections"
      :active-tab="activeTab"
      :tab-icons="tabIcons"
      :is-dark="isDark"
      :app-theme-label="appThemeLabel"
      @close-mobile-sidebar="mobileSidebarOpen = false"
      @toggle-proxy="toggleProxy"
      @select-tab="selectTab"
      @toggle-app-theme="toggleAppTheme"
    />

    <main
      ref="workspaceRef"
      :class="[
        'workspace',
        {
          'openrouter-workspace': activeTab === 'openrouter-chat',
          'logs-workspace': activeTab === 'logs',
          'workspace-scrollbar-visible': workspaceScrollbarVisible,
        },
      ]"
      @pointermove="handleWorkspacePointerMove"
      @pointerleave="hideWorkspaceScrollbar"
      @scroll="handleWorkspaceScroll"
    >
      <header class="topbar">
        <button
          type="button"
          class="mobile-menu-button"
          :aria-expanded="mobileSidebarOpen"
          aria-label="打开导航"
          @click="mobileSidebarOpen = true"
        >
          <span></span>
          <span></span>
          <span></span>
        </button>
        <div class="topbar-title">
          <p class="eyebrow">本地控制台</p>
          <h1>{{ currentTabLabel }}</h1>
          <p class="topbar-subtitle">OmniProxy 桌面网关</p>
        </div>
      </header>

      <AppWorkspaceToasts
        :quota-refresh-progress="quotaRefreshProgress"
        :error-message="errorMessage"
        :success-message="successMessage"
        @clear-error="errorMessage = ''"
        @clear-success="successMessage = ''"
      />

      <Transition
        name="page-switch"
        mode="out-in"
        appear
        @before-enter="restoreActiveWorkspaceScroll"
        @after-enter="afterPageEnter"
      >
      <DashboardView
        v-if="activeTab === 'dashboard'"
        key="dashboard"
        :proxy-status="proxyStatus"
        :proxy-endpoint="proxyEndpoint"
        :dashboard-signals="dashboardSignals"
        :active-tokens="activeTokens"
        :invalid-tokens="invalidTokens"
        :low-tokens="lowTokens"
        :cooling-tokens="coolingTokens"
        :exhausted-tokens="exhaustedTokens"
        :disabled-tokens="disabledTokens"
        :watch-tokens="watchTokens"
        :risk-tokens="riskTokens"
        :total-proxy-tokens="totalProxyTokens"
        :total-proxy-input-tokens="totalProxyInputTokens"
        :total-proxy-output-tokens="totalProxyOutputTokens"
        :today-proxy-tokens="todayProxyTokens"
        :today-proxy-requests="todayProxyRequests"
        :total-proxy-requests="totalProxyRequests"
        :active-requests="activeRequests"
        :active-token-ids="activeTokenIds"
        :tool-usage-rows="toolUsageRows"
        :subscription-overview-tokens="subscriptionOverviewTokens"
        :subscription-quota-page="subscriptionQuotaPage"
        :subscription-overview-page-count="subscriptionOverviewPageCount"
        :subscription-quota-page-text="subscriptionQuotaPageText"
        :paged-subscription-overview-tokens="pagedSubscriptionOverviewTokens"
        :api-overview-tokens="apiOverviewTokens"
        :api-quota-page="apiQuotaPage"
        :api-overview-page-count="apiOverviewPageCount"
        :api-quota-page-text="apiQuotaPageText"
        :paged-api-overview-tokens="pagedApiOverviewTokens"
        :logs="logs"
        :daily-usage-rows="dailyUsageRows"
        :dashboard-trend-rows="dashboardTrendRows"
        :dashboard-daily-usage-rows="dashboardDailyUsageRows"
        :format-number="formatNumber"
        :format-time="formatTime"
        :client-tool-label="clientToolLabel"
        :tool-usage-meta="toolUsageMeta"
        :tool-usage-duration="toolUsageDuration"
        :quota-overview-range-text="quotaOverviewRangeText"
        :is-token-active-now="isTokenActiveNow"
        :weekly-limit-reached="weeklyLimitReached"
        :display-status-class="displayStatusClass"
        :display-status-label="displayStatusLabel"
        :provider-label="providerLabel"
        :quota-primary-label="quotaPrimaryLabel"
        :quota-percent-value="quotaPercentValue"
        :quota-percent-text="quotaPercentText"
        :credential-label="credentialLabel"
        :api-quota-display="apiQuotaDisplay"
        :api-quota-meta="apiQuotaMeta"
        :trend-width="trendWidth"
        :request-trend-width="requestTrendWidth"
        @refresh="refreshAll"
        @open-settings="selectTab('settings')"
        @open-billing="openBillingView"
        @open-trends="selectTab('usage-trends')"
        @change-quota-page="changeQuotaOverviewPage"
      />
      <TokenTrendView
        v-else-if="activeTab === 'usage-trends'"
        key="usage-trends"
        :daily-usage-rows="dailyUsageRows"
        :format-number="formatNumber"
        @refresh="refreshAll"
      />
      <BillingView
        v-else-if="activeTab === 'billing'"
        key="billing"
        :entries="requestHistory"
        :daily-usage="billingUsage"
        :available-dates="billingDates"
        :selected-date="selectedBillingDate"
        :format-number="formatNumber"
        @date-change="changeBillingDate"
        @refresh="refreshBilling"
      />

      <QuotasView
        v-else-if="activeTab === 'quotas'"
        key="quotas"
        :providers="providers"
        :active-provider="activeProvider"
        :active-provider-info="activeProviderInfo"
        :active-provider-tokens="activeProviderTokens"
        :active-provider-enabled-count="activeProviderEnabledCount"
        :api-balance-summaries="activeProviderAPIBalanceSummaries"
        :refreshing-provider="refreshingProvider"
        :switching-only-token-ids="switchingOnlyTokenIds"
        :validating-ids="validatingIds"
        :consuming-reset-credit-ids="consumingResetCreditIds"
        :provider-tokens="providerTokens"
        :credential-label="credentialLabel"
        :provider-label="providerLabel"
        @refresh="refreshAll"
        @refresh-provider-quotas="refreshProviderQuotas"
        @select-provider="selectProvider"
        @toggle-token-selected="toggleTokenSelected"
        @select-token-group="selectTokenGroup"
        @refresh-quota="refreshQuota"
        @use-reset-credit="useCodexResetCredit"
      />

      <TokensView
        v-else-if="activeTab === 'tokens'"
        key="tokens"
        :providers="providers"
        :active-provider="activeProvider"
        :active-provider-info="activeProviderInfo"
        :active-provider-tokens="activeProviderTokens"
        :api-balance-summaries="activeProviderAPIBalanceSummaries"
        :exporting-tokens="exportingTokens"
        :exporting-codex-auth="exportingCodexAuth"
        :codex-auth-importing="codexAuthImporting"
        :codex-logging-in="codexLoggingIn"
        :batch-importing="batchImporting"
        :open-router-models="openRouterModels"
        :open-router-models-loading="openRouterModelsLoading"
        :open-router-models-error="openRouterModelsError"
        :open-router-models-fetched-at="openRouterModelsFetchedAt"
        :open-router-models-cached="openRouterModelsCached"
        :validating-ids="validatingIds"
        :refreshing-token-ids="refreshingTokenIds"
        :toggling-token-ids="togglingTokenIds"
        :provider-tokens="providerTokens"
        :credential-label="credentialLabel"
        :credential-display="credentialDisplay"
        :display-status-type="displayStatusType"
        :display-status-label="displayStatusLabel"
        :health-summary="healthSummary"
        :format-time="formatTime"
        :format-number="formatNumber"
        :format-balance="formatBalance"
        :quota-display="quotaDisplay"
        @select-provider="selectProvider"
        @login-codex="loginCodex"
        @export-token-backup="exportTokenBackup"
        @open-codex-auth-file-picker="openCodexAuthFilePicker"
        @import-codex-auth-files="importCodexAuthFiles"
        @export-codex-auth-backups="exportCodexAuthBackups"
        @refresh-open-router-models="refreshOpenRouterModels({ force: true })"
        @open-router-model-chat="openOpenRouterChat"
        @open-create-form="openCreateForm"
        @open-batch-import="openBatchImport"
        @verify-token="verifyToken"
        @refresh-token-auth="refreshAuthToken"
        @toggle-token-enabled="toggleTokenEnabled"
        @open-edit-form="openEditForm"
        @remove-token="removeToken"
      />

      <OpenRouterChatView
        v-else-if="activeTab === 'openrouter-chat'"
        key="openrouter-chat"
        :models="openRouterModels"
        :selected-model="selectedOpenRouterChatModel"
        :models-loading="openRouterModelsLoading"
        :models-error="openRouterModelsError"
        :open-router-tokens="openRouterTokens"
        :validating-ids="validatingIds"
        :format-time="formatTime"
        :format-number="formatNumber"
        @update:selected-model="selectOpenRouterChatModel"
        @refresh-models="refreshOpenRouterModels({ force: true })"
        @refresh-token="verifyToken"
        @open-create-key="openCreateForm('openrouter')"
      />

      <HistoryView
        v-else-if="activeTab === 'history'"
        key="history"
        :entries="requestHistory"
        :summary="requestHistorySummary"
        :providers="providers"
        :exporting="exportingHistory"
        :format-time="formatTime"
        :format-duration="formatDuration"
        :format-number="formatNumber"
        :provider-label="providerLabel"
        @refresh="refreshHistory"
        @export="exportRequestHistory"
        @diagnose="selectedHistoryEntry = $event"
      />

      <LogsView
        v-else-if="activeTab === 'logs'"
        key="logs"
        :logs="logs"
        :format-time="formatTime"
        :format-duration="formatDuration"
        @refresh="refreshRealtime"
      />

      <GatewayRoutesView
        v-else-if="activeTab === 'gateway-routes'"
        key="gateway-routes"
        :config="config"
        @persist-config="persistConfig"
        @route-draft-dirty="routeDraftsDirty = true"
      />

      <SettingsView
        v-else-if="activeTab === 'settings'"
        key="settings"
        :config="config"
        :data-directory="dataDirectory"
        :data-dir-changing="dataDirChanging"
        :auto-start-changing="autoStartChanging"
        :auto-start-enabled="autoStartEnabled"
        :config-snapshots="configSnapshots"
        :config-snapshot-busy="configSnapshotBusy"
        :exporting-config="exportingConfig"
        :importing-config="importingConfig"
        :task-automation-browser-profiles="taskAutomationBrowserProfiles"
        :task-automation-browser-profiles-loading="taskAutomationBrowserProfilesLoading"
        :task-automation-browser-profiles-error="taskAutomationBrowserProfilesError"
        :clearing-billing-usage="clearingBillingUsage"
        :clearing-request-history="clearingRequestHistory"
        :rebuilding-history-summaries="rebuildingHistorySummaries"
        @persist-config="persistConfig"
        @choose-data-directory="chooseDataDirectory"
        @toggle-auto-start="toggleAutoStart"
        @create-config-snapshot="createCurrentConfigSnapshot"
        @restore-config-snapshot="restoreConfigSnapshotById"
        @delete-config-snapshot="deleteConfigSnapshotById"
        @export-config="exportCurrentConfig"
        @import-config="importConfigFromFile"
        @refresh-task-automation-browser-profiles="refreshTaskAutomationBrowserProfiles"
        @clear-billing-usage="clearBillingUsageData"
        @clear-request-history="clearRequestHistoryData"
        @rebuild-history-summaries="rebuildHistorySummaryData"
      />

      <AboutView
        v-else-if="activeTab === 'about'"
        key="about"
        :app-info="appInfo"
        :config="config"
        :data-directory="dataDirectory"
        :proxy-status="proxyStatus"
        :auto-start-enabled="autoStartEnabled"
        :update-checking="updateChecking"
        :update-info="lastUpdateInfo"
        :update-download-status="updateDownloadStatus"
        :update-checked-at="lastUpdateCheckedAt"
        :update-diagnostics="updateDiagnostics"
        :update-diagnostics-loading="updateDiagnosticsLoading"
        :exporting-diagnostics="exportingDiagnostics"
        :format-time="formatTime"
        @manual-check-for-updates="manualCheckForUpdates"
        @download-update="startUpdateDownload"
        @install-update="installReadyUpdateFromUpdateSurface"
        @open-url="openExternalURL"
        @refresh-update-diagnostics="refreshUpdateDiagnostics"
        @copy-update-diagnostics="copyEndpointValue($event, '更新诊断信息')"
        @export-diagnostics="exportDiagnostics"
      />

      <QuickstartView
        v-else-if="activeTab === 'quickstart'"
        key="quickstart"
        v-model:selected-codex-models="selectedCodexModels"
        v-model:selected-claude-models="selectedClaudeModels"
        :config="config"
        :client-config-previews="clientConfigPreviews"
        :codex-model-options="codexModelOptions"
        :codex-model-selection-limit="codexModelSelectionLimit"
        :selected-codex-model-labels="selectedCodexModelLabels"
        :can-configure-codex-models="canConfigureCodexModels"
        :is-codex-model-option-disabled="isCodexModelOptionDisabled"
        :claude-model-options="claudeModelOptions"
        :claude-model-selection-limit="claudeModelSelectionLimit"
        :selected-claude-model-labels="selectedClaudeModelLabels"
        :can-configure-claude-models="canConfigureClaudeModels"
        :is-claude-model-option-disabled="isClaudeModelOptionDisabled"
        :codex-configuring="codexConfiguring"
        :codex-restoring="codexRestoring"
        :claude-models-configuring="claudeModelsConfiguring"
        :claude-desktop-configuring="claudeDesktopConfiguring"
        :claude-desktop-restoring="claudeDesktopRestoring"
        :claude-cli-restoring="claudeCliRestoring"
        :gemini-configuring="geminiConfiguring"
        :gemini-restoring="geminiRestoring"
        :opencode-configuring="opencodeConfiguring"
        :opencode-restoring="opencodeRestoring"
        :pi-configuring="piConfiguring"
        :pi-restoring="piRestoring"
        :deep-seek-tui-configuring="deepSeekTUIConfiguring"
        :deep-seek-tui-restoring="deepSeekTUIRestoring"
        @configure-codex="configureLocalCodex"
        @restore-codex="restoreLocalCodex"
        @configure-claude-models="configureLocalClaudeModels"
        @configure-claude-desktop-models="configureLocalClaudeDesktopModels"
        @restore-claude-desktop="restoreLocalClaudeDesktop"
        @restore-claude="restoreLocalClaude"
        @configure-gemini="configureLocalGemini"
        @restore-gemini="restoreLocalGemini"
        @configure-opencode="configureLocalOpenCode"
        @restore-opencode="restoreLocalOpenCode"
        @configure-pi="configureLocalPi"
        @restore-pi="restoreLocalPi"
        @configure-deepseek-tui="configureLocalDeepSeekTUI"
        @restore-deepseek-tui="restoreLocalDeepSeekTUI"
      />

      <HelpView
        v-else-if="activeTab === 'help'"
        key="help"
        :proxy-status="proxyStatus"
        :config="config"
        :active-tokens-count="activeTokens.length"
        :token-count="tokens.length"
        :low-tokens-count="lowTokens.length"
        :invalid-tokens-count="invalidTokens.length"
        :active-requests-count="activeRequests.length"
        :today-proxy-requests="todayProxyRequests"
        :format-number="formatNumber"
        @select-tab="selectTab"
        @copy-endpoint="copyEndpointValue"
      />
      </Transition>

      <DiagnosticDrawer
        :entry="selectedHistoryEntry"
        :format-time="formatTime"
        :format-duration="formatDuration"
        :provider-label="providerLabel"
        @close="closeHistoryDiagnosis"
      />

      <Transition name="modal-pop" appear>
        <FirstUseGuideModal
          v-if="firstUseGuideVisible"
          v-model:step-index="firstUseGuideStepIndex"
          :steps="firstUseGuideSteps"
          :current-step="currentFirstUseGuideStep"
          :proxy-endpoint="proxyEndpoint"
          @close="closeFirstUseGuide"
          @previous="previousFirstUseGuideStep"
          @run-action="runFirstUseGuideAction"
          @next="nextFirstUseGuideStep"
        />
      </Transition>

      <Transition name="modal-pop" appear>
        <TokenEditorModal
          v-if="form.visible"
          :form="form"
          :providers="providers"
          :is-codex-form="isAutoNameForm"
          :placeholder="credentialPlaceholder()"
          @close="closeForm"
          @submit="submitForm"
          @provider-change="onProviderChange"
        />
      </Transition>

      <Transition name="modal-pop" appear>
        <TokenBatchImportModal
          v-if="batchImportForm.visible"
          :form="batchImportForm"
          :providers="providers"
          :placeholder="batchImportPlaceholder()"
          :importing="batchImporting"
          @close="closeBatchImport"
          @submit="submitBatchImport"
          @provider-change="onBatchImportProviderChange"
        />
      </Transition>

      <Transition name="modal-pop" appear>
        <CodexLoginModal
          v-if="codexLoginDialog.visible"
          :dialog="codexLoginDialog"
          :busy="codexLoggingIn"
          @close="closeCodexLoginDialog"
          @copy-url="copyEndpointValue"
          @open-url="openCodexLoginURL"
          @complete="completeCodexLogin"
          @retry="loginCodex"
        />
      </Transition>

      <Transition name="modal-pop" appear>
        <div v-if="deleteCandidate" class="danger-confirm-backdrop" @click.self="closeDeleteConfirm">
          <section class="danger-confirm" role="dialog" aria-modal="true" aria-labelledby="delete-token-title">
            <div class="danger-confirm-mark" aria-hidden="true">
              <span></span>
            </div>
            <div class="danger-confirm-body">
              <p class="danger-confirm-kicker">危险操作</p>
              <h2 id="delete-token-title">删除这个账号？</h2>
              <p>
                删除后该账号会立即从调度池移除，已保存的凭据也会从本机账号池删除。历史请求记录不会被清空。
              </p>
              <div class="danger-confirm-card">
                <span>账号</span>
                <strong>{{ deleteCandidate.name }}</strong>
                <small>{{ providerLabel(deleteCandidate.provider) }} · {{ credentialLabel(deleteCandidate) }}</small>
              </div>
            </div>
            <div class="danger-confirm-actions">
              <button type="button" class="ghost-button" :disabled="deleteBusy" @click="closeDeleteConfirm">
                取消
              </button>
              <button type="button" class="danger-button solid" :disabled="deleteBusy" @click="confirmRemoveToken">
                {{ deleteBusy ? '删除中' : '删除账号' }}
              </button>
            </div>
          </section>
        </div>
      </Transition>

      <div v-if="loading" class="loading">加载中...</div>
    </main>
  </div>
</template>
