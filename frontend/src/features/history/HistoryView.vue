<script setup>
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { Download, Refresh } from '@element-plus/icons-vue'
import GeminiSelect from '../../components/GeminiSelect.vue'
import HistoryModelInsightPanel from './HistoryModelInsightPanel.vue'
import HistoryRankPanel from './HistoryRankPanel.vue'
import HistoryTable from './HistoryTable.vue'
import HistoryTrendPanel from './HistoryTrendPanel.vue'
import {
  aggregateHistoryByDay,
  buildHistoryDailyWindow,
  buildModelPieSegments,
  clientLabel,
  failureReasonLabel,
  filterHistory,
  historyClientOptions,
  historyWorkspaceLabel,
  historyWorkspaceOptions,
  isFailedHistory,
  normalizeClientRankLabel,
  normalizeDailyRows,
  normalizeFailureReasonLabel,
  normalizeHistorySummary,
  normalizeRanks,
  rankHistory,
  summarizeHistory,
} from '../../utils/historyAnalytics'

const props = defineProps({
  entries: {
    type: Array,
    default: () => [],
  },
  summary: {
    type: Object,
    default: null,
  },
  providers: {
    type: Array,
    default: () => [],
  },
  exporting: {
    type: String,
    default: '',
  },
  formatTime: {
    type: Function,
    required: true,
  },
  formatDuration: {
    type: Function,
    required: true,
  },
  providerLabel: {
    type: Function,
    required: true,
  },
  formatNumber: {
    type: Function,
    required: true,
  },
})

const emit = defineEmits(['refresh', 'export', 'diagnose'])

const filters = reactive({
  provider: 'all',
  client: 'all',
  level: 'all',
  status: 'all',
  model: '',
  tokenId: 'all',
  token: '',
  search: '',
})
const historyPage = ref(1)
const historyPageSize = 200
const historyTrendWindowDays = 14
const modelPieColors = ['#2563eb', '#159a5b', '#e0a11a', '#c43b3b', '#7c3aed', '#0891b2']
let filterRefreshTimer = null

const filteredHistory = computed(() => filterHistory(props.entries, filters))
const localHistorySummary = computed(() => summarizeHistory(filteredHistory.value))
const historySummary = computed(() => normalizeHistorySummary(props.summary) || localHistorySummary.value)
const historyDailyRows = computed(() => {
  if (props.summary) return normalizeDailyRows(props.summary.dailyRows)
  return buildHistoryDailyWindow(aggregateHistoryByDay(filteredHistory.value), historyTrendWindowDays)
})
const providerRanks = computed(() => {
  if (props.summary) return normalizeRanks(props.summary.providerRanks, '-').map((item) => ({
    ...item,
    label: item.label ? props.providerLabel(item.label) : '-',
  })).slice(0, 5)
  return rankHistory(filteredHistory.value, (entry) => props.providerLabel(entry.provider) || '-', 'count').slice(0, 5)
})
const clientOptions = computed(() => historyClientOptions(props.entries))
const workspaceOptions = computed(() => historyWorkspaceOptions(props.entries))
const providerFilterOptions = computed(() => [
  { value: 'all', label: '全部厂商' },
  ...props.providers.map((provider) => ({ value: provider.key, label: provider.label })),
])
const clientFilterOptions = computed(() => [
  { value: 'all', label: '全部工具' },
  ...clientOptions.value.map((client) => ({ value: client.key, label: client.label })),
])
const workspaceFilterOptions = computed(() => [
  { value: 'all', label: '全部工作区' },
  ...workspaceOptions.value.map((workspace) => ({ value: workspace.key, label: workspace.label })),
])
const levelFilterOptions = [
  { value: 'all', label: '全部级别' },
  { value: 'info', label: '正常' },
  { value: 'warn', label: '警告' },
  { value: 'error', label: '错误' },
]
const statusFilterOptions = [
  { value: 'all', label: '全部状态' },
  { value: 'success', label: '成功' },
  { value: 'error', label: '失败' },
  { value: '429', label: '429' },
  { value: '500', label: '500' },
  { value: '502', label: '502' },
  { value: '503', label: '503' },
  { value: '504', label: '504' },
]
const clientRanks = computed(() => {
  if (props.summary) return normalizeRanks(props.summary.clientRanks, '未记录工具', normalizeClientRankLabel).slice(0, 5)
  return rankHistory(filteredHistory.value, (entry) => clientLabel(entry), 'count').slice(0, 5)
})
const tokenRanks = computed(() => {
  if (props.summary) return normalizeRanks(props.summary.tokenRanks, '未记录账号').slice(0, 5)
  return rankHistory(filteredHistory.value, (entry) => historyWorkspaceLabel(entry), 'totalTokens').slice(0, 5)
})
const allModelRanks = computed(() => {
  if (props.summary) return normalizeRanks(props.summary.modelRanks, '未记录模型')
  return rankHistory(filteredHistory.value, (entry) => entry.model || entry.protocol || '未记录模型', 'totalTokens')
})
const modelRanks = computed(() =>
  allModelRanks.value.slice(0, 5),
)
const modelPieSegments = computed(() => buildModelPieSegments(allModelRanks.value, modelPieColors))
const tokenFailureRanks = computed(() => {
  if (props.summary) return normalizeRanks(props.summary.tokenFailureRanks, '未记录账号').slice(0, 5)
  return rankHistory(
    filteredHistory.value.filter(isFailedHistory),
    (entry) => entry.tokenName || '未记录账号',
    'count',
  ).slice(0, 5)
})
const failureReasonRanks = computed(() => {
  if (props.summary) return normalizeRanks(props.summary.failureReasonRanks, '无状态码', normalizeFailureReasonLabel).slice(0, 5)
  return rankHistory(
    filteredHistory.value.filter(isFailedHistory),
    (entry) => failureReasonLabel(entry),
    'count',
  ).slice(0, 5)
})
const historyTotalPages = computed(() => Math.max(1, Math.ceil(filteredHistory.value.length / historyPageSize)))
const pagedHistory = computed(() => {
  const page = Math.min(historyPage.value, historyTotalPages.value)
  const start = (page - 1) * historyPageSize
  return filteredHistory.value.slice(start, start + historyPageSize)
})

watch(filters, () => {
  historyPage.value = 1
  if (filterRefreshTimer) {
    window.clearTimeout(filterRefreshTimer)
  }
  filterRefreshTimer = window.setTimeout(() => {
    filterRefreshTimer = null
    emit('refresh', { ...filters })
  }, 250)
}, { deep: true })

function exportRequestHistory(format) {
  emit('export', {
    format,
    filters: { ...filters },
    entries: filteredHistory.value,
  })
}

onBeforeUnmount(() => {
  if (filterRefreshTimer) {
    window.clearTimeout(filterRefreshTimer)
    filterRefreshTimer = null
  }
})

function previousHistoryPage() {
  historyPage.value = Math.max(1, historyPage.value - 1)
}

function nextHistoryPage() {
  historyPage.value = Math.min(historyTotalPages.value, historyPage.value + 1)
}
</script>

<template>
  <section class="panel history-page-panel">
    <div class="section-heading">
      <div>
        <h2>请求记录分析</h2>
        <p>筛选代理请求、账号验证、额度刷新、重试结果、耗时和 Token 用量</p>
      </div>
      <div class="section-actions">
        <el-button :icon="Refresh" @click="emit('refresh', { ...filters })">刷新</el-button>
        <el-button :icon="Download" :loading="exporting === 'csv'" @click="exportRequestHistory('csv')">
          导出 CSV
        </el-button>
        <el-button :icon="Download" :loading="exporting === 'json'" @click="exportRequestHistory('json')">
          导出 JSON
        </el-button>
      </div>
    </div>

    <div class="history-filters">
      <label>
        <span>厂商</span>
        <GeminiSelect v-model="filters.provider" :options="providerFilterOptions" aria-label="筛选厂商" />
      </label>
      <label>
        <span>工具</span>
        <GeminiSelect v-model="filters.client" :options="clientFilterOptions" aria-label="筛选工具" />
      </label>
      <label>
        <span>级别</span>
        <GeminiSelect v-model="filters.level" :options="levelFilterOptions" aria-label="筛选级别" />
      </label>
      <label>
        <span>状态</span>
        <GeminiSelect v-model="filters.status" :options="statusFilterOptions" aria-label="筛选状态" />
      </label>
      <label>
        <span>工作区</span>
        <GeminiSelect v-model="filters.tokenId" :options="workspaceFilterOptions" aria-label="筛选工作区" />
      </label>
      <label>
        <span>模型</span>
        <input v-model="filters.model" type="search" placeholder="模型名称" />
      </label>
      <label>
        <span>账号</span>
        <input v-model="filters.token" type="search" placeholder="账号名称" />
      </label>
      <label class="history-search">
        <span>搜索</span>
        <input v-model="filters.search" type="search" placeholder="模型、账号、路径或状态码" />
      </label>
    </div>

    <div class="history-summary-grid">
      <div>
        <span>请求数</span>
        <strong>{{ formatNumber(historySummary.total) }}</strong>
      </div>
      <div>
        <span>失败率</span>
        <strong>{{ historySummary.failureRate }}%</strong>
        <small>{{ formatNumber(historySummary.failed) }} 次失败</small>
      </div>
      <div>
        <span>Token</span>
        <strong>{{ formatNumber(historySummary.totalTokens) }}</strong>
      </div>
      <div>
        <span>平均耗时</span>
        <strong>{{ formatDuration(historySummary.averageDuration) }}</strong>
      </div>
    </div>

    <div class="history-insights">
      <HistoryTrendPanel
        :rows="historyDailyRows"
        :total-tokens="historySummary.totalTokens"
        :window-days="historyTrendWindowDays"
        :format-number="formatNumber"
      />
      <HistoryModelInsightPanel
        :model-ranks="modelRanks"
        :segments="modelPieSegments"
        :format-number="formatNumber"
      />
      <HistoryRankPanel title="厂商分布" :items="providerRanks" empty-label="暂无厂商数据" :format-number="formatNumber" />
      <HistoryRankPanel title="工具分布" :items="clientRanks" empty-label="暂无工具数据" :format-number="formatNumber" />
      <HistoryRankPanel title="账号 / 工作区" :items="tokenRanks" empty-label="暂无账号数据" value-key="totalTokens" value-suffix="Token" :format-number="formatNumber" />
      <HistoryRankPanel title="失败账号" :items="tokenFailureRanks" empty-label="暂无失败账号" :format-number="formatNumber" />
      <HistoryRankPanel
        title="失败原因"
        :items="failureReasonRanks"
        empty-label="暂无失败原因"
        :format-number="formatNumber"
        wide
      />
    </div>

    <HistoryTable
      :entries="pagedHistory"
      :filtered-count="filteredHistory.length"
      :page="historyPage"
      :total-pages="historyTotalPages"
      :page-size="historyPageSize"
      :format-number="formatNumber"
      :format-duration="formatDuration"
      :provider-label="providerLabel"
      @diagnose="emit('diagnose', $event)"
      @previous-page="previousHistoryPage"
      @next-page="nextHistoryPage"
    />
  </section>
</template>

<style src="./HistoryView.css"></style>
