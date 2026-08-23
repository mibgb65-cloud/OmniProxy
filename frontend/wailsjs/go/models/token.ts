export namespace token {

	export class CodexWeeklyEstimateBaseline {
	    inputTokens: number;
	    outputTokens: number;
	    totalTokens: number;
	    cacheCreationTokens?: number;
	    cacheReadTokens?: number;

	    static createFrom(source: any = {}) {
	        return new CodexWeeklyEstimateBaseline(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inputTokens = source["inputTokens"];
	        this.outputTokens = source["outputTokens"];
	        this.totalTokens = source["totalTokens"];
	        this.cacheCreationTokens = source["cacheCreationTokens"];
	        this.cacheReadTokens = source["cacheReadTokens"];
	    }
	}
	export class DailyTokenUsage {
	    date: string;
	    requestCount: number;
	    inputTokens: number;
	    outputTokens: number;
	    totalTokens: number;
	    cacheCreationTokens?: number;
	    cacheReadTokens?: number;

	    static createFrom(source: any = {}) {
	        return new DailyTokenUsage(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.requestCount = source["requestCount"];
	        this.inputTokens = source["inputTokens"];
	        this.outputTokens = source["outputTokens"];
	        this.totalTokens = source["totalTokens"];
	        this.cacheCreationTokens = source["cacheCreationTokens"];
	        this.cacheReadTokens = source["cacheReadTokens"];
	    }
	}
	export class UpsertRequest {
	    name: string;
	    provider: string;
	    credentialType: string;
	    region?: string;
	    baseUrl?: string;
	    tokenValue: string;

	    static createFrom(source: any = {}) {
	        return new UpsertRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.provider = source["provider"];
	        this.credentialType = source["credentialType"];
	        this.region = source["region"];
	        this.baseUrl = source["baseUrl"];
	        this.tokenValue = source["tokenValue"];
	    }
	}

}
