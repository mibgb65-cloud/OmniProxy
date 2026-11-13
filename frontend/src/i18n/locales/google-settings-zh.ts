export const googleSettingsZh = {
  "google.settings.title": "Google 账号设置",
  "google.settings.close": "关闭 Google 账号设置",
  "google.settings.loadFailed": "无法读取 Google 设置，请关闭后重试。",
  "google.rotation.queueDescription":
    "按账号顺序使用 Google AI 订阅，当前账号额度耗尽或不可用时切换到同池的下一个可用账号。",
  "google.rotation.balancedDescription":
    "在同池可用的 Google 订阅账号间轮流分配新请求；仅均衡请求数，不保证 Token 消耗完全相同。",
  "google.rotation.failureHint":
    "只对 Google AI 订阅账号生效，API Key 账号不参与轮换。沿用网关故障切换开关：连续两次网络或服务端失败后暂时跳过该账号；结果不确定的请求不会重放，固定账号不切换。",
} as const
