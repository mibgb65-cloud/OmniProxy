import { saveConfig } from '../services/api.js'

export function gatewayRoutesPayload(routes) {
  const source = routes && typeof routes === 'object' ? routes : {}
  return {
    codex: gatewayRoutePayload(source.codex),
    claude: gatewayRoutePayload(source.claude),
    openai: gatewayRoutePayload(source.openai),
    gemini: gatewayRoutePayload(source.gemini),
  }
}

export function modelRoutesPayload(routes) {
  const source = routes && typeof routes === 'object' ? routes : {}
  return Object.fromEntries(
    Object.entries(source)
      .map(([model, route]) => [String(model || '').trim(), gatewayRoutePayload(route)])
      .filter(([model, route]) => model && route.provider),
  )
}

export function gatewayRoutePayload(route) {
  const source = route && typeof route === 'object' ? route : {}
  return {
    ...gatewayRouteTargetPayload(source),
    fallbacks: gatewayRouteFallbackPayloads(source.fallbacks),
  }
}

function gatewayRouteTargetPayload(route) {
  const source = route && typeof route === 'object' ? route : {}
  return {
    provider: String(source.provider || '').trim(),
    credentialType: String(source.credentialType || '').trim(),
    model: String(source.model || '').trim(),
  }
}

function gatewayRouteFallbackPayloads(fallbacks) {
  if (!Array.isArray(fallbacks)) {
    return []
  }
  return fallbacks
    .map((fallback) => gatewayRouteTargetPayload(fallback))
    .filter((fallback) => fallback.provider)
}

export function configPayload(config) {
  const source = config && typeof config === 'object' ? config : {}
  return {
    proxyPort: Number(source.proxyPort),
    controlPort: Number(source.controlPort),
    schedulingMode: source.schedulingMode,
    websocketMode: source.websocketMode,
    checkBetaUpdates: Boolean(source.checkBetaUpdates),
    taskAutomationEnabled: Boolean(source.taskAutomationEnabled),
    taskAutomationClients: Array.isArray(source.taskAutomationClients) ? source.taskAutomationClients : [],
    taskAutomationLaunchMode: trimText(source.taskAutomationLaunchMode),
    taskAutomationLaunchTarget: trimText(source.taskAutomationLaunchTarget),
    taskAutomationFallbackUrl: trimText(source.taskAutomationFallbackUrl),
    taskAutomationBrowser: trimText(source.taskAutomationBrowser),
    taskAutomationBrowserUserDataDir: trimText(source.taskAutomationBrowserUserDataDir),
    taskAutomationBrowserProfile: trimText(source.taskAutomationBrowserProfile),
    taskAutomationReturnToClient: Boolean(source.taskAutomationReturnToClient),
    taskAutomationIdleSeconds: Number(source.taskAutomationIdleSeconds),
    taskAutomationReturnDelaySeconds: Number(source.taskAutomationReturnDelaySeconds),
    outboundProxyEnabled: Boolean(source.outboundProxyEnabled),
    outboundProxyUrl: trimText(source.outboundProxyUrl),
    outboundProxyProviders: Array.isArray(source.outboundProxyProviders) ? source.outboundProxyProviders : [],
    outboundProxyModels: Array.isArray(source.outboundProxyModels) ? source.outboundProxyModels : [],
    upstreamBaseUrl: trimText(source.upstreamBaseUrl),
    openaiBaseUrl: trimText(source.openaiBaseUrl),
    anthropicBaseUrl: trimText(source.anthropicBaseUrl),
    deepseekBaseUrl: trimText(source.deepseekBaseUrl),
    deepseekAnthropicBaseUrl: trimText(source.deepseekAnthropicBaseUrl),
    kimiBaseUrl: trimText(source.kimiBaseUrl),
    zhipuBaseUrl: trimText(source.zhipuBaseUrl),
    zhipuAnthropicBaseUrl: trimText(source.zhipuAnthropicBaseUrl),
    minimaxBaseUrl: trimText(source.minimaxBaseUrl),
    minimaxAnthropicBaseUrl: trimText(source.minimaxAnthropicBaseUrl),
    geminiBaseUrl: trimText(source.geminiBaseUrl),
    openrouterBaseUrl: trimText(source.openrouterBaseUrl),
    tokenrouterBaseUrl: trimText(source.tokenrouterBaseUrl),
    sub2apiBaseUrl: trimText(source.sub2apiBaseUrl),
    newapiBaseUrl: trimText(source.newapiBaseUrl),
    anyrouterBaseUrl: trimText(source.anyrouterBaseUrl),
    zoBaseUrl: trimText(source.zoBaseUrl),
    premBaseUrl: trimText(source.premBaseUrl),
    premAutoStartPcciProxy: Boolean(source.premAutoStartPcciProxy),
    customGatewayBaseUrl: trimText(source.customGatewayBaseUrl),
    customGatewayAnthropicBaseUrl: trimText(source.customGatewayAnthropicBaseUrl),
    xiaomiBaseUrl: trimText(source.xiaomiBaseUrl),
    xiaomiApiBaseUrl: trimText(source.xiaomiApiBaseUrl),
    xiaomiApiAnthropicBaseUrl: trimText(source.xiaomiApiAnthropicBaseUrl),
    xiaomiTokenPlanBaseUrl: trimText(source.xiaomiTokenPlanBaseUrl),
    xiaomiTokenPlanAnthropicBaseUrl: trimText(source.xiaomiTokenPlanAnthropicBaseUrl),
    xiaomiTokenPlanSgpBaseUrl: trimText(source.xiaomiTokenPlanSgpBaseUrl),
    xiaomiTokenPlanSgpAnthropicBaseUrl: trimText(source.xiaomiTokenPlanSgpAnthropicBaseUrl),
    xiaomiTokenPlanAmsBaseUrl: trimText(source.xiaomiTokenPlanAmsBaseUrl),
    xiaomiTokenPlanAmsAnthropicBaseUrl: trimText(source.xiaomiTokenPlanAmsAnthropicBaseUrl),
    xiaomiCredentialPriority: source.xiaomiCredentialPriority,
    codexBaseUrl: trimText(source.codexBaseUrl),
    gatewayRoutes: gatewayRoutesPayload(source.gatewayRoutes),
    modelRoutes: modelRoutesPayload(source.modelRoutes),
    codexUsageEndpoint: trimText(source.codexUsageEndpoint),
    codexAutoActivateUsage: Boolean(source.codexAutoActivateUsage),
    claudeSubscriptionUsageEnabled: Boolean(source.claudeSubscriptionUsageEnabled),
    switchThreshold: Number(source.switchThreshold),
    maxRetries: Number(source.maxRetries),
    historyRetentionDays: Number(source.historyRetentionDays),
    healthWatchThreshold: Number(source.healthWatchThreshold),
    healthRiskThreshold: Number(source.healthRiskThreshold),
    longRequestAlertSeconds: Number(source.longRequestAlertSeconds),
  }
}

function trimText(value) {
  return String(value || '').trim()
}

export function createConfigActions(state, dataActions) {
  async function persistConfig() {
    try {
      const saved = await saveConfig(configPayload(state.config))
      Object.assign(state.config, saved)
      state.routeDraftsDirty.value = false
      await dataActions.refreshRealtime()
      state.successMessage.value = '设置已保存'
    } catch (error) {
      state.errorMessage.value = error.message
    }
  }

  return { persistConfig }
}
