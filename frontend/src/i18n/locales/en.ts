import type { zh } from "./zh"

/**
 * English translations. The `Record<keyof typeof zh, string>` type makes any
 * missing or extra key a compile error.
 */
export const en: Record<keyof typeof zh, string> = {
  "app.title": "OmniProxy - Local AI API Gateway",

  "common.language": "Language",
  "common.collapse": "Collapse",
  "common.expand": "Expand",

  "nav.dashboard": "Dashboard",
  "nav.accounts": "Accounts",
  "nav.providers": "Providers",
  "nav.gateway": "Gateway",
  "nav.routing": "Routing",
  "nav.quota": "Quota",
  "nav.usage": "Usage",
  "nav.requests": "Requests",
  "nav.logs": "Logs",
  "nav.settings": "Settings",

  "dashboard.title": "Dashboard",
  "dashboard.description": "OmniProxy runtime overview. Feature panels arrive with their backends.",
  "dashboard.application": "Application",
  "dashboard.applicationDescription": "Runtime information from the Go backend",
  "dashboard.version": "Version",
  "dashboard.profile": "Profile",
  "dashboard.proxyPort": "Proxy port",
  "dashboard.controlPort": "Control port",
  "dashboard.dataDirectory": "Data directory",
  "dashboard.loading": "Loading…",
  "dashboard.loadFailed": "Failed to load app info: {error}",
  "dashboard.bindingCheck": "Binding check",
  "dashboard.bindingCheckDescription": "Wails typed binding round-trip",
  "dashboard.ping": "Ping backend",
  "dashboard.pinging": "Pinging…",
  "dashboard.pingFailed": "Ping failed: {error}",
  "dashboard.pingReply": "Backend replied: {result}",

  "pages.accounts.title": "Accounts",
  "pages.accounts.description":
    "Multi-account management. The account backend lands in a later milestone.",
  "pages.providers.title": "Providers",
  "pages.providers.description":
    "Provider configuration and health. The provider backend lands in a later milestone.",
  "pages.gateway.title": "Gateway",
  "pages.gateway.description":
    "Proxy server status and control. The gateway backend lands in a later milestone.",
  "pages.routing.title": "Routing",
  "pages.routing.description":
    "Model routing rules. The routing engine lands in a later milestone.",
  "pages.quota.title": "Quota",
  "pages.quota.description": "Quota monitoring per account and provider.",
  "pages.usage.title": "Usage",
  "pages.usage.description": "Token and request usage statistics.",
  "pages.requests.title": "Requests",
  "pages.requests.description": "Request history and failure analysis.",
  "pages.logs.title": "Logs",
  "pages.logs.description": "Structured application logs from the Go backend.",
  "pages.settings.title": "Settings",
  "pages.settings.description": "Application settings and client integrations.",
}
