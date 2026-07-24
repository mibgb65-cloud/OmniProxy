export const pendingUpdateVersionKey = 'omniproxy.pendingUpdateVersion'
export const skippedUpdateVersionKey = 'omniproxy.skippedUpdateVersion'
export const snoozedUpdateUntilKey = 'omniproxy.snoozedUpdateUntil'
export const updateCheckIntervalMs = 4 * 60 * 60 * 1000
export const updateSnoozeMs = 24 * 60 * 60 * 1000

function defaultUpdateStorage() {
  return typeof window === 'undefined' ? null : window.localStorage
}

function updateVersionKey(update) {
  return String(update?.latestVersion || '').trim()
}

export function updateDownloadMatches(update, status) {
  if (!update?.updateAvailable || !status) {
    return false
  }
  const latestVersion = String(update.latestVersion || '').trim()
  const statusVersion = String(status.version || '').trim()
  const downloadUrl = String(update.downloadUrl || '').trim()
  const statusDownloadUrl = String(status.downloadUrl || '').trim()
  return Boolean(
    (latestVersion && statusVersion && statusVersion === latestVersion) ||
      (downloadUrl && statusDownloadUrl && statusDownloadUrl === downloadUrl),
  )
}

export function currentUpdateDownloadState(update, status) {
  return updateDownloadMatches(update, status) ? String(status?.state || 'idle') : 'idle'
}

export function updateDownloadPayload(update) {
  return {
    version: update?.latestVersion || '',
    downloadUrl: update?.downloadUrl || '',
    checksumUrl: update?.checksumUrl || '',
    fileName: update?.downloadFileName || '',
    expectedSize: Number(update?.downloadSize || 0),
  }
}

export function updateReleasePageUrl(update) {
  return String(update?.releaseUrl || '').trim()
}

export function isUpdatePromptSuppressed(update, storage = defaultUpdateStorage(), now = Date.now()) {
  const latestVersion = updateVersionKey(update)
  if (!latestVersion || !storage) {
    return false
  }
  if (String(storage.getItem(skippedUpdateVersionKey) || '').trim() === latestVersion) {
    return true
  }
  const snoozedUntil = Number(storage.getItem(snoozedUpdateUntilKey) || 0)
  if (Number.isFinite(snoozedUntil) && snoozedUntil > now) {
    return true
  }
  if (snoozedUntil) {
    storage.removeItem(snoozedUpdateUntilKey)
  }
  return false
}

export function snoozeUpdatePrompt(storage = defaultUpdateStorage(), now = Date.now()) {
  if (!storage) {
    return
  }
  storage.setItem(snoozedUpdateUntilKey, String(now + updateSnoozeMs))
}

export function skipUpdateVersion(update, storage = defaultUpdateStorage()) {
  const latestVersion = updateVersionKey(update)
  if (!latestVersion || !storage) {
    return
  }
  storage.setItem(skippedUpdateVersionKey, latestVersion)
}

export function buildTitlebarUpdatePrompt({
  update = {},
  status = { state: 'idle' },
  currentVersion = '',
  isMacOSPlatform = false,
} = {}) {
  const updateAvailable = Boolean(update?.updateAvailable)
  const currentVersionLabel = update?.currentVersion || currentVersion || '当前版本'
  const latestVersion = update?.latestVersion || '新版本'
  const canDownload = Boolean(updateAvailable && update?.downloadUrl && update?.checksumUrl)
  const downloadState = currentUpdateDownloadState(update, status)
  const downloadActive = downloadState === 'downloading'
  const canInstall = downloadState === 'downloaded'
  const canRetryDownload = downloadState === 'failed' && canDownload
  const downloadPercent = Math.max(0, Math.min(100, Math.round(Number(status?.percent || 0))))
  const badge = canInstall
    ? update?.prerelease
      ? 'Beta 已准备好'
      : '更新已准备好'
    : update?.prerelease
      ? 'Beta 更新可用'
      : '更新可用'

  return {
    update: updateAvailable ? update : null,
    canDownload,
    canInstall,
    canRetryDownload,
    currentVersion: currentVersionLabel,
    latestVersion,
    badge,
    title: canInstall ? `新版本 ${latestVersion} 已准备好` : `发现${update?.prerelease ? ' Beta' : ''}新版本 ${latestVersion}`,
    description: canInstall
      ? isMacOSPlatform
        ? '点击打开 DMG 后，请先退出当前 OmniProxy，再将 OmniProxy 拖入 Applications 完成替换。'
        : '点击重启安装将关闭当前 OmniProxy，启动安装器，并在安装完成后重新打开应用。'
      : downloadActive
        ? `正在后台下载更新安装包，当前进度 ${downloadPercent}%。`
        : canRetryDownload
          ? '更新安装包下载失败，可以重新下载或打开关于应用查看详情。'
          : canDownload
            ? isMacOSPlatform
              ? '已发现新版本，OmniProxy 会自动下载 DMG，完成后提示打开。'
              : '已发现新版本，OmniProxy 会自动下载安装包，完成后提示重启。'
            : '暂未获取到可用安装包，可以打开关于应用查看发布页。',
    primaryText: canInstall ? (isMacOSPlatform ? '打开 DMG' : '重启安装') : canRetryDownload ? '重新下载' : downloadActive ? '查看进度' : '查看详情',
    tooltip: `发现新版本 ${latestVersion}`,
  }
}
