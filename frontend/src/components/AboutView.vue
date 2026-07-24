<script setup>
import { computed } from 'vue'
import {
  Clock,
  Download,
  FolderOpened,
  InfoFilled,
  Link as LinkIcon,
  Memo,
  Monitor,
  RefreshRight,
  Setting,
  SwitchButton,
} from '@element-plus/icons-vue'
import { updateReleasePageUrl } from '../utils/appUpdate.js'

const props = defineProps({
  appInfo: {
    type: Object,
    default: () => ({}),
  },
  config: {
    type: Object,
    required: true,
  },
  dataDirectory: {
    type: Object,
    required: true,
  },
  proxyStatus: {
    type: Object,
    required: true,
  },
  autoStartEnabled: {
    type: Boolean,
    required: true,
  },
  updateChecking: {
    type: Boolean,
    required: true,
  },
  updateInfo: {
    type: Object,
    default: null,
  },
  updateDownloadStatus: {
    type: Object,
    default: () => ({ state: 'idle' }),
  },
  updateCheckedAt: {
    type: String,
    default: '',
  },
  updateDiagnostics: {
    type: Object,
    default: null,
  },
  updateDiagnosticsLoading: {
    type: Boolean,
    default: false,
  },
  exportingDiagnostics: {
    type: Boolean,
    default: false,
  },
  formatTime: {
    type: Function,
    default: null,
  },
})

defineEmits([
  'manual-check-for-updates',
  'download-update',
  'install-update',
  'open-url',
  'refresh-update-diagnostics',
  'copy-update-diagnostics',
  'export-diagnostics',
])

const currentVersion = computed(
  () => props.appInfo?.version || props.updateInfo?.currentVersion || 'dev',
)
const isDevelopmentBuild = computed(() => {
  const normalizedVersion = String(currentVersion.value || '').trim().toLowerCase()
  return Boolean(props.appInfo?.isDevelopment) || normalizedVersion === 'dev' || normalizedVersion === 'development'
})
const isMacOSPlatform = computed(() => String(props.appInfo?.platform || '').toLowerCase().startsWith('darwin/'))
const releasePageUrl = computed(() => updateReleasePageUrl(props.updateInfo))
const updateDownloadState = computed(() => String(props.updateDownloadStatus?.state || 'idle'))
const downloadMatchesCurrentUpdate = computed(() => {
  if (!props.updateInfo?.updateAvailable) return false
  const status = props.updateDownloadStatus || {}
  return (
    (status.version && status.version === props.updateInfo.latestVersion) ||
    (status.downloadUrl && status.downloadUrl === props.updateInfo.downloadUrl)
  )
})
const updateDownloadActive = computed(
  () => downloadMatchesCurrentUpdate.value && updateDownloadState.value === 'downloading',
)
const updateDownloadReady = computed(
  () => downloadMatchesCurrentUpdate.value && updateDownloadState.value === 'downloaded',
)
const updateDownloadInstalling = computed(
  () => downloadMatchesCurrentUpdate.value && updateDownloadState.value === 'installing',
)
const updateDownloadFailed = computed(
  () => downloadMatchesCurrentUpdate.value && updateDownloadState.value === 'failed',
)
const updateDownloadPercent = computed(() =>
  Math.max(0, Math.min(100, Math.round(Number(props.updateDownloadStatus?.percent || 0)))),
)
const releaseActionLabel = computed(() => {
  if (updateDownloadActive.value) return `下载中 ${updateDownloadPercent.value}%`
  if (updateDownloadInstalling.value) return isMacOSPlatform.value ? 'DMG 已打开' : '安装器已启动'
  if (updateDownloadReady.value) return isMacOSPlatform.value ? '打开 DMG' : '重启安装'
  if (updateDownloadFailed.value) return '重新下载'
  if (props.updateInfo?.updateAvailable && (!props.updateInfo?.downloadUrl || !props.updateInfo?.checksumUrl)) return '打开发布页'
  return props.updateInfo?.updateAvailable ? '下载更新' : '打开发布页'
})
const releaseActionIcon = computed(() => {
  if (updateDownloadReady.value || updateDownloadInstalling.value) return SwitchButton
  return props.updateInfo?.updateAvailable && props.updateInfo?.downloadUrl && props.updateInfo?.checksumUrl
    ? Download
    : LinkIcon
})
const updateBadge = computed(() => {
  if (isDevelopmentBuild.value) return { type: 'info', label: '开发版本' }
  if (!props.updateInfo) return { type: 'info', label: '未检查' }
  if (props.updateInfo.updateAvailable && updateDownloadReady.value) return { type: 'success', label: '已准备好' }
  if (props.updateInfo.updateAvailable && updateDownloadActive.value) return { type: 'warning', label: '下载中' }
  if (props.updateInfo.updateAvailable && updateDownloadFailed.value) return { type: 'danger', label: '下载失败' }
  if (props.updateInfo.updateAvailable && props.updateInfo.prerelease) return { type: 'warning', label: 'Beta 可用' }
  if (props.updateInfo.updateAvailable) return { type: 'warning', label: '有新版本' }
  return { type: 'success', label: '已是最新' }
})
const updateTitle = computed(() => {
  if (isDevelopmentBuild.value) return '开发版本跳过远端更新检测'
  if (!props.updateInfo) return '尚未进行手动检查'
  if (props.updateInfo.updateAvailable && updateDownloadReady.value) {
    return `新版本 ${props.updateInfo.latestVersion || ''} 已准备好`.trim()
  }
  if (props.updateInfo.updateAvailable) {
    return `发现${props.updateInfo.prerelease ? ' Beta' : ''}新版本 ${props.updateInfo.latestVersion || ''}`.trim()
  }
  return `当前已是最新版本 ${props.updateInfo.currentVersion || currentVersion.value}`.trim()
})
const updateDescription = computed(() => {
  if (isDevelopmentBuild.value) return '当前构建用于开发验证，不参与正式发布版本比较。'
  if (!props.updateInfo) return '启动后会自动检查一次；也可以在这里立即检查。'
  if (props.updateInfo.updateAvailable) {
    if (updateDownloadReady.value) {
      return isMacOSPlatform.value
        ? `新版本已准备好，请退出当前 OmniProxy 后打开 DMG 完成替换：${props.updateDownloadStatus?.fileName || props.updateInfo.downloadFileName || '-'}`
        : `新版本已准备好，请重启 OmniProxy 以完成更新：${props.updateDownloadStatus?.fileName || props.updateInfo.downloadFileName || '-'}`
    }
    if (updateDownloadInstalling.value) {
      return isMacOSPlatform.value
        ? '更新 DMG 已打开，请退出当前 OmniProxy 后将 OmniProxy 拖入 Applications 完成替换。'
        : '正在启动更新安装器，OmniProxy 将自动退出并在安装完成后重新打开。'
    }
    if (updateDownloadActive.value) {
      return `正在后台下载 ${props.updateDownloadStatus?.fileName || props.updateInfo.downloadFileName || '更新安装包'}`
    }
    if (updateDownloadFailed.value) {
      return props.updateDownloadStatus?.error || '更新安装包下载失败。'
    }
    return isMacOSPlatform.value
      ? `当前版本 ${props.updateInfo.currentVersion || currentVersion.value}，最新版本 ${props.updateInfo.latestVersion || '-'}，将自动下载 DMG。`
      : `当前版本 ${props.updateInfo.currentVersion || currentVersion.value}，最新版本 ${props.updateInfo.latestVersion || '-'}，将自动下载安装包。`
  }
  return '未发现可用更新。'
})
const updateDownloadDetail = computed(() => {
  if (!downloadMatchesCurrentUpdate.value || !['downloading', 'downloaded', 'failed', 'installing'].includes(updateDownloadState.value)) {
    return ''
  }
  const received = formatBytes(props.updateDownloadStatus?.bytesReceived || 0)
  const total = Number(props.updateDownloadStatus?.totalBytes || 0)
  const parts = total > 0 ? [`${received} / ${formatBytes(total)}`] : [received]
  if (props.updateDownloadStatus?.verified) {
    parts.push('SHA256 已校验')
  } else if (props.updateInfo?.checksumUrl) {
    parts.push('等待校验')
  }
  return parts.join(' · ')
})
const releaseNotesBlocks = computed(() => parseReleaseNotes(props.updateInfo?.body))
const updateDiagnosticsRows = computed(() => {
  const diagnostics = props.updateDiagnostics || {}
  const status = diagnostics.status || props.updateDownloadStatus || {}
  return [
    { label: '当前状态', value: status.state || 'idle' },
    { label: '目标版本', value: status.version || props.updateInfo?.latestVersion || '-' },
    { label: '安装包', value: `${diagnostics.installerCount || 0} 个`, detail: `临时文件 ${diagnostics.partialCount || 0} 个` },
    { label: '日志大小', value: formatBytes(diagnostics.logSize || 0), detail: diagnostics.logExists ? '日志可读取' : '暂无日志' },
  ]
})
const updateDiagnosticsLogLines = computed(() => {
  const text = String(props.updateDiagnostics?.logTail || '').trim()
  return text ? text.split(/\r?\n/).slice(-8) : []
})
const updateDiagnosticsText = computed(() => {
  const diagnostics = props.updateDiagnostics || {}
  const status = diagnostics.status || props.updateDownloadStatus || {}
  return [
    `OmniProxy 更新诊断`,
    `版本: ${currentVersion.value}`,
    `平台: ${props.appInfo?.platform || '-'}`,
    `状态: ${status.state || 'idle'}`,
    `目标版本: ${status.version || props.updateInfo?.latestVersion || '-'}`,
    `错误: ${status.error || diagnostics.error || '-'}`,
    `更新目录: ${diagnostics.directory || '-'}`,
    `状态文件: ${diagnostics.statusPath || '-'} (${diagnostics.statusExists ? '存在' : '不存在'})`,
    `日志文件: ${diagnostics.logPath || '-'} (${diagnostics.logExists ? '存在' : '不存在'})`,
    `安装包数量: ${diagnostics.installerCount || 0}`,
    `临时文件数量: ${diagnostics.partialCount || 0}`,
    '',
    '日志尾部:',
    diagnostics.logTail || '-',
  ].join('\n')
})

function parseReleaseNotes(value) {
  const lines = String(value || '').replace(/\r\n/g, '\n').trim().split('\n')
  const blocks = []
  let paragraphLines = []
  let listItems = []

  const flushParagraph = () => {
    if (!paragraphLines.length) return
    blocks.push({
      type: 'paragraph',
      parts: tokenizeReleaseNoteText(paragraphLines.join(' ')),
    })
    paragraphLines = []
  }
  const flushList = () => {
    if (!listItems.length) return
    blocks.push({ type: 'list', items: listItems })
    listItems = []
  }

  lines.forEach((rawLine) => {
    const line = rawLine.trim()
    if (!line) {
      flushParagraph()
      flushList()
      return
    }

    const heading = line.match(/^(#{1,6})\s+(.+)$/)
    if (heading) {
      flushParagraph()
      flushList()
      blocks.push({
        type: 'heading',
        level: Math.min(heading[1].length, 3),
        parts: tokenizeReleaseNoteText(heading[2]),
      })
      return
    }

    const listItem = line.match(/^([-*+]|\d+[.)])\s+(.+)$/)
    if (listItem) {
      flushParagraph()
      listItems.push({ parts: tokenizeReleaseNoteText(listItem[2]) })
      return
    }

    flushList()
    paragraphLines.push(line.replace(/^>\s?/, ''))
  })

  flushParagraph()
  flushList()
  return blocks
}

function tokenizeReleaseNoteText(value) {
  const text = String(value || '')
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
    .replace(/\*\*([^*]+)\*\*/g, '$1')
    .replace(/__([^_]+)__/g, '$1')
    .replace(/\*([^*]+)\*/g, '$1')
    .replace(/[ \t]+/g, ' ')
    .trim()
  if (!text) return []

  const parts = []
  const codePattern = /`([^`]+)`/g
  let lastIndex = 0
  let match = codePattern.exec(text)
  while (match) {
    if (match.index > lastIndex) {
      parts.push({ type: 'text', text: text.slice(lastIndex, match.index) })
    }
    parts.push({ type: 'code', text: match[1] })
    lastIndex = match.index + match[0].length
    match = codePattern.exec(text)
  }
  if (lastIndex < text.length) {
    parts.push({ type: 'text', text: text.slice(lastIndex) })
  }
  return parts
}

const serviceRows = computed(() => [
  {
    label: '代理服务',
    value: props.proxyStatus.running ? '运行中' : '已停止',
    detail: `端口 :${props.proxyStatus.port || props.config.proxyPort}`,
  },
  {
    label: '控制 API',
    value: `:${props.config.controlPort}`,
    detail: '仅监听本机地址',
  },
  {
    label: '开机自启',
    value: props.autoStartEnabled ? '已开启' : '未开启',
    detail: '启动参数 --minimized',
  },
  {
    label: 'WebSocket',
    value: props.config.websocketMode === 'enabled' ? '已启用' : '已停用',
    detail: 'Codex 流式代理',
  },
])
const configRows = computed(() => [
  {
    label: '调度模式',
    value: schedulingModeLabel(props.config.schedulingMode),
  },
  {
    label: '切换阈值',
    value: `${props.config.switchThreshold}%`,
  },
  {
    label: '自动重试',
    value: `${props.config.maxRetries} 次`,
  },
  {
    label: '更新源',
    value: props.appInfo?.updateEndpoint || '-',
    mono: true,
  },
])

function schedulingModeLabel(value) {
  const labels = {
    queue: '队列模式',
    balanced: '优先平衡使用',
  }
  return labels[value] || value || '-'
}

function dataSourceLabel(value) {
  const labels = {
    env: '环境变量',
    bootstrap: '指针文件',
    legacy: '旧版目录',
    unconfigured: '默认未配置',
    unknown: '未知',
  }
  return labels[value] || value || '-'
}

function formatDate(value) {
  if (!value) return '-'
  if (props.formatTime) {
    return props.formatTime(value)
  }
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '-' : date.toLocaleString()
}

function formatBytes(value) {
  const bytes = Number(value || 0)
  if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${bytes} B`
}
</script>

<template>
  <section class="about-page">
    <div class="about-page-column">
      <article class="panel about-version-panel">
        <div class="section-heading">
          <div>
            <h2>应用版本</h2>
            <p>当前客户端、运行环境和更新状态</p>
          </div>
          <el-tag :type="updateBadge.type" effect="light" round>{{ updateBadge.label }}</el-tag>
        </div>

        <div class="version-hero">
          <div>
            <span>当前版本</span>
            <strong>{{ currentVersion }}</strong>
            <small>{{ appInfo.isDevelopment ? '开发构建' : '发布构建' }}</small>
          </div>
          <div class="about-actions">
            <el-button
              class="about-check-update-button"
              type="primary"
              :icon="RefreshRight"
              :disabled="isDevelopmentBuild"
              :loading="updateChecking"
              @click="$emit('manual-check-for-updates')"
            >
              {{ isDevelopmentBuild ? '开发版跳过更新' : updateChecking ? '检查中' : '检查更新' }}
            </el-button>
            <el-button
              v-if="releasePageUrl"
              :icon="releaseActionIcon"
              :loading="updateDownloadActive"
              :disabled="updateDownloadActive || updateDownloadInstalling"
              @click="updateDownloadReady ? $emit('install-update') : updateInfo?.updateAvailable && updateInfo?.downloadUrl && updateInfo?.checksumUrl ? $emit('download-update') : $emit('open-url', releasePageUrl)"
            >
              {{ releaseActionLabel }}
            </el-button>
            <el-button
              v-if="updateInfo?.updateAvailable && releasePageUrl"
              :icon="LinkIcon"
              @click="$emit('open-url', releasePageUrl)"
            >
              发布页
            </el-button>
          </div>
        </div>

        <div
          :class="[
            'update-status-box',
            updateInfo?.updateAvailable ? 'warning' : '',
            {
              active: updateDownloadActive,
              ready: updateDownloadReady,
              failed: updateDownloadFailed,
              installing: updateDownloadInstalling,
            },
          ]"
        >
          <InfoFilled class="about-status-icon" aria-hidden="true" />
          <div>
            <strong>{{ updateTitle }}</strong>
            <p>{{ updateDescription }}</p>
            <small v-if="updateCheckedAt">上次检查 {{ formatDate(updateCheckedAt) }}</small>
            <div v-if="updateDownloadActive || updateDownloadReady || updateDownloadFailed || updateDownloadInstalling" class="update-download-progress">
              <el-progress
                :percentage="updateDownloadPercent"
                :status="updateDownloadFailed ? 'exception' : updateDownloadReady || updateDownloadInstalling ? 'success' : undefined"
              />
              <small v-if="updateDownloadDetail">{{ updateDownloadDetail }}</small>
            </div>
          </div>
        </div>
      </article>

      <article class="panel about-info-panel">
        <div class="section-heading">
          <div>
            <h2>数据与配置</h2>
            <p>当前数据目录、启动来源和关键策略</p>
          </div>
        </div>
        <div class="about-path-row">
          <FolderOpened class="about-inline-icon" aria-hidden="true" />
          <span>数据目录</span>
          <strong class="mono">{{ dataDirectory.dataDir || '未加载' }}</strong>
          <small v-if="dataDirectory.pendingDataDir && dataDirectory.restartRequired">
            重启后使用：{{ dataDirectory.pendingDataDir }}
          </small>
          <small v-else>来源：{{ dataSourceLabel(dataDirectory.source) }}</small>
        </div>
        <div class="about-row-list compact">
          <div v-for="row in configRows" :key="row.label">
            <span>{{ row.label }}</span>
            <strong :class="{ mono: row.mono }">{{ row.value }}</strong>
          </div>
        </div>
        <div v-if="appInfo.updateEndpoint" class="about-footer-link">
          <el-button link type="primary" :icon="LinkIcon" @click="$emit('open-url', appInfo.updateEndpoint)">
            打开更新源
          </el-button>
        </div>
      </article>

      <article class="panel about-info-panel update-diagnostics-panel">
        <div class="section-heading">
          <div>
            <h2>更新诊断</h2>
            <p>下载状态、临时文件和最近更新日志</p>
          </div>
          <div class="update-diagnostics-actions">
            <el-button :icon="RefreshRight" :loading="updateDiagnosticsLoading" @click="$emit('refresh-update-diagnostics')">
              刷新
            </el-button>
            <el-button :icon="Memo" :disabled="!updateDiagnostics" @click="$emit('copy-update-diagnostics', updateDiagnosticsText)">
              复制
            </el-button>
            <el-button :icon="Download" :loading="exportingDiagnostics" @click="$emit('export-diagnostics')">
              {{ exportingDiagnostics ? '导出中' : '导出诊断包' }}
            </el-button>
          </div>
        </div>
        <div class="about-row-list compact update-diagnostics-grid">
          <div v-for="row in updateDiagnosticsRows" :key="row.label">
            <span>{{ row.label }}</span>
            <strong>{{ row.value }}</strong>
            <small v-if="row.detail">{{ row.detail }}</small>
          </div>
        </div>
        <div class="about-path-row">
          <span>更新目录</span>
          <strong class="mono">{{ updateDiagnostics?.directory || '-' }}</strong>
          <small>状态：{{ updateDiagnostics?.statusPath || '-' }}</small>
        </div>
        <div class="update-log-tail">
          <span>最近日志</span>
          <pre v-if="updateDiagnosticsLogLines.length">{{ updateDiagnosticsLogLines.join('\n') }}</pre>
          <small v-else>暂无更新日志</small>
        </div>
      </article>
    </div>

    <div class="about-page-column">
      <article class="panel about-info-panel">
        <div class="section-heading">
          <div>
            <h2>运行信息</h2>
            <p>桌面进程与构建环境</p>
          </div>
        </div>
        <div class="about-info-grid">
          <div>
            <Monitor class="about-info-icon" aria-hidden="true" />
            <span>平台</span>
            <strong>{{ appInfo.platform || '-' }}</strong>
          </div>
          <div>
            <Setting class="about-info-icon" aria-hidden="true" />
            <span>Go 版本</span>
            <strong>{{ appInfo.goVersion || '-' }}</strong>
          </div>
          <div>
            <Clock class="about-info-icon" aria-hidden="true" />
            <span>启动时间</span>
            <strong>{{ formatDate(appInfo.startedAt) }}</strong>
          </div>
        </div>
        <div class="about-path-row">
          <span>程序路径</span>
          <strong class="mono">{{ appInfo.executablePath || '-' }}</strong>
        </div>
      </article>

      <article class="panel about-info-panel">
        <div class="section-heading">
          <div>
            <h2>本机服务</h2>
            <p>代理、控制端口与后台启动状态</p>
          </div>
        </div>
        <div class="about-row-list">
          <div v-for="row in serviceRows" :key="row.label">
            <span>{{ row.label }}</span>
            <strong>{{ row.value }}</strong>
            <small>{{ row.detail }}</small>
          </div>
        </div>
      </article>
    </div>

    <article v-if="releaseNotesBlocks.length" class="panel about-release-panel">
      <div class="release-notes">
        <div class="release-notes-header">
          <div>
            <span class="release-notes-label">版本说明</span>
            <strong>{{ updateInfo?.latestVersion || currentVersion }}</strong>
          </div>
          <small>完整发布说明</small>
        </div>
        <div class="release-notes-body">
          <template v-for="(block, blockIndex) in releaseNotesBlocks" :key="`${block.type}-${blockIndex}`">
            <h3
              v-if="block.type === 'heading'"
              :class="['release-note-heading', `level-${block.level}`]"
            >
              <template v-for="(part, partIndex) in block.parts" :key="partIndex">
                <code v-if="part.type === 'code'" class="release-note-code">{{ part.text }}</code>
                <span v-else>{{ part.text }}</span>
              </template>
            </h3>
            <ul v-else-if="block.type === 'list'" class="release-note-list">
              <li v-for="(item, itemIndex) in block.items" :key="itemIndex">
                <template v-for="(part, partIndex) in item.parts" :key="partIndex">
                  <code v-if="part.type === 'code'" class="release-note-code">{{ part.text }}</code>
                  <span v-else>{{ part.text }}</span>
                </template>
              </li>
            </ul>
            <p v-else class="release-note-paragraph">
              <template v-for="(part, partIndex) in block.parts" :key="partIndex">
                <code v-if="part.type === 'code'" class="release-note-code">{{ part.text }}</code>
                <span v-else>{{ part.text }}</span>
              </template>
            </p>
          </template>
        </div>
      </div>
    </article>
  </section>
</template>

<style src="./AboutView.css"></style>
