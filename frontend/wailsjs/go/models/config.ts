export namespace config {

	export class GatewayRouteConfig {
	    provider: string;
	    credentialType?: string;
	    model?: string;
	    fallbacks?: GatewayRouteConfig[];

	    static createFrom(source: any = {}) {
	        return new GatewayRouteConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.credentialType = source["credentialType"];
	        this.model = source["model"];
	        this.fallbacks = this.convertValues(source["fallbacks"], GatewayRouteConfig);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GatewayRoutes {
	    codex: GatewayRouteConfig;
	    claude: GatewayRouteConfig;
	    openai: GatewayRouteConfig;
	    gemini: GatewayRouteConfig;

	    static createFrom(source: any = {}) {
	        return new GatewayRoutes(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.codex = this.convertValues(source["codex"], GatewayRouteConfig);
	        this.claude = this.convertValues(source["claude"], GatewayRouteConfig);
	        this.openai = this.convertValues(source["openai"], GatewayRouteConfig);
	        this.gemini = this.convertValues(source["gemini"], GatewayRouteConfig);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Config {
	    proxyPort: number;
	    controlPort: number;
	    schedulingMode: string;
	    websocketMode: string;
	    checkBetaUpdates: boolean;
	    taskAutomationEnabled: boolean;
	    taskAutomationClients: string[];
	    taskAutomationLaunchMode: string;
	    taskAutomationLaunchTarget: string;
	    taskAutomationFallbackUrl: string;
	    taskAutomationBrowser: string;
	    taskAutomationBrowserUserDataDir: string;
	    taskAutomationBrowserProfile: string;
	    taskAutomationReturnToClient: boolean;
	    taskAutomationIdleSeconds: number;
	    taskAutomationReturnDelaySeconds: number;
	    outboundProxyEnabled: boolean;
	    outboundProxyUrl: string;
	    outboundProxyProviders: string[];
	    outboundProxyModels: string[];
	    upstreamBaseUrl: string;
	    openaiBaseUrl: string;
	    anthropicBaseUrl: string;
	    deepseekBaseUrl: string;
	    deepseekAnthropicBaseUrl: string;
	    kimiBaseUrl: string;
	    zhipuBaseUrl: string;
	    zhipuAnthropicBaseUrl: string;
	    minimaxBaseUrl: string;
	    minimaxAnthropicBaseUrl: string;
	    geminiBaseUrl: string;
	    openrouterBaseUrl: string;
	    tokenrouterBaseUrl: string;
	    sub2apiBaseUrl: string;
	    newapiBaseUrl: string;
	    anyrouterBaseUrl: string;
	    zoBaseUrl: string;
	    premBaseUrl: string;
	    premAutoStartPcciProxy: boolean;
	    customGatewayBaseUrl: string;
	    customGatewayAnthropicBaseUrl: string;
	    xiaomiBaseUrl: string;
	    xiaomiApiBaseUrl: string;
	    xiaomiApiAnthropicBaseUrl: string;
	    xiaomiTokenPlanBaseUrl: string;
	    xiaomiTokenPlanAnthropicBaseUrl: string;
	    xiaomiTokenPlanSgpBaseUrl: string;
	    xiaomiTokenPlanSgpAnthropicBaseUrl: string;
	    xiaomiTokenPlanAmsBaseUrl: string;
	    xiaomiTokenPlanAmsAnthropicBaseUrl: string;
	    xiaomiCredentialPriority: string;
	    codexBaseUrl: string;
	    gatewayRoutes: GatewayRoutes;
	    modelRoutes?: Record<string, GatewayRouteConfig>;
	    switchThreshold: number;
	    maxRetries: number;
	    historyRetentionDays: number;
	    healthWatchThreshold: number;
	    healthRiskThreshold: number;
	    longRequestAlertSeconds: number;
	    codexUsageEndpoint: string;
	    claudeSubscriptionUsageEnabled: boolean;

	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.proxyPort = source["proxyPort"];
	        this.controlPort = source["controlPort"];
	        this.schedulingMode = source["schedulingMode"];
	        this.websocketMode = source["websocketMode"];
	        this.checkBetaUpdates = source["checkBetaUpdates"];
	        this.taskAutomationEnabled = source["taskAutomationEnabled"];
	        this.taskAutomationClients = source["taskAutomationClients"];
	        this.taskAutomationLaunchMode = source["taskAutomationLaunchMode"];
	        this.taskAutomationLaunchTarget = source["taskAutomationLaunchTarget"];
	        this.taskAutomationFallbackUrl = source["taskAutomationFallbackUrl"];
	        this.taskAutomationBrowser = source["taskAutomationBrowser"];
	        this.taskAutomationBrowserUserDataDir = source["taskAutomationBrowserUserDataDir"];
	        this.taskAutomationBrowserProfile = source["taskAutomationBrowserProfile"];
	        this.taskAutomationReturnToClient = source["taskAutomationReturnToClient"];
	        this.taskAutomationIdleSeconds = source["taskAutomationIdleSeconds"];
	        this.taskAutomationReturnDelaySeconds = source["taskAutomationReturnDelaySeconds"];
	        this.outboundProxyEnabled = source["outboundProxyEnabled"];
	        this.outboundProxyUrl = source["outboundProxyUrl"];
	        this.outboundProxyProviders = source["outboundProxyProviders"];
	        this.outboundProxyModels = source["outboundProxyModels"];
	        this.upstreamBaseUrl = source["upstreamBaseUrl"];
	        this.openaiBaseUrl = source["openaiBaseUrl"];
	        this.anthropicBaseUrl = source["anthropicBaseUrl"];
	        this.deepseekBaseUrl = source["deepseekBaseUrl"];
	        this.deepseekAnthropicBaseUrl = source["deepseekAnthropicBaseUrl"];
	        this.kimiBaseUrl = source["kimiBaseUrl"];
	        this.zhipuBaseUrl = source["zhipuBaseUrl"];
	        this.zhipuAnthropicBaseUrl = source["zhipuAnthropicBaseUrl"];
	        this.minimaxBaseUrl = source["minimaxBaseUrl"];
	        this.minimaxAnthropicBaseUrl = source["minimaxAnthropicBaseUrl"];
	        this.geminiBaseUrl = source["geminiBaseUrl"];
	        this.openrouterBaseUrl = source["openrouterBaseUrl"];
	        this.tokenrouterBaseUrl = source["tokenrouterBaseUrl"];
	        this.sub2apiBaseUrl = source["sub2apiBaseUrl"];
	        this.newapiBaseUrl = source["newapiBaseUrl"];
	        this.anyrouterBaseUrl = source["anyrouterBaseUrl"];
	        this.zoBaseUrl = source["zoBaseUrl"];
	        this.premBaseUrl = source["premBaseUrl"];
	        this.premAutoStartPcciProxy = source["premAutoStartPcciProxy"];
	        this.customGatewayBaseUrl = source["customGatewayBaseUrl"];
	        this.customGatewayAnthropicBaseUrl = source["customGatewayAnthropicBaseUrl"];
	        this.xiaomiBaseUrl = source["xiaomiBaseUrl"];
	        this.xiaomiApiBaseUrl = source["xiaomiApiBaseUrl"];
	        this.xiaomiApiAnthropicBaseUrl = source["xiaomiApiAnthropicBaseUrl"];
	        this.xiaomiTokenPlanBaseUrl = source["xiaomiTokenPlanBaseUrl"];
	        this.xiaomiTokenPlanAnthropicBaseUrl = source["xiaomiTokenPlanAnthropicBaseUrl"];
	        this.xiaomiTokenPlanSgpBaseUrl = source["xiaomiTokenPlanSgpBaseUrl"];
	        this.xiaomiTokenPlanSgpAnthropicBaseUrl = source["xiaomiTokenPlanSgpAnthropicBaseUrl"];
	        this.xiaomiTokenPlanAmsBaseUrl = source["xiaomiTokenPlanAmsBaseUrl"];
	        this.xiaomiTokenPlanAmsAnthropicBaseUrl = source["xiaomiTokenPlanAmsAnthropicBaseUrl"];
	        this.xiaomiCredentialPriority = source["xiaomiCredentialPriority"];
	        this.codexBaseUrl = source["codexBaseUrl"];
	        this.gatewayRoutes = this.convertValues(source["gatewayRoutes"], GatewayRoutes);
	        this.modelRoutes = this.convertValues(source["modelRoutes"], GatewayRouteConfig, true);
	        this.switchThreshold = source["switchThreshold"];
	        this.maxRetries = source["maxRetries"];
	        this.historyRetentionDays = source["historyRetentionDays"];
	        this.healthWatchThreshold = source["healthWatchThreshold"];
	        this.healthRiskThreshold = source["healthRiskThreshold"];
	        this.longRequestAlertSeconds = source["longRequestAlertSeconds"];
	        this.codexUsageEndpoint = source["codexUsageEndpoint"];
	        this.claudeSubscriptionUsageEnabled = source["claudeSubscriptionUsageEnabled"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DataDirectoryChangeResult {
	    dataDir: string;
	    previousDataDir: string;
	    bootstrapPath: string;
	    envOverride: boolean;
	    migratedFiles: string[];
	    skippedFiles: string[];
	    restartRequired: boolean;
	    cancelled: boolean;

	    static createFrom(source: any = {}) {
	        return new DataDirectoryChangeResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dataDir = source["dataDir"];
	        this.previousDataDir = source["previousDataDir"];
	        this.bootstrapPath = source["bootstrapPath"];
	        this.envOverride = source["envOverride"];
	        this.migratedFiles = source["migratedFiles"];
	        this.skippedFiles = source["skippedFiles"];
	        this.restartRequired = source["restartRequired"];
	        this.cancelled = source["cancelled"];
	    }
	}
	export class DataDirectoryInfo {
	    dataDir: string;
	    bootstrapPath: string;
	    envOverride: boolean;
	    source: string;

	    static createFrom(source: any = {}) {
	        return new DataDirectoryInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dataDir = source["dataDir"];
	        this.bootstrapPath = source["bootstrapPath"];
	        this.envOverride = source["envOverride"];
	        this.source = source["source"];
	    }
	}


}
