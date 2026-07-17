<script setup>
import { computed } from 'vue'

const props = defineProps({
  dialog: {
    type: Object,
    required: true,
  },
  busy: {
    type: Boolean,
    required: true,
  },
})

const emit = defineEmits(['close', 'copy-url', 'open-url', 'complete', 'retry'])

const statusText = computed(() => {
  switch (props.dialog.status) {
    case 'starting':
      return '正在生成安全链接'
    case 'completing':
      return '正在等待并导入授权结果'
    case 'success':
      return '登录完成'
    case 'error':
      return '登录没有完成'
    default:
      return '等待浏览器授权'
  }
})

function close() {
  if (!props.busy) emit('close')
}
</script>

<template>
  <div class="modal-backdrop codex-login-backdrop" @click.self="close">
    <section
      class="modal codex-login-modal"
      role="dialog"
      aria-modal="true"
      aria-labelledby="codex-login-title"
      aria-describedby="codex-login-description"
    >
      <header class="codex-login-head">
        <div>
          <span class="section-kicker">Codex OAuth</span>
          <h2 id="codex-login-title">获取 Codex 登录链接</h2>
          <p id="codex-login-description">链接不会自动打开。你可以复制后在任意浏览器完成授权。</p>
        </div>
        <button type="button" class="icon-button" :disabled="busy" aria-label="关闭登录弹窗" @click="close">×</button>
      </header>

      <div class="codex-login-status" :class="`is-${dialog.status}`" role="status" aria-live="polite">
        <span class="codex-login-status-dot" aria-hidden="true"></span>
        <div>
          <strong>{{ statusText }}</strong>
          <p>{{ dialog.message }}</p>
        </div>
      </div>

      <div v-if="dialog.authUrl" class="codex-login-link-card">
        <div class="codex-login-link-head">
          <span>授权链接</span>
          <button type="button" class="ghost-button compact-button" @click="emit('copy-url', dialog.authUrl, '授权链接')">
            复制链接
          </button>
        </div>
        <code>{{ dialog.authUrl }}</code>
        <button type="button" class="primary-button codex-login-open-button" @click="emit('open-url')">
          打开登录页面
        </button>
      </div>

      <ol v-if="dialog.status === 'ready' || dialog.status === 'completing'" class="codex-login-steps">
        <li>打开上面的登录页面并完成 ChatGPT 授权。</li>
        <li>授权成功后返回 OmniProxy，点击“完成授权并导入”。</li>
      </ol>

      <div class="codex-login-format">
        <span>导入格式</span>
        <strong>CPA Codex JSON</strong>
        <small>access_token、refresh_token、id_token 等字段使用 CLIProxyAPI 兼容的顶层格式。</small>
      </div>

      <div class="modal-actions codex-login-actions">
        <button type="button" class="ghost-button" :disabled="busy" @click="close">关闭</button>
        <button v-if="dialog.status === 'error'" type="button" class="ghost-button" :disabled="busy" @click="emit('retry')">
          重新生成链接
        </button>
        <button v-if="dialog.status === 'ready'" type="button" class="primary-button" @click="emit('complete')">
          完成授权并导入
        </button>
        <button v-else-if="dialog.status === 'success'" type="button" class="primary-button" @click="close">
          完成
        </button>
      </div>
    </section>
  </div>
</template>

<style src="./CodexLoginModal.css"></style>
