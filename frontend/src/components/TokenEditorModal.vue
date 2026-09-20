<script setup>
import { computed } from 'vue'
import GeminiSelect from './GeminiSelect.vue'

const props = defineProps({
  form: {
    type: Object,
    required: true,
  },
  providers: {
    type: Array,
    required: true,
  },
  isCodexForm: {
    type: Boolean,
    required: true,
  },
  placeholder: {
    type: String,
    required: true,
  },
})

defineEmits(['close', 'submit', 'provider-change'])

const providerOptions = computed(() =>
  props.providers.map((provider) => ({ value: provider.key, label: provider.label })),
)

const credentialTypeOptions = computed(() => {
  const options = [
    { value: 'api_key', label: props.form.provider === 'xiaomi' ? 'MiMo 按量 API Key (sk-)' : 'API Key' },
  ]
  if (props.form.provider === 'openai') {
    options.push({ value: 'codex_auth_json', label: 'Codex auth.json' })
  }
  if (props.form.provider === 'anthropic') {
    options.push({ value: 'claude_oauth_json', label: 'Claude OAuth JSON' })
  }
  if (props.form.provider === 'xiaomi') {
    options.push({ value: 'mimo_token_plan', label: 'MiMo Token Plan (tp-)' })
  }
  if (props.form.provider === 'zhipu') {
    options.push({ value: 'coding_plan', label: 'GLM Coding Plan' })
  }
  return options
})

const regionOptions = [
  { value: 'cn', label: '中国区' },
  { value: 'sgp', label: '新加坡 SGP' },
  { value: 'ams', label: '欧洲 AMS' },
]

function credentialTypeLocked(form) {
  return !['openai', 'xiaomi', 'zhipu', 'anthropic'].includes(form.provider)
}

function isJSONCredential(form) {
  return ['codex_auth_json', 'claude_oauth_json'].includes(form.credentialType)
}

function credentialValueLabel(form) {
  if (form.credentialType === 'codex_auth_json') return 'auth.json 内容'
  if (form.credentialType === 'claude_oauth_json') return 'Claude OAuth JSON'
  if (form.credentialType === 'mimo_token_plan') return 'Token Plan Key'
  if (form.credentialType === 'coding_plan') return 'Coding Plan Key'
  return 'API Key'
}

function credentialHint() {
  if (props.form.provider === 'anyrouter') {
    return 'Base URL 会跟随这个账号保存，支持 AnyRouter 的 Codex/OpenAI 和 Claude Code/Anthropic 入口。'
  }
  if (['sub2api', 'newapi'].includes(props.form.provider)) {
    return 'Base URL 会跟随这个账号保存，支持同一上游的 OpenAI、Anthropic、Gemini 协议入口。'
  }
  if (props.form.provider === 'zo') {
    return '保存 Zo Access Token 后，可通过 /zo/v1/chat/completions 或 /zo/v1/messages 使用本地兼容入口。'
  }
  if (props.form.provider === 'prem') {
    return '保存 Prem API Key 后，OmniProxy 会按账号调度并转发到全局 Prem confidential-proxy Base URL。'
  }
  if (props.form.provider === 'openrouter') {
    return '保存后可在账号管理刷新 OpenRouter 模型列表，模型 ID 会按 provider/model 展示。'
  }
  if (props.form.provider === 'featherless') {
    return '可添加多个 API Key；请求会按调度模式轮换，遇到 429 或 503 时自动切换可用 Key。'
  }
  if (props.form.credentialType === 'claude_oauth_json') {
    return '支持包含 access_token / refresh_token / expired 的 Claude Code OAuth JSON。'
  }
  if (props.form.provider === 'zhipu' && props.form.credentialType === 'coding_plan') {
    return 'Coding Plan 会使用 api.z.ai 的订阅额度接口；普通 API Key 会查询 BigModel API 余额。'
  }
  return ''
}

function autoNameText(form) {
  if (form.provider === 'anthropic' && form.credentialType === 'claude_oauth_json') {
    return 'Claude OAuth JSON 会优先使用 email 作为账号名称。'
  }
  return 'Codex 将自动使用 auth.json 中的邮箱作为账号名称。'
}

function requiresBaseUrl(form) {
  return ['sub2api', 'newapi', 'anyrouter'].includes(form.provider)
}

function baseUrlPlaceholder(form) {
  if (form.provider === 'newapi') return 'http://127.0.0.1:3000'
  if (form.provider === 'anyrouter') return 'https://anyrouter.top'
  return 'https://aiapi.aicnio.com'
}

function baseUrlHint(form) {
  if (form.provider === 'newapi') {
    return '保存到当前账号；/newapi、/newapi/anthropic、/newapi/gemini 会转发到这个上游。'
  }
  return '保存到当前账号；/sub2api、/sub2api/anthropic、/sub2api/gemini 会转发到这个上游。'
}
</script>

<template>
  <div class="modal-backdrop token-editor-backdrop" @click.self="$emit('close')">
    <form class="modal token-editor-modal" @submit.prevent="$emit('submit')">
      <div class="section-heading">
        <div>
          <h2>{{ form.editingId ? '编辑账号' : '添加账号' }}</h2>
          <p>{{ isCodexForm ? autoNameText(form) : '账号名称必填且不可重复' }}</p>
        </div>
        <button type="button" class="icon-button" @click="$emit('close')">×</button>
      </div>
      <label v-if="!isCodexForm">
        <span>账号名称</span>
        <input v-model="form.name" autofocus />
      </label>
      <div v-else class="form-hint">
        {{ autoNameText(form) }}
      </div>
      <label>
        <span>厂商</span>
        <GeminiSelect
          v-model="form.provider"
          :options="providerOptions"
          aria-label="选择厂商"
          @change="$emit('provider-change')"
        />
      </label>
      <label>
        <span>凭据类型</span>
        <GeminiSelect
          v-model="form.credentialType"
          :options="credentialTypeOptions"
          :disabled="credentialTypeLocked(form)"
          aria-label="选择凭据类型"
        />
      </label>
      <label v-if="form.provider === 'xiaomi' && form.credentialType === 'mimo_token_plan'">
        <span>Token Plan 区域</span>
        <GeminiSelect v-model="form.region" :options="regionOptions" aria-label="选择 Token Plan 区域" />
        <small>请选择 Token Plan 账号所属服务中心。</small>
      </label>
      <label v-if="requiresBaseUrl(form)">
        <span>Base URL</span>
        <input
          v-model="form.baseUrl"
          type="url"
          :placeholder="baseUrlPlaceholder(form)"
          autocomplete="off"
          spellcheck="false"
        />
        <small>{{ baseUrlHint(form) }}</small>
      </label>
      <label>
        <span>{{ credentialValueLabel(form) }}</span>
        <textarea
          v-model="form.tokenValue"
          :rows="isJSONCredential(form) ? 9 : 4"
          :placeholder="placeholder"
        ></textarea>
        <small v-if="credentialHint()">{{ credentialHint() }}</small>
        <small v-if="form.editingId">不填写则继续使用当前已保存凭据</small>
      </label>
      <div class="modal-actions">
        <button type="button" class="ghost-button" @click="$emit('close')">取消</button>
        <button type="submit" class="primary-button">保存</button>
      </div>
    </form>
  </div>
</template>

<style src="./TokenEditorModal.css"></style>
