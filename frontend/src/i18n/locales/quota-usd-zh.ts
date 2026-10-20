export const quotaUsdZh = {
  "subscription.usd.title": "额度美元估算",
  "subscription.usd.fiveHour": "5h 预计总额",
  "subscription.usd.weekly": "一周预计总额",
  "subscription.usd.remaining": "预计剩余 ≈ {{amount}}",
  "subscription.usd.scope": "本机样本参考，非官方余额",
  "subscription.usd.explanation":
    "沿用旧版算法：当前窗口内保留的本机代理 Token 基础等值成本 ÷ 官方已用比例。至少已用 5% 才估算；按实际模型参考价分别计费，缓存读取不重复计入普通输入。历史缓存明细缺失时给出区间。参考价非实时，不含长上下文、加速或工具附加费；其他客户端消费、历史清理及调价均可能使结果偏低或失真。两个窗口不可相加，也不是可提现余额。",
  "subscription.usd.pending": "待估算",
  "subscription.usd.insufficient": "已用不足 5%，待估算",
  "subscription.usd.noUsage": "当前窗口无本机用量",
  "subscription.usd.missingWindow": "等待窗口与重置时间",
  "subscription.usd.expired": "窗口已重置，待刷新",
  "subscription.usd.unknownPrice": "{{count}} 条请求价格未知",
  "subscription.usd.missingUsage": "用量不完整，待估算",
  "subscription.usd.unavailable": "估算暂不可用",
  "subscription.usd.resetAdjusted": "仅统计提前重置后的用量",
} as const
