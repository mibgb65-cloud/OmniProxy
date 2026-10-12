import type { AccountItem, ProviderConfiguration } from "../../services/omniproxy"

export type PreparationState = "loading" | "error" | "missing" | "configured"

/** 配置端口不代表已经监听；这里只生成供接入准备使用的本机地址。 */
export function getConfiguredProxyAddress(port: number | undefined): string | null {
  return port !== undefined && Number.isInteger(port) && port >= 1 && port <= 65535
    ? `http://127.0.0.1:${port}`
    : null
}

/** 账号和所属提供商必须同时启用；统计只表示配置情况，不代表密钥有效。 */
export function getConfiguredResources(
  providers: ProviderConfiguration[],
  accounts: AccountItem[],
) {
  const enabledProviders = new Set(providers.filter((item) => item.enabled).map((item) => item.id))
  return {
    providerCount: enabledProviders.size,
    accountCount: accounts.filter((item) => item.enabled && enabledProviders.has(item.providerId))
      .length,
  }
}
