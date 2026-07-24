<script setup>
import { Moon, Sunny, SwitchButton } from '@element-plus/icons-vue'

defineProps({
  activeTab: { type: String, required: true },
  activeTokensCount: { type: Number, required: true },
  appIconUrl: { type: String, required: true },
  appThemeLabel: { type: String, required: true },
  configProxyPort: { type: [Number, String], required: true },
  isDark: { type: Boolean, default: false },
  mobileSidebarOpen: { type: Boolean, default: false },
  navSections: { type: Array, required: true },
  proxyEndpoint: { type: String, required: true },
  proxyStatus: { type: Object, required: true },
  tabIcons: { type: Object, required: true },
  tokensCount: { type: Number, required: true },
})

defineEmits([
  'close-mobile-sidebar',
  'select-tab',
  'toggle-app-theme',
  'toggle-proxy',
])
</script>

<template>
  <div
    v-if="mobileSidebarOpen"
    class="mobile-sidebar-backdrop"
    aria-hidden="true"
    @click="$emit('close-mobile-sidebar')"
  ></div>

  <aside class="sidebar" :class="{ open: mobileSidebarOpen }">
    <div class="brand">
      <div class="brand-mark">
        <img :src="appIconUrl" alt="" width="42" height="42" />
      </div>
      <div>
        <strong>OmniProxy</strong>
        <span>本地 API 网关</span>
      </div>
    </div>

    <div class="sidebar-status">
      <div class="sidebar-status-main">
        <div :class="['status-light', { online: proxyStatus.running }]"></div>
        <div>
          <strong>{{ proxyStatus.running ? '代理运行中' : '代理未启动' }}</strong>
          <span>{{ proxyEndpoint }} · {{ tokensCount }} 个账号</span>
        </div>
      </div>
      <div class="sidebar-status-meta">
        <div>
          <span>端口</span>
          <strong>{{ proxyStatus.port || configProxyPort }}</strong>
        </div>
        <div>
          <span>可用账号</span>
          <strong>{{ activeTokensCount }}</strong>
        </div>
        <div>
          <span>状态</span>
          <strong>{{ proxyStatus.running ? '运行中' : '已停止' }}</strong>
        </div>
      </div>
      <button type="button" class="sidebar-proxy-button" @click="$emit('toggle-proxy')">
        <component :is="SwitchButton" class="button-icon" aria-hidden="true" />
        <span>{{ proxyStatus.running ? '停止代理' : '启动代理' }}</span>
      </button>
    </div>

    <nav class="nav-list">
      <section v-for="section in navSections" :key="section.label" class="nav-section">
        <span class="nav-section-label">{{ section.label }}</span>
        <button
          v-for="tab in section.items"
          :key="tab.key"
          type="button"
          :class="{ active: activeTab === tab.key }"
          @click="$emit('select-tab', tab.key)"
        >
          <component :is="tabIcons[tab.key]" class="nav-icon" aria-hidden="true" />
          <span>{{ tab.label }}</span>
        </button>
      </section>
    </nav>

    <div class="sidebar-tools">
      <button type="button" class="ghost-button" @click="$emit('toggle-app-theme')">
        <component :is="isDark ? Sunny : Moon" class="button-icon" aria-hidden="true" />
        <span>{{ appThemeLabel }}</span>
      </button>
    </div>
  </aside>
</template>
