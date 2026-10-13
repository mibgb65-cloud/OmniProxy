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
} as const

export function isKnownProviderKind(kind: string): kind is keyof typeof providerNameKeys {
  return Object.hasOwn(providerNameKeys, kind)
}
