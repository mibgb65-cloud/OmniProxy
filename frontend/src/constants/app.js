export const tabs = [
  { key: 'dashboard', label: '仪表盘' },
  { key: 'usage-trends', label: '用量趋势' },
  { key: 'billing', label: '费用账单' },
  { key: 'quotas', label: '额度' },
  { key: 'tokens', label: '账号管理' },
  { key: 'openrouter-chat', label: 'OpenRouter 对话' },
  { key: 'history', label: '请求历史' },
  { key: 'logs', label: '实时日志' },
  { key: 'quickstart', label: '一键配置' },
  { key: 'gateway-routes', label: '网关路由' },
  { key: 'settings', label: '全局设置' },
  { key: 'about', label: '关于应用' },
  { key: 'help', label: '使用说明' },
]

export const providers = [
  { key: 'openai', label: 'OpenAI', note: '支持 API Key 和 Codex auth.json' },
  { key: 'anthropic', label: 'Anthropic', note: 'API Key 或 Claude OAuth JSON' },
  { key: 'deepseek', label: 'DeepSeek', note: 'API Key' },
  { key: 'kimi', label: 'Kimi Code', note: 'API Key，Claude Code 模型 kimi-for-coding' },
  { key: 'xiaomi', label: 'Xiaomi MiMo', note: '按量 API Key 或 Token Plan' },
  { key: 'zhipu', label: 'Zhipu GLM', note: 'API Key 或 Coding Plan，模型 glm-5.1' },
  { key: 'minimax', label: 'MiniMax', note: 'API Key，模型 MiniMax-M2.7' },
  { key: 'gemini', label: 'Gemini', note: 'Google Gemini API Key' },
  { key: 'openrouter', label: 'OpenRouter', note: 'OpenRouter API Key，自动拉取模型列表' },
  { key: 'featherless', label: 'Featherless AI', note: 'API Key，OpenAI 兼容，支持多 Key 轮换与模型目录同步' },
  { key: 'tokenrouter', label: 'TokenRouter', note: 'TokenRouter API Key，OpenAI 兼容路由' },
  { key: 'sub2api', label: 'sub2api', note: 'sub2api OpenAI / Anthropic / Gemini 兼容网关 API Key' },
  { key: 'newapi', label: 'new-api', note: 'new-api OpenAI / Anthropic / Gemini 兼容网关 API Key' },
  { key: 'anyrouter', label: 'AnyRouter', note: 'AnyRouter API Key，Codex / Claude Code 兼容网关' },
  { key: 'zo', label: 'Zo Computer', note: 'Zo Access Token，OpenAI / Anthropic 兼容适配' },
  { key: 'prem', label: 'Prem', note: 'Prem API Key，通过本机 confidential-proxy 双协议入口' },
  { key: 'atria', label: 'Atria', note: 'Atria API Key（atr_ 开头），OpenAI / Anthropic 兼容，模型 Atria-Dawn-Preview' },
  { key: 'custom', label: '自定义网关', note: 'OpenAI / Anthropic 兼容网关 API Key' },
]

export const credentialTypes = {
  api_key: 'API Key',
  codex_auth_json: 'Codex auth.json',
  mimo_token_plan: 'MiMo Token Plan',
  coding_plan: 'Coding Plan',
  claude_oauth_json: 'Claude OAuth JSON',
}

export const statusMeta = {
  active: { label: '正常', className: 'success' },
  low: { label: '低额度', className: 'warning' },
  exhausted: { label: '耗尽', className: 'muted' },
  invalid: { label: '无效', className: 'danger' },
}
