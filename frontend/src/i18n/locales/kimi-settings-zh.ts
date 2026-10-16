export const kimiSettingsZh = {
  "kimi.settings.title": "Kimi 账号设置",
  "kimi.settings.close": "关闭账号设置",
  "kimi.settings.order": "默认使用顺序",
  "kimi.settings.planFirst": "Kimi Code 订阅 → API Key（按量）",
  "kimi.settings.apiFirst": "API Key（按量） → Kimi Code 订阅",
  "kimi.settings.orderDescription":
    "同款模型只显示一次。先选择计费方式，再按地区顺序选择已启用且支持模型的账号；没有合格首选账号时使用下一通道。",
  "kimi.settings.orderScope":
    "修改后自动保存，固定账号和显式通道优先。请求失败不跨通道重试；订阅可能使用官方已开启的加油包，不改动官方加油包设置。",
  "kimi.settings.region": "默认地区顺序",
  "kimi.settings.cnFirst": "国内 → 海外",
  "kimi.settings.globalFirst": "海外 → 国内",
  "kimi.settings.regionDescription":
    "在所选计费方式中优先使用此地区；没有合格账号时选择另一地区。同地区按账号创建顺序选择。",
  "kimi.settings.loading": "正在读取默认顺序…",
  "kimi.settings.loadFailed": "无法读取默认使用顺序，请重试。",
  "kimi.settings.retry": "重新读取",
  "kimi.settings.saveFailed": "保存失败，原有顺序已保留，请重试。",
  "kimi.settings.saving": "正在保存…",
  "kimi.settings.saved": "默认使用顺序已保存并生效。",
  "kimi.billing": "计费方式",
  "kimi.api": "普通按量 API",
  "kimi.coding": "Kimi Code 订阅",
  "kimi.codeQuota": "Kimi Code 订阅额度",
  "kimi.codeUnknown":
    "官方订阅剩余额度尚未查询，请到控制台核对。普通按量余额不代表订阅剩余；模型及上下文权限以会员套餐为准。",
  "kimi.openCodeConsole": "查看订阅控制台",
  "kimi.openCodeFailed": "无法打开订阅控制台，请稍后重试。",
} as const
