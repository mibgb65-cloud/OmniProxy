export const providerNameKeys = {
  openai: "providers.kind.openai",
  anthropic: "providers.kind.anthropic",
  google: "providers.kind.google",
  deepseek: "providers.kind.deepseek",
  zhipu: "providers.kind.zhipu",
  kimi: "providers.kind.kimi",
  minimax: "providers.kind.minimax",
  mimo: "providers.kind.mimo",
  stepfun: "providers.kind.stepfun",
  atria: "providers.kind.atria",
  river: "providers.kind.river",
} as const

export function isKnownProviderKind(kind: string): kind is keyof typeof providerNameKeys {
  return Object.hasOwn(providerNameKeys, kind)
}

// 认证类型由多个提供商共享；品牌名称必须依据 kind 或内置提供商 ID 选择。
export function subscriptionTypeKey(provider: string | undefined) {
  if (provider === "google" || provider === "builtin_google") return "google.subscriptionType"
  if (provider === "openai" || provider === "builtin_openai") return "subscription.type"
  return "providers.subscriptionLogin"
}
