import type { googleZh } from "./google-zh"
export const googleEn: Record<keyof typeof googleZh, string> = {
  "google.resources": "Gemini project resources",
  "google.description":
    "Keys in the same project share rate limits. This view shows declared projects and locally observed errors; official remaining resources are not connected.",
  "google.unknown": "Unknown (official quota query not connected)",
  "google.officialRemaining": "Official balance / remaining quota",
  "google.unbound": "Project not bound",
  "google.shared": "User-declared project, not verified; keys share project quota.",
  "google.scopeUnknown": "Project scope is unknown; this is not an independent quota pool.",
  "google.localUsage":
    "Local token usage is available on the Usage page and cannot determine official balance.",
  "google.bind": "Bind project",
  "google.bindAccount": "Bind project: {{name}}",
  "google.bindingHint":
    "Enter the Google Cloud project ID for this key. Use the same ID for keys in one project. This only saves a local declaration.",
  "google.projectId": "Project ID",
  "google.projectHint":
    "6–30 lowercase letters, digits or hyphens; start with a letter and end with a letter or digit. Leave empty to unbind. Bind again after key replacement.",
  "google.cancel": "Cancel",
  "google.save": "Save",
  "google.saving": "Saving…",
  "google.refresh": "Refresh local records",
  "google.consoleUsage": "Official usage / limits",
  "google.consoleBilling": "Official billing",
  "google.usage": "Local usage",
  "google.loading": "Loading project records…",
  "google.error.changed": "The account changed. Refresh accounts before binding again.",
  "google.error.invalid": "Invalid project ID or account. Check and retry.",
  "google.error.failed": "Operation failed. Please retry.",
  "google.rateLimited": "Rate limit observed",
  "google.dailyExhausted": "Daily quota exhaustion observed",
  "google.paymentRequired": "Payment required observed",
  "google.authenticationFailed": "Authentication failure observed",
  "google.permissionDenied": "Permission denial observed",
  "google.observed": "Observed upstream error",
  "google.observedAt": "Observed locally at {{time}}",
  "google.limit": "Limit from official error: {{count}}",
  "google.retryAt": "Upstream suggested retry time: {{time}}",
  "google.dailyReset": "Next daily reset (Pacific midnight): {{time}}",
  "google.stale": "Expired observation for reference only; recovery has not been confirmed.",
  "google.noObservations":
    "No local rate limit or payment errors observed. This does not confirm available official quota.",
  "google.chatOnly":
    "Gemini currently supports Chat API. Sync official models on the Routing page before calling a model ID.",
  "models.inputLimit": "Input limit: {{count}} tokens",
  "models.outputLimit": "Output limit: {{count}} tokens",
}
