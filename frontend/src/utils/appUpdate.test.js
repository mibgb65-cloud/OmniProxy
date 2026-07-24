import test from 'node:test'
import assert from 'node:assert/strict'

import {
  buildTitlebarUpdatePrompt,
  currentUpdateDownloadState,
  isUpdatePromptSuppressed,
  skipUpdateVersion,
  skippedUpdateVersionKey,
  snoozeUpdatePrompt,
  snoozedUpdateUntilKey,
  updateSnoozeMs,
  updateDownloadMatches,
  updateDownloadPayload,
  updateReleasePageUrl,
} from './appUpdate.js'

test('updateDownloadMatches accepts matching version or download URL', () => {
  const update = {
    updateAvailable: true,
    latestVersion: 'v1.2.0',
    downloadUrl: 'https://example.com/OmniProxy.exe',
  }

  assert.equal(updateDownloadMatches(update, { version: 'v1.2.0' }), true)
  assert.equal(updateDownloadMatches(update, { downloadUrl: 'https://example.com/OmniProxy.exe' }), true)
  assert.equal(updateDownloadMatches(update, { version: 'v1.1.9' }), false)
  assert.equal(updateDownloadMatches({ ...update, updateAvailable: false }, { version: 'v1.2.0' }), false)
})

test('currentUpdateDownloadState ignores stale download status', () => {
  assert.equal(
    currentUpdateDownloadState(
      { updateAvailable: true, latestVersion: 'v1.2.0' },
      { version: 'v1.1.9', state: 'downloaded' },
    ),
    'idle',
  )
})

test('updateDownloadPayload maps release info to backend request payload', () => {
  assert.deepEqual(
    updateDownloadPayload({
      latestVersion: 'v1.2.0',
      downloadUrl: 'https://example.com/app.dmg',
      checksumUrl: 'https://example.com/app.dmg.sha256',
      downloadFileName: 'app.dmg',
      downloadSize: '42',
    }),
    {
      version: 'v1.2.0',
      downloadUrl: 'https://example.com/app.dmg',
      checksumUrl: 'https://example.com/app.dmg.sha256',
      fileName: 'app.dmg',
      expectedSize: 42,
    },
  )
})

test('updateReleasePageUrl prefers the release page over the installer asset', () => {
  assert.equal(
    updateReleasePageUrl({
      releaseUrl: 'https://github.com/mibgb65-cloud/OmniProxy/releases/tag/v1.2.7-beta.4',
      downloadUrl: 'https://github.com/mibgb65-cloud/OmniProxy/releases/download/v1.2.7-beta.4/OmniProxy.exe',
    }),
    'https://github.com/mibgb65-cloud/OmniProxy/releases/tag/v1.2.7-beta.4',
  )
  assert.equal(
    updateReleasePageUrl({ downloadUrl: 'https://example.com/installer.exe' }),
    '',
  )
})

test('buildTitlebarUpdatePrompt reflects downloaded macOS update', () => {
  const prompt = buildTitlebarUpdatePrompt({
    update: {
      updateAvailable: true,
      currentVersion: 'v1.1.9',
      latestVersion: 'v1.2.0',
      downloadUrl: 'https://example.com/app.dmg',
      checksumUrl: 'https://example.com/app.dmg.sha256',
      prerelease: true,
    },
    status: {
      state: 'downloaded',
      version: 'v1.2.0',
    },
    isMacOSPlatform: true,
  })

  assert.equal(prompt.canInstall, true)
  assert.equal(prompt.badge, 'Beta 已准备好')
  assert.equal(prompt.primaryText, '打开 DMG')
  assert.match(prompt.description, /拖入 Applications/)
})

test('skipUpdateVersion suppresses only the selected version', () => {
  const storage = createMemoryStorage()
  skipUpdateVersion({ latestVersion: 'v1.2.0' }, storage)

  assert.equal(storage.getItem(skippedUpdateVersionKey), 'v1.2.0')
  assert.equal(isUpdatePromptSuppressed({ latestVersion: 'v1.2.0' }, storage), true)
  assert.equal(isUpdatePromptSuppressed({ latestVersion: 'v1.2.1' }, storage), false)
})

test('snoozeUpdatePrompt suppresses prompts until it expires', () => {
  const storage = createMemoryStorage()
  const now = 1000
  snoozeUpdatePrompt(storage, now)

  assert.equal(storage.getItem(snoozedUpdateUntilKey), String(now + updateSnoozeMs))
  assert.equal(isUpdatePromptSuppressed({ latestVersion: 'v1.2.0' }, storage, now + 1), true)
  assert.equal(isUpdatePromptSuppressed({ latestVersion: 'v1.2.0' }, storage, now + updateSnoozeMs + 1), false)
  assert.equal(storage.getItem(snoozedUpdateUntilKey), null)
})

function createMemoryStorage() {
  const values = new Map()
  return {
    getItem(key) {
      return values.has(key) ? values.get(key) : null
    },
    removeItem(key) {
      values.delete(key)
    },
    setItem(key, value) {
      values.set(key, String(value))
    },
  }
}
