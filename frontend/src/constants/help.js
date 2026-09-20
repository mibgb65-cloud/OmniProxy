export const helpCredentialGroups = [
  {
    title: '订阅与 OAuth 账号',
    summary: 'Codex auth.json、Claude OAuth JSON、MiMo Token Plan、GLM Coding Plan',
    detail: '适合需要订阅额度窗口、自动刷新额度或客户端专用鉴权的场景。',
  },
  {
    title: '按量 API Key',
    summary: 'OpenAI、Anthropic、DeepSeek、Kimi、MiMo、Gemini、OpenRouter、Featherless AI、TokenRouter、Zo Computer、Prem',
    detail: '适合 OpenAI / Anthropic 兼容接口转发，额度页会展示余额、剩余额度或最近统计。',
  },
  {
    title: '网关类账号',
    summary: 'sub2api、new-api、Prem confidential-proxy、自定义网关',
    detail: '适合把已有兼容网关纳入 OmniProxy 调度，并统一暴露本机 loopback 入口。',
  },
]

export const helpWorkflowSteps = [
  {
    step: '01',
    title: '准备账号池',
    description: '在账号管理中按厂商添加凭据。Codex auth.json 会自动解析账号名；API Key 可以批量导入，适合密集账号池。',
    actions: [
      { label: '账号管理', tab: 'tokens' },
      { label: '一键配置', tab: 'quickstart' },
    ],
  },
  {
    step: '02',
    title: '确认路由和策略',
    description: '在全局设置确认本地端口、上游 Base URL、调度模式、低额度跳过阈值和自动重试次数。',
    actions: [{ label: '全局设置', tab: 'settings' }],
  },
  {
    step: '03',
    title: '启动本地代理',
    description: '客户端只连 127.0.0.1，真实上游 Token 留在本机。代理会根据账号状态、选择范围和并发占用自动挑选账号。',
    actions: [{ label: '回到仪表盘', tab: 'dashboard' }],
  },
  {
    step: '04',
    title: '观察额度与请求',
    description: '额度页看账号是否低额度、耗尽或无效；请求历史和实时日志用于定位失败原因、模型、Token 消耗和重试路径。',
    actions: [
      { label: '额度', tab: 'quotas' },
      { label: '请求历史', tab: 'history' },
      { label: '实时日志', tab: 'logs' },
    ],
  },
]

export const helpTroubleshootingItems = [
  {
    problem: '客户端没有请求进入 OmniProxy',
    action: '先确认本地代理已启动，Base URL 使用 127.0.0.1 对应端口；如果使用一键配置，重新写入并检查客户端配置文件路径。',
  },
  {
    problem: '账号返回 401、鉴权失败或显示无效',
    action: '在账号管理中验证该账号。订阅类账号优先刷新认证，API Key 类账号检查上游 Base URL 和 Key 类型是否匹配。',
  },
  {
    problem: '频繁 429 或额度过低',
    action: '到额度页查看每个账号窗口和余额。需要临时隔离时，只选择可用账号；需要自动避让时调高低额度跳过阈值。',
  },
  {
    problem: 'Claude Code 模型不符合预期',
    action: '在一键配置中重新选择最多 4 个 Claude 模型槽位，并注意 DeepSeek、MiMo、Kimi、GLM 的模型名差异。',
  },
  {
    problem: '响应慢或并发被占用',
    action: '看仪表盘实时连接和请求历史。优先平衡使用会避开并发占用更高的账号，队列模式更适合固定优先级。',
  },
]
