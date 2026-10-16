import type { googleZh } from "./google-zh"
export const googleEn: Record<keyof typeof googleZh, string> = {
  "google.subscriptionAdd": "Sign in with Google",
  "google.subscriptionTitle": "Sign in to Google AI",
  "google.subscriptionDescription":
    "Authorize Antigravity on Google's official page to use Gemini models and subscription quota available to your account. Initial sign-in requires a compatible local Antigravity CLI.",
  "google.subscriptionStorage":
    "Authorization is stored separately in protected local credentials. Use the same Google account when signing in again.",
  "google.subscriptionType": "Google / Antigravity",
  "google.quota.title": "Antigravity subscription quota",
  "google.models.title": "Available models",
  "google.models.refresh": "Refresh Google model list",
  "google.models.loading": "Loading models available to this account…",
  "google.models.show": "View all {{count}} account models",
  "google.models.close": "Close account model list",
  "google.models.count": "{{supported}} connected · {{total}} available upstream",
  "google.models.connected": "Connected to this gateway · Chat API",
  "google.models.pending": "Available upstream · Gateway support pending",
  "google.models.hint":
    "Connected models support Chat API; other entries describe the upstream catalog. Google determines access for each request.",
  "google.models.noAdapter": "These models are not yet supported by this gateway",
  "google.models.empty": "Google has not returned any selectable models.",
  "google.models.failed":
    "The model list is unavailable. Refresh to retry; quota queries are independent.",
  "google.quota.refresh": "Refresh Google subscription quota",
  "google.quota.loading": "Loading official Google quota…",
  "google.quota.missing": "Google has not returned any available quota windows.",
  "google.quota.gemini": "Gemini models",
  "google.quota.thirdParty": "Third-party models (Claude / GPT)",
  "google.quota.fiveHour": "5h quota",
  "google.quota.weekly": "Weekly quota",
  "google.quota.exhausted": "A quota window is exhausted",
  "google.quota.hint":
    "Official shared Antigravity quota, shown separately by model group; never added together or converted to request counts.",
  "subscription.error.quota_changed": "Account authorization changed. Refresh quota and try again.",
  "subscription.error.agy_unavailable":
    "Antigravity CLI (agy) was not found. Install the official component and try again.",
  "subscription.error.client_incompatible":
    "The installed Antigravity client configuration is not recognized. Check the official component version.",
  "subscription.error.reauth_required": "Google authorization expired. Sign in again.",
  "subscription.error.access_denied":
    "Google denied access. Check account permissions and supported regions.",
  "subscription.error.rate_limited": "Google is rate limiting requests. Try again later.",
  "subscription.error.project_unavailable":
    "Google did not return an Antigravity project. Complete account setup in the official application first.",
  "subscription.error.invalid_response":
    "Google returned an unrecognized response. Try again later.",
  "subscription.error.upstream_unavailable":
    "Google could not be reached. Check your connection and try again.",
  "subscription.error.credential_unavailable":
    "Local Google authorization could not be read. Sign in again.",
  "subscription.error.invalid_input":
    "Authorization parameters are invalid. Close the dialog and start again.",
  "subscription.error.cancelled": "Authorization was cancelled. You can start again.",
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
