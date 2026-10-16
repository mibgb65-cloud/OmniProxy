export const mimoZh = {
  "mimo.settings.title": "小米 MiMo 账号设置",
  "mimo.settings.close": "关闭账号设置",
  "mimo.settings.order": "默认使用顺序",
  "mimo.settings.planFirst": "Token Plan → API Key（按量）",
  "mimo.settings.apiFirst": "API Key（按量） → Token Plan",
  "mimo.settings.orderDescription":
    "同名模型只显示一次。优先使用所选通道；没有已启用且支持该模型的账号时，使用下一通道。",
  "mimo.settings.orderScope":
    "修改后自动保存，适用于所有小米 MiMo 账号。固定账号与显式通道优先；同一通道按账号创建顺序选择，请求失败不会换通道重试。",
  "mimo.settings.loading": "正在读取默认顺序…",
  "mimo.settings.loadFailed": "无法读取默认使用顺序，请重试。",
  "mimo.settings.retry": "重新读取",
  "mimo.settings.saveFailed": "保存失败，原有顺序已保留，请重试。",
  "mimo.settings.saving": "正在保存…",
  "mimo.settings.saved": "默认使用顺序已保存并生效。",
  "mimo.openAccess": "查看 Token Plan 接入信息",
  "mimo.closeAccess": "关闭 Token Plan 接入信息",
  "mimo.accessTitle": "Token Plan 接入",
  "mimo.accessDescription":
    "以下为兼容协议的基础地址。个人版使用 tp- Key，团队版使用 ttp- Key；请在账号中选择控制台指定的集群。",
  "mimo.accessModels": "套餐支持 mimo-v2.6-pro、mimo-v2.6-flash；高速版使用普通按量 API。",
  "mimo.apiEndpoint": "按量入口",
  "mimo.probePayg": "测试按量入口",
  "mimo.probeScope": "卡片测试范围：普通按量默认入口。Token Plan 调用使用账号所选集群。",
  "mimo.connection": "小米 MiMo 连接",
  "mimo.billing": "计费方式",
  "mimo.payg": "普通按量 API",
  "mimo.plan": "Token Plan",
  "mimo.region": "套餐集群",
  "mimo.cn": "中国",
  "mimo.sgp": "新加坡",
  "mimo.ams": "欧洲",
  "mimo.choose": "请选择计费方式和套餐集群。",
  "mimo.needsConfiguration": "需确认 MiMo 通道",
  "mimo.keyHint":
    "普通按量使用普通 Key；Token Plan 使用个人 tp- 或团队 ttp- Key，并选择控制台显示的集群。一批账号共用此选择。",
  "mimo.routingHint":
    "普通按量和 Token Plan 分别计费；同名模型按设置中的默认使用顺序选择账号，也可用固定账号或显式通道。请求失败不会换通道重试。",
  "mimo.quotaTitle": "Token Plan 套餐额度",
  "mimo.quotaUnknown":
    "官方剩余 Credits 暂未查询。本机 Token 用量可在用量页查看；套餐额度和有效期请到官方平台查询。",
  "mimo.openConsole": "查看官方套餐",
  "mimo.openFailed": "未能打开官方平台，请稍后重试。",
  "routing.error.ambiguous_mimo_pool":
    "存在多个小米 MiMo 计费通道或集群，请选择通道模型名或固定账号。",
} as const
