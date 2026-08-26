import { main as main_coreModels } from './main_core'
import { main as main_historyModels } from './main_history'
import { main as main_openrouterModels } from './main_openrouter'
import { main as main_tokensModels } from './main_tokens'
import { main as main_updateModels } from './main_update'
import { main as main_miscModels } from './main_misc'

export namespace main {
  export import activeRequestResponse = main_coreModels.activeRequestResponse
  export import apiKeyBatchImportRequest = main_coreModels.apiKeyBatchImportRequest
  export import apiKeyBatchImportSkipped = main_coreModels.apiKeyBatchImportSkipped
  export import apiKeyBatchImportResult = main_coreModels.apiKeyBatchImportResult
  export import appInfo = main_coreModels.appInfo
  export import balancePackageResponse = main_tokensModels.balancePackageResponse
  export import claudeModelsConfigureRequest = main_coreModels.claudeModelsConfigureRequest
  export import claudeOAuthLoginStartResponse = main_miscModels.claudeOAuthLoginStartResponse
  export import claudeOAuthLoginStatusResponse = main_miscModels.claudeOAuthLoginStatusResponse
  export import clientConfigPreview = main_miscModels.clientConfigPreview
  export import clientConfigureResult = main_coreModels.clientConfigureResult
  export import codexAuthExportResult = main_coreModels.codexAuthExportResult
  export import codexConfigureRequest = main_miscModels.codexConfigureRequest
  export import codexConfigureResult = main_coreModels.codexConfigureResult
  export import codexOAuthLoginStartResponse = main_miscModels.codexOAuthLoginStartResponse
  export import codexOAuthLoginStatusResponse = main_miscModels.codexOAuthLoginStatusResponse
  export import healthResponse = main_tokensModels.healthResponse
  export import tokenStatsResponse = main_tokensModels.tokenStatsResponse
  export import codexResetCreditResponse = main_miscModels.codexResetCreditResponse
  export import usageResponse = main_tokensModels.usageResponse
  export import tokenResponse = main_tokensModels.tokenResponse
  export import codexResetCreditConsumeResponse = main_miscModels.codexResetCreditConsumeResponse
  export import codexUsageActivationResponse = main_miscModels.codexUsageActivationResponse
  export import configExportResult = main_miscModels.configExportResult
  export import configImportResult = main_miscModels.configImportResult
  export import configSnapshotSummary = main_miscModels.configSnapshotSummary
  export import diagnosticsExportResult = main_miscModels.diagnosticsExportResult
  export import retryAttemptResponse = main_historyModels.retryAttemptResponse
  export import historyResponse = main_historyModels.historyResponse
  export import logResponse = main_historyModels.logResponse
  export import mimoConfigureResult = main_coreModels.mimoConfigureResult
  export import openRouterChatMessage = main_openrouterModels.openRouterChatMessage
  export import openRouterChatRequest = main_openrouterModels.openRouterChatRequest
  export import openRouterChatUsageResponse = main_openrouterModels.openRouterChatUsageResponse
  export import openRouterChatResponse = main_openrouterModels.openRouterChatResponse
  export import openRouterPricing = main_openrouterModels.openRouterPricing
  export import openRouterModelResponse = main_openrouterModels.openRouterModelResponse
  export import openRouterModelsResponse = main_openrouterModels.openRouterModelsResponse
  export import providerModelCatalogItem = main_miscModels.providerModelCatalogItem
  export import providerModelCatalogRequest = main_miscModels.providerModelCatalogRequest
  export import providerModelCatalogResponse = main_miscModels.providerModelCatalogResponse
  export import tokenExportResult = main_tokensModels.tokenExportResult
  export import updateDownloadStatus = main_updateModels.updateDownloadStatus
  export import updateDiagnostics = main_updateModels.updateDiagnostics
  export import updateDownloadRequest = main_updateModels.updateDownloadRequest
  export import updateInfo = main_updateModels.updateInfo
  export import validationResponse = main_tokensModels.validationResponse
}
