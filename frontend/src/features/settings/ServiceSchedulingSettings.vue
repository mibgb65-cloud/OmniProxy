<script setup>
defineProps({
  config: {
    type: Object,
    required: true,
  },
})
</script>

<template>
  <section class="settings-section settings-service-section">
    <div class="settings-section-head">
      <div>
        <h3>本机服务</h3>
        <p>端口和本地代理能力，端口变更后需要重启代理。</p>
      </div>
    </div>
    <div class="settings-grid compact-settings-grid">
      <label>
        <span>代理端口</span>
        <input v-model="config.proxyPort" type="number" min="1" max="65535" />
      </label>
      <label>
        <span>控制端口</span>
        <input v-model="config.controlPort" type="number" min="1" max="65535" />
      </label>
      <label class="toggle-field">
        <span>启用 Codex WebSocket</span>
        <input
          v-model="config.websocketMode"
          class="toggle-input"
          type="checkbox"
          true-value="enabled"
          false-value="disabled"
        />
        <span class="toggle-switch" aria-hidden="true">
          <span class="toggle-thumb"></span>
        </span>
      </label>
    </div>
  </section>

  <section class="settings-section settings-scheduling-section">
    <div class="settings-section-head">
      <div>
        <h3>调度与保护</h3>
        <p>控制账号轮换、低额度跳过和失败重试。</p>
      </div>
    </div>
    <div class="settings-grid compact-settings-grid">
      <label class="toggle-field wide-field codex-auto-activation-field">
        <span class="codex-auto-activation-copy">
          <strong>自动激活 Codex 额度窗口</strong>
          <small id="codex-auto-activation-help">
            开启并保存后，检测到 5 小时或周额度尚未开始计时时，会用该账号自动发送一条最小消息并复查；已激活账号不会重复发送。
          </small>
        </span>
        <input
          v-model="config.codexAutoActivateUsage"
          class="toggle-input"
          type="checkbox"
          aria-label="自动激活 Codex 额度窗口"
          aria-describedby="codex-auto-activation-help"
        />
        <span class="toggle-switch" aria-hidden="true">
          <span class="toggle-thumb"></span>
        </span>
      </label>
      <div class="settings-segmented-field">
        <span>账号调度模式</span>
        <div
          :class="[
            'settings-segmented',
            'settings-segmented-two',
            config.schedulingMode === 'balanced' ? 'active-right' : 'active-left',
          ]"
          role="group"
          aria-label="账号调度模式"
        >
          <button
            type="button"
            :class="{ active: config.schedulingMode === 'queue' }"
            :aria-pressed="config.schedulingMode === 'queue'"
            @click="config.schedulingMode = 'queue'"
          >
            队列模式
          </button>
          <button
            type="button"
            :class="{ active: config.schedulingMode === 'balanced' }"
            :aria-pressed="config.schedulingMode === 'balanced'"
            @click="config.schedulingMode = 'balanced'"
          >
            优先平衡使用
          </button>
        </div>
      </div>
      <label>
        <span>额度切换阈值</span>
        <input v-model="config.switchThreshold" type="number" min="1" max="100" />
      </label>
      <label>
        <span>自动重试次数</span>
        <input v-model="config.maxRetries" type="number" min="0" max="5" />
      </label>
      <div class="settings-segmented-field">
        <span>MiMo 优先使用</span>
        <div
          :class="[
            'settings-segmented',
            'settings-segmented-two',
            config.xiaomiCredentialPriority === 'api_key' ? 'active-right' : 'active-left',
          ]"
          role="group"
          aria-label="MiMo 凭据优先级"
        >
          <button
            type="button"
            :class="{ active: config.xiaomiCredentialPriority === 'mimo_token_plan' }"
            :aria-pressed="config.xiaomiCredentialPriority === 'mimo_token_plan'"
            @click="config.xiaomiCredentialPriority = 'mimo_token_plan'"
          >
            Token Plan
          </button>
          <button
            type="button"
            :class="{ active: config.xiaomiCredentialPriority === 'api_key' }"
            :aria-pressed="config.xiaomiCredentialPriority === 'api_key'"
            @click="config.xiaomiCredentialPriority = 'api_key'"
          >
            按量 API
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
