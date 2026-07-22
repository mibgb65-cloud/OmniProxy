import { token } from './token'

export namespace main {
	export class balancePackageResponse {
	    name?: string;
	    consumeType?: string;
	    balanceRemaining?: number;
	    balanceTotal?: number;
	    unit?: string;
	    status?: string;
	    expirationTime?: string;
	    suitableModel?: string;
	    suitableScene?: string;

	    static createFrom(source: any = {}) {
	        return new balancePackageResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.consumeType = source["consumeType"];
	        this.balanceRemaining = source["balanceRemaining"];
	        this.balanceTotal = source["balanceTotal"];
	        this.unit = source["unit"];
	        this.status = source["status"];
	        this.expirationTime = source["expirationTime"];
	        this.suitableModel = source["suitableModel"];
	        this.suitableScene = source["suitableScene"];
	    }
	}
	export class healthResponse {
	    lastCheckedAt?: string;
	    nextCheckAt?: string;
	    consecutiveErrors?: number;
	    lastStatus?: number;
	    lastMessage?: string;

	    static createFrom(source: any = {}) {
	        return new healthResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lastCheckedAt = source["lastCheckedAt"];
	        this.nextCheckAt = source["nextCheckAt"];
	        this.consecutiveErrors = source["consecutiveErrors"];
	        this.lastStatus = source["lastStatus"];
	        this.lastMessage = source["lastMessage"];
	    }
	}
	export class tokenExportResult {
	    path?: string;
	    count: number;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new tokenExportResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.count = source["count"];
	        this.message = source["message"];
	    }
	}
	export class tokenStatsResponse {
	    requestCount: number;
	    inputTokens: number;
	    outputTokens: number;
	    totalTokens: number;
	    cacheCreationTokens?: number;
	    cacheReadTokens?: number;
	    lastInputTokens?: number;
	    lastOutputTokens?: number;
	    lastTotalTokens?: number;
	    lastCacheCreationTokens?: number;
	    lastCacheReadTokens?: number;
	    daily?: token.DailyTokenUsage[];
	    updatedAt?: string;

	    static createFrom(source: any = {}) {
	        return new tokenStatsResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.requestCount = source["requestCount"];
	        this.inputTokens = source["inputTokens"];
	        this.outputTokens = source["outputTokens"];
	        this.totalTokens = source["totalTokens"];
	        this.cacheCreationTokens = source["cacheCreationTokens"];
	        this.cacheReadTokens = source["cacheReadTokens"];
	        this.lastInputTokens = source["lastInputTokens"];
	        this.lastOutputTokens = source["lastOutputTokens"];
	        this.lastTotalTokens = source["lastTotalTokens"];
	        this.lastCacheCreationTokens = source["lastCacheCreationTokens"];
	        this.lastCacheReadTokens = source["lastCacheReadTokens"];
	        this.daily = this.convertValues(source["daily"], token.DailyTokenUsage);
	        this.updatedAt = source["updatedAt"];
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
	export class usageResponse {
	    source?: string;
	    planType?: string;
	    limitReached?: boolean;
	    primaryUsedPercent: number;
	    primaryUsedPercentExact?: number;
	    primaryRemainingPercent: number;
	    primaryResetAt?: number;
	    secondaryUsedPercent: number;
	    secondaryUsedPercentExact?: number;
	    secondaryRemainingPercent: number;
	    secondaryResetAt?: number;
	    apiRemaining?: number;
	    balanceRemaining?: number;
	    balanceTotal?: number;
	    balanceUsed?: number;
	    balanceUnit?: string;
	    balanceUnlimited?: boolean;
	    balancePackages?: balancePackageResponse[];
	    codexResetCreditsAvailable?: number;
	    codexResetCredits?: codexResetCreditResponse[];
	    codexResetCreditsNextExpiresAt?: number;
	    codexResetCreditsError?: string;
	    subscriptionQuotaAvailable?: boolean;
	    message?: string;
	    updatedAt?: string;

	    static createFrom(source: any = {}) {
	        return new usageResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.planType = source["planType"];
	        this.limitReached = source["limitReached"];
	        this.primaryUsedPercent = source["primaryUsedPercent"];
	        this.primaryUsedPercentExact = source["primaryUsedPercentExact"];
	        this.primaryRemainingPercent = source["primaryRemainingPercent"];
	        this.primaryResetAt = source["primaryResetAt"];
	        this.secondaryUsedPercent = source["secondaryUsedPercent"];
	        this.secondaryUsedPercentExact = source["secondaryUsedPercentExact"];
	        this.secondaryRemainingPercent = source["secondaryRemainingPercent"];
	        this.secondaryResetAt = source["secondaryResetAt"];
	        this.apiRemaining = source["apiRemaining"];
	        this.balanceRemaining = source["balanceRemaining"];
	        this.balanceTotal = source["balanceTotal"];
	        this.balanceUsed = source["balanceUsed"];
	        this.balanceUnit = source["balanceUnit"];
	        this.balanceUnlimited = source["balanceUnlimited"];
	        this.balancePackages = this.convertValues(source["balancePackages"], balancePackageResponse);
	        this.codexResetCreditsAvailable = source["codexResetCreditsAvailable"];
	        this.codexResetCredits = this.convertValues(source["codexResetCredits"], codexResetCreditResponse);
	        this.codexResetCreditsNextExpiresAt = source["codexResetCreditsNextExpiresAt"];
	        this.codexResetCreditsError = source["codexResetCreditsError"];
	        this.subscriptionQuotaAvailable = source["subscriptionQuotaAvailable"];
	        this.message = source["message"];
	        this.updatedAt = source["updatedAt"];
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
	export class tokenResponse {
	    id: string;
	    name: string;
	    provider: string;
	    credentialType: string;
	    accountId?: string;
	    region?: string;
	    baseUrl?: string;
	    hasTokenValue: boolean;
	    maskedTokenValue?: string;
	    remaining: number;
	    usage: usageResponse;
	    stats: tokenStatsResponse;
	    health: healthResponse;
	    healthScore: number;
	    healthLevel?: string;
	    healthMessage?: string;
	    status: string;
	    disabled: boolean;
	    selected: boolean;
	    lastUsedAt?: string;
	    lastError?: string;
	    cooldownUntil?: string;
	    createdAt: string;
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new tokenResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.provider = source["provider"];
	        this.credentialType = source["credentialType"];
	        this.accountId = source["accountId"];
	        this.region = source["region"];
	        this.baseUrl = source["baseUrl"];
	        this.hasTokenValue = source["hasTokenValue"];
	        this.maskedTokenValue = source["maskedTokenValue"];
	        this.remaining = source["remaining"];
	        this.usage = this.convertValues(source["usage"], usageResponse);
	        this.stats = this.convertValues(source["stats"], tokenStatsResponse);
	        this.health = this.convertValues(source["health"], healthResponse);
	        this.healthScore = source["healthScore"];
	        this.healthLevel = source["healthLevel"];
	        this.healthMessage = source["healthMessage"];
	        this.status = source["status"];
	        this.disabled = source["disabled"];
	        this.selected = source["selected"];
	        this.lastUsedAt = source["lastUsedAt"];
	        this.lastError = source["lastError"];
	        this.cooldownUntil = source["cooldownUntil"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
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
	export class validationResponse {
	    ok: boolean;
	    status: number;
	    durationMs: number;
	    remaining?: number;
	    usage?: usageResponse;
	    message: string;
	    checkedPath: string;

	    static createFrom(source: any = {}) {
	        return new validationResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.status = source["status"];
	        this.durationMs = source["durationMs"];
	        this.remaining = source["remaining"];
	        this.usage = this.convertValues(source["usage"], usageResponse);
	        this.message = source["message"];
	        this.checkedPath = source["checkedPath"];
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
}
