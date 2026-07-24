<script setup>
defineProps({
  appIconUrl: { type: String, required: true },
  appInfo: { type: Object, required: true },
  proxyStatus: { type: Object, required: true },
  windowMaximised: { type: Boolean, default: false },
  titlebarUpdateVisible: { type: Boolean, default: false },
  titlebarUpdatePrompt: { type: Object, required: true },
  titlebarUpdatePopoverOpen: { type: Boolean, default: false },
})

defineEmits([
  'close-titlebar-update-popover',
  'close-window',
  'confirm-titlebar-update-popover',
  'minimise-window',
  'skip-current-update',
  'snooze-titlebar-update',
  'toggle-titlebar-update-popover',
  'toggle-window-maximise',
])
</script>

<template>
  <header
    class="window-titlebar"
    :class="{ maximised: windowMaximised }"
    aria-label="窗口控制栏"
    @dblclick="$emit('toggle-window-maximise')"
  >
    <div class="window-titlebar-drag">
      <img :src="appIconUrl" alt="" width="18" height="18" />
      <div>
        <strong>OmniProxy</strong>
        <span>{{ appInfo.isDevelopment ? 'Dev' : appInfo.version }} · {{ proxyStatus.running ? '代理运行中' : '代理未启动' }}</span>
      </div>
    </div>
    <div
      v-if="titlebarUpdateVisible"
      class="window-titlebar-actions"
      aria-label="应用状态"
      @pointerdown.stop
    >
      <button
        type="button"
        class="titlebar-update-button"
        :title="titlebarUpdatePrompt.tooltip"
        aria-haspopup="dialog"
        :aria-expanded="titlebarUpdatePopoverOpen"
        @click.stop="$emit('toggle-titlebar-update-popover')"
        @dblclick.stop
      >
        <span class="titlebar-update-mark" aria-hidden="true"></span>
        <span>新版本</span>
      </button>
      <div
        v-if="titlebarUpdatePopoverOpen"
        class="titlebar-update-popover"
        role="dialog"
        aria-label="新版本提示"
        @click.stop
        @dblclick.stop
      >
        <div class="titlebar-update-popover-head">
          <span class="titlebar-update-popover-icon" aria-hidden="true"></span>
          <div>
            <span class="titlebar-update-popover-kicker">{{ titlebarUpdatePrompt.badge }}</span>
            <strong>{{ titlebarUpdatePrompt.title }}</strong>
          </div>
          <button
            type="button"
            class="titlebar-update-popover-close"
            aria-label="关闭更新提示"
            @click="$emit('close-titlebar-update-popover')"
          >
            <span aria-hidden="true"></span>
          </button>
        </div>
        <p>{{ titlebarUpdatePrompt.description }}</p>
        <div class="titlebar-update-popover-meta">
          <div>
            <span>当前版本</span>
            <strong>{{ titlebarUpdatePrompt.currentVersion }}</strong>
          </div>
          <div>
            <span>最新版本</span>
            <strong>{{ titlebarUpdatePrompt.latestVersion }}</strong>
          </div>
        </div>
        <div class="titlebar-update-popover-actions">
          <button type="button" class="ghost-button compact-button" @click="$emit('snooze-titlebar-update')">稍后</button>
          <button type="button" class="ghost-button compact-button" @click="$emit('skip-current-update')">跳过此版本</button>
          <button type="button" class="primary-button compact-button" @click="$emit('confirm-titlebar-update-popover')">
            {{ titlebarUpdatePrompt.primaryText }}
          </button>
        </div>
      </div>
    </div>
    <div class="window-controls" aria-label="窗口操作">
      <button type="button" class="window-control minimise" aria-label="最小化" @click.stop="$emit('minimise-window')">
        <span class="control-mark" aria-hidden="true"></span>
      </button>
      <button
        type="button"
        :class="['window-control', windowMaximised ? 'restore' : 'maximise']"
        :aria-label="windowMaximised ? '还原窗口' : '最大化'"
        @click.stop="$emit('toggle-window-maximise')"
      >
        <span class="control-mark" aria-hidden="true"></span>
      </button>
      <button type="button" class="window-control close" aria-label="关闭窗口" @click.stop="$emit('close-window')">
        <span class="control-mark" aria-hidden="true"></span>
      </button>
    </div>
  </header>
</template>
