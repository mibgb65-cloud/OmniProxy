<script setup>
import { computed, ref, watch } from 'vue'
import { CircleCheckFilled, Connection, Delete, Download, Edit, Key, Refresh, Upload, UserFilled } from '@element-plus/icons-vue'

const props = defineProps({
  providers: {
    type: Array,
    required: true,
  },
  activeProvider: {
    type: String,
    required: true,
  },
  activeProviderInfo: {
    type: Object,
    required: true,
  },
  activeProviderTokens: {
    type: Array,
    required: true,
  },
  apiBalanceSummaries: {
    type: Array,
    default: () => [],
  },
  exportingTokens: {
    type: Boolean,
    required: true,
  },
  exportingCodexAuth: {
    type: Boolean,
    required: true,
  },
  codexAuthImporting: {
    type: Boolean,
    required: true,
  },
  codexLoggingIn: {
    type: Boolean,
    default: false,
  },
  batchImporting: {
    type: Boolean,
    required: true,
  },
  openRouterModels: {
    type: Array,
    default: () => [],
  },
  openRouterModelsLoading: {
    type: Boolean,
    default: false,
  },
  openRouterModelsError: {
    type: String,
    default: '',
  },
  openRouterModelsFetchedAt: {
    type: String,
    default: '',
  },
  openRouterModelsCached: {
    type: Boolean,
    default: false,
  },
  validatingIds: {
    type: Object,
    required: true,
  },
  refreshingTokenIds: {
    type: Object,
    required: true,
  },
  togglingTokenIds: {
    type: Object,
    required: true,
  },
  providerTokens: {
    type: Function,
    required: true,
  },
  credentialLabel: {
    type: Function,
    required: true,
  },
  credentialDisplay: {
    type: Function,
    required: true,
  },
  displayStatusType: {
    type: Function,
    required: true,
  },
  displayStatusLabel: {
    type: Function,
    required: true,
  },
  healthSummary: {
    type: Function,
    required: true,
  },
  formatTime: {
    type: Function,
    required: true,
  },
  formatNumber: {
    type: Function,
    required: true,
  },
  formatBalance: {
    type: Function,
    required: true,
  },
  quotaDisplay: {
    type: Function,
    required: true,
  },
})

const emit = defineEmits([
  'select-provider',
  'export-token-backup',
  'login-codex',
  'open-codex-auth-file-picker',
  'import-codex-auth-files',
  'export-codex-auth-backups',
  'refresh-open-router-models',
  'open-router-model-chat',
  'open-create-form',
  'open-batch-import',
  'verify-token',
  'refresh-token-auth',
  'toggle-token-enabled',
  'open-edit-form',
  'remove-token',
])

const codexAuthInput = ref(null)
const openRouterModelPage = ref(1)
const openRouterModelPageSize = 12

const openRouterModelPageCount = computed(() =>
  Math.max(1, Math.ceil(props.openRouterModels.length / openRouterModelPageSize)),
)
const openRouterModelStart = computed(() => (openRouterModelPage.value - 1) * openRouterModelPageSize)
const openRouterModelEnd = computed(() =>
  Math.min(props.openRouterModels.length, openRouterModelStart.value + openRouterModelPageSize),
)
const pagedOpenRouterModels = computed(() =>
  props.openRouterModels.slice(openRouterModelStart.value, openRouterModelEnd.value),
)

watch(
  () => props.openRouterModels.length,
  () => {
    if (openRouterModelPage.value > openRouterModelPageCount.value) {
      openRouterModelPage.value = openRouterModelPageCount.value
    }
  },
)

watch(
  () => props.activeProvider,
  () => {
    openRouterModelPage.value = 1
  },
)

function openCodexAuthFilePicker() {
  emit('open-codex-auth-file-picker')
  codexAuthInput.value?.click()
}

function changeOpenRouterModelPage(delta) {
  const nextPage = openRouterModelPage.value + delta
  openRouterModelPage.value = Math.min(openRouterModelPageCount.value, Math.max(1, nextPage))
}

function canRefreshAuthToken(item) {
  return item?.provider === 'openai' && item?.credentialType === 'codex_auth_json'
}

function apiBalanceSummaryMeta(summary) {
  const parts = [`${props.formatNumber(summary.count)} 个 API Key`]
  if (Number(summary.total || 0) > 0) {
    parts.push(`总额 ${props.formatBalance(summary.total)} ${summary.unit}`)
  }
  if (Number(summary.used || 0) > 0) {
    parts.push(`已用 ${props.formatBalance(summary.used)} ${summary.unit}`)
  }
  return parts.join(' · ')
}
</script>

<template>
  <section class="panel tokens-page-panel">
    <div class="provider-switch" aria-label="厂商选择">
      <button
        v-for="provider in providers"
        :key="provider.key"
        type="button"
        :class="{ active: activeProvider === provider.key }"
        @click="$emit('select-provider', provider.key)"
      >
        {{ provider.label }}
        <span>{{ providerTokens(provider.key).length }}</span>
      </button>
    </div>

    <div class="provider-summary">
      <div>
        <h3>{{ activeProviderInfo.label }}</h3>
        <p>{{ activeProviderInfo.note }} · {{ activeProviderTokens.length }} 个账号</p>
      </div>
      <div
        v-if="apiBalanceSummaries.length"
        class="provider-api-balance-summary"
        aria-label="API Key 总额度"
      >
        <article v-for="summary in apiBalanceSummaries" :key="summary.unit">
          <span>API Key 总额度 · {{ summary.unit }}</span>
          <strong>{{ formatBalance(summary.remaining) }} {{ summary.unit }}</strong>
          <small>{{ apiBalanceSummaryMeta(summary) }}</small>
        </article>
      </div>
      <div class="provider-summary-actions">
        <el-button
          v-if="activeProvider === 'openai'"
          type="primary"
          plain
          :icon="UserFilled"
          :loading="codexLoggingIn"
          @click="$emit('login-codex')"
        >
          {{ codexLoggingIn ? '处理中' : '登录 Codex' }}
        </el-button>
        <el-button :icon="Download" :loading="exportingTokens" @click="$emit('export-token-backup')">
          {{ exportingTokens ? '导出中' : '导出账号池' }}
        </el-button>
        <input
          ref="codexAuthInput"
          class="hidden-file-input"
          type="file"
          accept=".json,application/json"
          multiple
          @change="$emit('import-codex-auth-files', $event)"
        />
        <el-button
          v-if="activeProvider === 'openai'"
          :icon="Upload"
          :loading="codexAuthImporting"
          @click="openCodexAuthFilePicker"
        >
          {{ codexAuthImporting ? '导入中' : '导入 auth 文件' }}
        </el-button>
        <el-button
          v-if="activeProvider === 'openai'"
          :icon="Download"
          :loading="exportingCodexAuth"
          @click="$emit('export-codex-auth-backups')"
        >
          {{ exportingCodexAuth ? '导出中' : '导出 auth 文件' }}
        </el-button>
        <el-button
          v-if="activeProvider === 'openrouter'"
          :icon="Refresh"
          :loading="openRouterModelsLoading"
          @click="$emit('refresh-open-router-models')"
        >
          {{ openRouterModelsLoading ? '刷新中' : '刷新模型' }}
        </el-button>
        <el-button :icon="Upload" :loading="batchImporting" @click="$emit('open-batch-import', activeProvider)">
          {{ batchImporting ? '导入中' : '批量导入 Key' }}
        </el-button>
        <el-button type="primary" :icon="Connection" @click="$emit('open-create-form', activeProvider)">
          添加 {{ activeProviderInfo.label }}
        </el-button>
      </div>
    </div>

    <div v-if="activeProvider === 'openrouter'" class="openrouter-model-panel">
      <div class="openrouter-model-head">
        <div>
          <strong>OpenRouter 模型</strong>
          <small>
            {{ openRouterModels.length ? `${openRouterModels.length} 个模型` : '添加 API Key 后可刷新模型列表' }}
            <template v-if="openRouterModelsFetchedAt">
              · {{ openRouterModelsCached ? '缓存' : '刚刚刷新' }} {{ formatTime(openRouterModelsFetchedAt) }}
            </template>
          </small>
        </div>
      </div>
      <div v-if="openRouterModelsError" class="inline-error">{{ openRouterModelsError }}</div>
      <div v-else-if="openRouterModelsLoading && !openRouterModels.length" class="openrouter-model-skeleton-list">
        <div v-for="index in 6" :key="index" class="openrouter-model-skeleton-row">
          <span></span>
          <small></small>
        </div>
      </div>
      <div v-else-if="openRouterModels.length" class="openrouter-model-list">
        <button
          v-for="model in pagedOpenRouterModels"
          :key="model.id"
          type="button"
          class="openrouter-model-row"
          @click="$emit('open-router-model-chat', model)"
        >
          <div>
            <strong>{{ model.id }}</strong>
            <small>{{ model.name || model.id }}</small>
          </div>
          <span v-if="model.contextLength">{{ formatNumber(model.contextLength) }} ctx</span>
        </button>
      </div>
      <div v-if="openRouterModels.length" class="openrouter-model-pagination">
        <span>
          {{ formatNumber(openRouterModelStart + 1) }}-{{ formatNumber(openRouterModelEnd) }}
          / {{ formatNumber(openRouterModels.length) }}
        </span>
        <div>
          <button
            type="button"
            class="ghost-button compact-button"
            :disabled="openRouterModelPage <= 1"
            @click="changeOpenRouterModelPage(-1)"
          >
            上一页
          </button>
          <strong>第 {{ openRouterModelPage }} / {{ openRouterModelPageCount }} 页</strong>
          <button
            type="button"
            class="ghost-button compact-button"
            :disabled="openRouterModelPage >= openRouterModelPageCount"
            @click="changeOpenRouterModelPage(1)"
          >
            下一页
          </button>
        </div>
      </div>
    </div>

    <div class="account-list-wrap">
      <div class="account-list" role="list" aria-label="账号列表">
        <article v-for="item in activeProviderTokens" :key="item.id" class="account-list-item" role="listitem">
          <div class="account-list-main">
            <div class="account-list-icon" aria-hidden="true">
              <Key />
            </div>
            <div class="account-list-text">
              <div class="account-list-title">
                <strong>{{ item.name }}</strong>
                <el-tag :type="displayStatusType(item)" effect="light" class="status-tag">
                  {{ displayStatusLabel(item) }}
                </el-tag>
              </div>
              <div class="account-list-subline">
                <span class="account-credential-pill mono" :title="credentialDisplay(item)">{{ credentialDisplay(item) }}</span>
                <span v-if="item.lastError" class="account-subtext">{{ item.lastError }}</span>
                <span v-else class="account-subtext">{{ credentialLabel(item) }} · {{ healthSummary(item) }}</span>
              </div>
            </div>
          </div>

          <div class="account-list-meta" aria-label="账号状态">
            <div>
              <span>额度</span>
              <strong>{{ quotaDisplay(item) }}</strong>
            </div>
            <div>
              <span>用量</span>
              <strong>{{ formatNumber(item.stats?.totalTokens) }}</strong>
              <small>{{ formatNumber(item.stats?.requestCount) }} 次请求</small>
            </div>
            <div>
              <span>最后使用</span>
              <strong>{{ formatTime(item.lastUsedAt) }}</strong>
            </div>
          </div>

          <div class="account-list-actions">
            <button
              type="button"
              class="account-toggle"
              :class="{ active: !item.disabled }"
              :aria-pressed="!item.disabled"
              :disabled="togglingTokenIds[item.id]"
              @click="$emit('toggle-token-enabled', item, item.disabled)"
            >
              <span></span>
            </button>
            <div class="row-actions">
              <el-button
                v-if="canRefreshAuthToken(item)"
                size="small"
                class="account-action-button"
                plain
                :icon="Refresh"
                :loading="refreshingTokenIds[item.id]"
                @click="$emit('refresh-token-auth', item)"
              >
                刷新
              </el-button>
              <el-button
                size="small"
                class="account-action-button"
                plain
                :icon="CircleCheckFilled"
                :loading="validatingIds[item.id]"
                @click="$emit('verify-token', item)"
              >
                验证
              </el-button>
              <el-button size="small" class="account-action-button" plain :icon="Edit" @click="$emit('open-edit-form', item)">编辑</el-button>
              <el-button
                size="small"
                class="account-action-button"
                type="danger"
                plain
                :icon="Delete"
                @click="$emit('remove-token', item)"
              >
                删除
              </el-button>
            </div>
          </div>
        </article>
      </div>
      <div v-if="!activeProviderTokens.length" class="empty">
        暂无 {{ activeProviderInfo.label }} 账号
      </div>
    </div>
  </section>
</template>

<style src="./TokensView.css"></style>
