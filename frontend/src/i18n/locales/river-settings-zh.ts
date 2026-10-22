export const riverSettingsZh = {
  "river.usage.notice":
    "仅统计本机留存的代理请求，不代表 River 官方账单或余额。缓存已包含在输入中；“—”表示上游未返回明细，“≥”表示仅统计到部分已知用量。",
  "river.usage.loading": "正在读取本机用量…",
  "river.usage.failed": "无法读取本机用量。",
  "river.usage.retry": "重试",
  "river.usage.incomplete": "包含进行中或用量未完整返回的请求，当前数值仅为已记录用量。",
  "river.settings.title": "River 账号设置",
  "river.settings.close": "关闭 River 账号设置",
  "river.settings.loadFailed": "无法读取 River 设置，请关闭后重试。",
  "river.rotation.queue": "队列使用",
  "river.rotation.balanced": "均衡使用",
  "river.rotation.queueDescription": "按账号顺序使用，当前 Key 不可用时切换到同池的其他可用 Key。",
  "river.rotation.balancedDescription":
    "在同池可用 Key 间轮流分配新请求；均衡请求数，不保证 Token 消耗完全相同。",
  "river.rotation.failureHint":
    "沿用网关故障切换开关。连续两次网络或服务端失败后暂时跳过该 Key；结果不确定的请求不会重放，固定账号不切换。",
} as const
