<script setup>
const props = defineProps({
  config: {
    type: Object,
    required: true,
  },
})

const outboundProxyPresets = [
  { label: '10808 mixed', url: 'http://127.0.0.1:10808' },
  { label: '7890 mixed', url: 'http://127.0.0.1:7890' },
  { label: 'SOCKS5 10808', url: 'socks5://127.0.0.1:10808' },
]
const recommendedOutboundProxyProviders = ['openai', 'anthropic', 'gemini', 'openrouter', 'zo', 'prem']
const outboundProxyProviderGroups = [
  {
    title: '国内网络建议出站',
    note: '这些接入厂商通常依赖海外入口，模型列表、额度刷新和对话请求都会走出站代理。',
    items: [
      {
        key: 'openai-codex',
        label: 'OpenAI / Codex',
        providers: ['openai'],
        description: 'OpenAI API、Codex auth.json、chatgpt.com Codex 接口',
        recommended: true,
      },
      {
        key: 'anthropic-claude',
        label: 'Anthropic Claude',
        providers: ['anthropic'],
        description: 'Anthropic API、Claude OAuth 和 Claude 兼容路由',
        recommended: true,
      },
      {
        key: 'google-gemini',
        label: 'Google Gemini',
        providers: ['gemini'],
        description: 'Google Gemini 原生接口和模型列表',
        recommended: true,
      },
      {
        key: 'openrouter',
        label: 'OpenRouter',
        providers: ['openrouter'],
        description: 'OpenRouter 模型列表、测试对话、余额和代理转发',
        recommended: true,
      },
      {
        key: 'zo',
        label: 'Zo Computer',
        providers: ['zo'],
        description: 'Zo Computer 模型映射、模型列表和对话请求',
        recommended: true,
      },
      {
        key: 'prem',
        label: 'Prem',
        providers: ['prem'],
        description: 'Prem confidential-proxy OpenAI / Anthropic 双协议入口',
        recommended: true,
      },
    ],
  },
  {
    title: '国内通常可直连',
    note: '这些是当前内置国内厂商，默认不走出站代理。',
    items: [
      {
        key: 'deepseek',
        label: 'DeepSeek',
        providers: ['deepseek'],
        description: 'DeepSeek API 和 DeepSeek 兼容路由',
      },
      {
        key: 'kimi',
        label: 'Kimi Code',
        providers: ['kimi'],
        description: 'kimi-for-coding',
      },
      {
        key: 'zhipu',
        label: 'Zhipu GLM',
        providers: ['zhipu'],
        description: '智谱 GLM API、Coding Plan 和兼容接口',
      },
      {
        key: 'minimax',
        label: 'MiniMax',
        providers: ['minimax'],
        description: 'MiniMax API 和 Coding Plan',
      },
      {
        key: 'mimo',
        label: 'Xiaomi MiMo',
        providers: ['xiaomi'],
        description: 'Xiaomi MiMo API Key 和 Token Plan',
      },
    ],
  },
  {
    title: '取决于你的上游',
    note: '自定义网关、Sub2API、new-api、TokenRouter 是否需要出站，取决于你配置的实际服务地址。',
    items: [
      {
        key: 'tokenrouter',
        label: 'TokenRouter',
        providers: ['tokenrouter'],
        description: 'TokenRouter 账号、模型和路由规则接口',
      },
      {
        key: 'sub2api',
        label: 'Sub2API',
        providers: ['sub2api'],
        description: 'Sub2API OpenAI / Anthropic / Gemini 兼容接口',
      },
      {
        key: 'newapi',
        label: 'new-api',
        providers: ['newapi'],
        description: 'new-api OpenAI / Anthropic / Gemini 兼容接口',
      },
      {
        key: 'anyrouter',
        label: 'AnyRouter',
        providers: ['anyrouter'],
        description: 'AnyRouter Codex/OpenAI 与 Claude Code/Anthropic 兼容接口',
      },
      {
        key: 'custom',
        label: '自定义网关',
        providers: ['custom'],
        description: '自定义 OpenAI / Anthropic 兼容网关',
      },
    ],
  },
]

function setOutboundProxyUrl(url) {
  props.config.outboundProxyUrl = url
  props.config.outboundProxyEnabled = true
}

function resetOutboundProxyProviders() {
  props.config.outboundProxyProviders = [...recommendedOutboundProxyProviders]
}

function toggleOutboundProxyProvider(item) {
  if (isOutboundProxyProviderSelected(item)) {
    removeOutboundProxyProviders(item.providers)
  } else {
    addOutboundProxyProviders(item.providers)
  }
}

function addOutboundProxyProviders(providers) {
  props.config.outboundProxyProviders = normalizeOutboundProxyProviders([
    ...(Array.isArray(props.config.outboundProxyProviders) ? props.config.outboundProxyProviders : []),
    ...providers,
  ])
}

function removeOutboundProxyProviders(providers) {
  const keys = new Set(providers.map((provider) => String(provider || '').trim().toLowerCase()).filter(Boolean))
  props.config.outboundProxyProviders = (Array.isArray(props.config.outboundProxyProviders)
    ? props.config.outboundProxyProviders
    : []
  ).filter((item) => !keys.has(String(item || '').trim().toLowerCase()))
}

function isOutboundProxyProviderSelected(item) {
  return item.providers.every((provider) => hasOutboundProxyProvider(provider))
}

function hasOutboundProxyProvider(provider) {
  const key = String(provider || '').trim().toLowerCase()
  return selectedOutboundProxyProviders().some((item) => String(item || '').trim().toLowerCase() === key)
}

function selectedOutboundProxyProviders() {
  return Array.isArray(props.config.outboundProxyProviders) ? props.config.outboundProxyProviders : []
}

function selectedOutboundProxyProviderCount() {
  return selectedOutboundProxyProviders().length
}

function customOutboundProxyProviders() {
  const known = new Set(
    outboundProxyProviderGroups.flatMap((group) => group.items).flatMap((item) => item.providers.map((provider) => provider.toLowerCase())),
  )
  return selectedOutboundProxyProviders().filter((provider) => !known.has(String(provider || '').trim().toLowerCase()))
}

function normalizeOutboundProxyProviders(providers) {
  const seen = new Set()
  const next = []
  for (const provider of providers) {
    const value = String(provider || '').trim().toLowerCase()
    const key = value.toLowerCase()
    if (!value || seen.has(key)) continue
    seen.add(key)
    next.push(value)
  }
  return next
}
</script>

<template>
  <section class="settings-section settings-outbound-section">
    <div class="settings-section-head">
      <div>
        <h3>出站代理</h3>
        <p>按接入厂商决定是否走 Clash、v2rayN 等本机代理端口，未选中的厂商继续直连。</p>
      </div>
    </div>
    <div class="settings-grid">
      <label class="toggle-field">
        <span>启用厂商出站代理</span>
        <input v-model="config.outboundProxyEnabled" class="toggle-input" type="checkbox" />
        <span class="toggle-switch" aria-hidden="true">
          <span class="toggle-thumb"></span>
        </span>
      </label>
      <label>
        <span>本机代理地址</span>
        <input v-model="config.outboundProxyUrl" type="text" placeholder="10808 或 http://127.0.0.1:10808" />
      </label>
      <div class="wide-field settings-chip-field">
        <span>常用端口</span>
        <div class="settings-chip-list">
          <button
            v-for="preset in outboundProxyPresets"
            :key="preset.url"
            type="button"
            class="settings-chip-button"
            :class="{ active: config.outboundProxyUrl === preset.url }"
            :aria-pressed="config.outboundProxyUrl === preset.url"
            @click="setOutboundProxyUrl(preset.url)"
          >
            {{ preset.label }}
          </button>
        </div>
        <small>Clash mixed-port 和 v2rayN mixed/http 入站可直接用 HTTP 地址；SOCKS 入站使用 socks5://。</small>
      </div>
      <div class="wide-field outbound-model-selector">
        <div class="outbound-model-selector-head">
          <div>
            <span>走出站代理的接入厂商</span>
            <small>已选择 {{ selectedOutboundProxyProviderCount() }} 个厂商</small>
          </div>
          <button type="button" class="ghost-button compact-button" @click="resetOutboundProxyProviders">
            恢复国内推荐
          </button>
        </div>
        <div v-for="group in outboundProxyProviderGroups" :key="group.title" class="outbound-model-group">
          <div class="outbound-model-group-head">
            <strong>{{ group.title }}</strong>
            <small>{{ group.note }}</small>
          </div>
          <div class="outbound-model-options">
            <button
              v-for="item in group.items"
              :key="item.key"
              type="button"
              class="outbound-model-option"
              :class="{ active: isOutboundProxyProviderSelected(item), recommended: item.recommended }"
              :aria-pressed="isOutboundProxyProviderSelected(item)"
              @click="toggleOutboundProxyProvider(item)"
            >
              <span class="outbound-model-option-title">
                <strong>{{ item.label }}</strong>
                <em>{{ isOutboundProxyProviderSelected(item) ? '走出站' : '直连' }}</em>
              </span>
              <small>{{ item.description }}</small>
              <code>{{ item.providers.join(' / ') }}</code>
            </button>
          </div>
        </div>
        <div v-if="customOutboundProxyProviders().length" class="settings-chip-field">
          <span>未归类厂商</span>
          <div class="settings-chip-list">
            <button
              v-for="provider in customOutboundProxyProviders()"
              :key="provider"
              type="button"
              class="settings-chip-button active"
              aria-pressed="true"
              @click="removeOutboundProxyProviders([provider])"
            >
              {{ provider }}
            </button>
          </div>
          <small>这些厂商来自旧配置；点击可移除。</small>
        </div>
      </div>
    </div>
  </section>
</template>
