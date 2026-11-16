import type { TranslationKey } from "../../i18n/translation"

/** 订阅登录弹窗按品牌配置；自定义 OpenAI 兼容提供商仍沿用 OpenAI 的文案。 */
export type SubscriptionLoginKind = "openai" | "google" | "workbuddy_ai"

export type SubscriptionLoginConfig = {
  kind: SubscriptionLoginKind
  // 品牌图标按 kind 选择，不能用自定义 provider id。
  brandKind: string
  titleKey: TranslationKey
  descriptionKey: TranslationKey
  storageKey: TranslationKey
  addKey: TranslationKey
  /**
   * 平台是否支持设备码授权：Google 与 OpenAI 的官方组件支持，WorkBuddy 只支持浏览器授权。
   * 界面展示仍沿用既有策略——设备码选择只给 OpenAI；Google 的后端 Begin 只接受
   * `mode === "browser"`，WorkBuddy 显示单张浏览器授权卡片。
   */
  deviceCode: boolean
  /** 浏览器授权卡片额外的一句说明；用于解释上游页面的既有行为。 */
  browserHintKey?: TranslationKey
}

const configs: Record<SubscriptionLoginKind, SubscriptionLoginConfig> = {
  openai: {
    kind: "openai",
    brandKind: "openai",
    titleKey: "subscription.title",
    descriptionKey: "subscription.description",
    storageKey: "subscription.secureStorage",
    addKey: "subscription.add",
    deviceCode: true,
  },
  google: {
    kind: "google",
    brandKind: "google",
    titleKey: "google.subscriptionTitle",
    descriptionKey: "google.subscriptionDescription",
    storageKey: "google.subscriptionStorage",
    addKey: "google.subscriptionAdd",
    deviceCode: true,
  },
  workbuddy_ai: {
    kind: "workbuddy_ai",
    brandKind: "workbuddy_ai",
    titleKey: "workbuddy.loginTitle",
    descriptionKey: "workbuddy.loginDescription",
    storageKey: "workbuddy.loginNote",
    addKey: "workbuddy.loginTitle",
    deviceCode: false,
    browserHintKey: "workbuddy.loginReturnHint",
  },
}

export function subscriptionLoginConfig(providerId: string): SubscriptionLoginConfig {
  if (providerId === "builtin_google") return configs.google
  if (providerId === "builtin_workbuddy_ai") return configs.workbuddy_ai
  return configs.openai
}
