export const subscriptionEn = {
  "subscription.error.initialization_failed":
    "The login component or authorization directory could not be initialized. Check the component installation and directory access.",
  "subscription.error.request_failed":
    "The official login component request failed. Try again later.",
  "subscription.error.operation_cancelled": "Authorization was cancelled. Start again.",
  "subscription.error.operation_timeout": "The login component timed out. Try again later.",
  "subscription.error.refresh_timeout":
    "Account status refresh timed out. The previous state was kept; try again later.",
  "subscription.error.refresh_busy": "Another account operation is still running. Refresh later.",
  "subscription.error.refresh_unavailable":
    "Account status could not be refreshed. The previous state was kept; try again later.",
  "subscription.error.refresh_cancelled": "This refresh stopped. You can try again later.",
  "subscription.error.account_update_failed":
    "Account status could not be saved. Check the local data directory and try again.",
  "subscription.refreshDismiss": "Dismiss refresh notice",
  "subscription.error.invalid_account":
    "The account state changed. Reopen the login dialog and try again.",
  "subscription.error.invalid_provider":
    "This provider does not support ChatGPT subscription sign-in.",
  "subscription.reset.title": "Reset credits",
  "subscription.reset.receipt": "Most recent result",
  "subscription.reset.close": "Close reset credits",
  "subscription.reset.count": "Available reset credits",
  "subscription.reset.refresh": "Refresh reset credits",
  "subscription.reset.cardTitle": "Quota reset credit",
  "subscription.reset.granted": "Granted:",
  "subscription.reset.expires": "Expires:",
  "subscription.reset.noExpiry": "No expiry",
  "subscription.reset.expired": "Expired",
  "subscription.reset.status.available": "Available",
  "subscription.reset.status.redeeming": "Processing",
  "subscription.reset.status.redeemed": "Used",
  "subscription.reset.status.unknown": "Status unknown",
  "subscription.reset.use": "Use this credit",
  "subscription.reset.next": "the next available reset credit",
  "subscription.reset.useNext": "Use next available credit",
  "subscription.reset.confirm":
    "Use {{title}}? This consumes one reset credit to reset eligible quota windows.",
  "subscription.reset.confirmUse": "Confirm use",
  "subscription.reset.using": "Using…",
  "subscription.reset.cancel": "Cancel",
  "subscription.reset.pending":
    "The result is not confirmed. Verify this attempt before using another credit.",
  "subscription.reset.verify": "Verify this attempt",
  "subscription.reset.unavailable": "The service has not returned reset credit information.",
  "subscription.reset.empty": "No reset credits are currently available.",
  "subscription.reset.partial":
    "Some credit details are unavailable. The service can select the next available credit.",
  "subscription.reset.outcome.reset": "Reset credit used successfully.",
  "subscription.reset.outcome.alreadyRedeemed":
    "This reset was already completed. No additional credit was used.",
  "subscription.reset.outcome.nothingToReset": "No quota window is currently eligible for a reset.",
  "subscription.reset.outcome.noCredit": "No reset credit is currently available.",
  "subscription.reset.refreshFailed":
    "The result was returned, but the latest quota could not be loaded. Please refresh.",
  "subscription.error.reset_record_failed":
    "The operation record is unavailable. Refresh before continuing.",
  "subscription.error.reset_unavailable": "Reset credits are unavailable. Please refresh first.",
  "subscription.error.reset_no_credit": "No reset credits are available. Please refresh.",
  "subscription.error.reset_not_usable":
    "This credit cannot be used right now. Refresh to check its status.",
  "subscription.settings.title": "OpenAI account settings",
  "subscription.settings.close": "Close account settings",
  "subscription.settings.websocket": "Codex WebSocket",
  "subscription.settings.websocketDescription":
    "Allow Codex CLI and the desktop app to use WebSocket through this OmniProxy gateway. Disabling rejects WebSocket requests; HTTP/SSE remains available.",
  "subscription.settings.loading": "Loading connection settings…",
  "subscription.settings.loadFailed": "Could not load connection settings. Try again.",
  "subscription.settings.retry": "Reload settings",
  "subscription.settings.saveFailed":
    "Could not save. The previous switch state is kept. Try again.",
  "subscription.settings.saving": "Saving…",
  "subscription.settings.saved": "Settings saved and applied.",
  "subscription.rotation.title": "Account rotation",
  "subscription.rotation.queue": "Sequential",
  "subscription.rotation.balanced": "Balanced",
  "subscription.rotation.queueDescription":
    "Use the current account until its quota is exhausted, then switch to the next available account.",
  "subscription.rotation.balancedDescription":
    "Distribute requests across available accounts. With fresh quota data, accounts with more remaining quota receive more requests.",
  "subscription.activation.title": "Automatically activate Codex quota windows",
  "subscription.activation.description":
    "When enabled, a minimal message is sent with the same account when its five-hour or weekly window has not started, followed by a quota recheck. Active windows are skipped.",
  "subscription.activation.saved": "Automatic activation settings saved.",
  "subscription.activation.saveFailed":
    "Automatic activation settings were not saved. Your previous settings remain active. Try again.",
  "subscription.activation.checking": "Checking and rechecking quota windows…",
  "subscription.activation.summary":
    "Last check: {{checked}} accounts, {{sent}} messages sent, {{activated}} accounts confirmed with timers running.",
  "subscription.activation.dismissNotice": "Dismiss quota check notice",
  "subscription.activation.pending":
    "Some windows are still being observed or await confirmation. Requests already sent are only rechecked, never resent automatically.",
  "subscription.activation.failed":
    "Some checks or activations could not finish. Check account authorization and network.",
  "subscription.activation.reason.models":
    "An account's official model catalog is empty, so the activation message cannot pick a model. Sync that account's models first.",
  "subscription.activation.reason.quota":
    "Reading the account quota windows failed. It will be rechecked automatically.",
  "subscription.activation.reason.prepare":
    "The pre-send quota recheck did not pass, so no message was sent this round.",
  "subscription.activation.reason.request": "The activation message did not complete.",
  "subscription.activation.reason.record":
    "The activation record could not be written, so no message was sent this round.",
  "subscription.activation.reason.identity":
    "The account identity changed. Authorize that account again.",
  "subscription.activation.reason.timing":
    "Not enough timing evidence for the quota window yet; waiting for the next recheck.",
  "subscription.activation.reason.accounts": "Reading the account or provider list failed.",
  "subscription.activation.statusFailed": "Automatic activation status is temporarily unavailable.",
  "subscription.quota.title": "Quota / balance",
  "subscription.quota.retained": "Showing last recorded information",
  "subscription.error.quota_cache_unavailable":
    "Could not save or restore the quota snapshot. Previous data is retained. Please retry.",
  "subscription.quota.fiveHour": "5h quota",
  "subscription.quota.weekly": "Weekly quota",
  "subscription.quota.remainingLabel": "{{label}} remaining",
  "subscription.quota.loading": "Loading quota…",
  "subscription.quota.retry": "Refresh quota",
  "subscription.quota.signIn": "Sign in to check quota.",
  "subscription.quota.missingWindow": "Some quota windows are not available yet.",
  "subscription.quota.timeUnknown": "Reset unknown",
  "subscription.quota.timeUnknownHint": "The service has not returned a reset time.",
  "subscription.quota.notStarted": "Timer not started",
  "subscription.quota.expired": "Refresh needed",
  "subscription.quota.daysHours": "Resets in {{days}}d {{hours}}h",
  "subscription.quota.hoursMinutes": "Resets in {{hours}}h {{minutes}}m",
  "subscription.quota.minutes": "Resets in {{minutes}}m",
  "subscription.quota.resetAt": "Reset: {{time}}",
  "subscription.quota.created": "Created:",
  "subscription.quota.updated": "Updated:",
  "subscription.error.quota_timeout": "Quota query timed out. Check your connection and refresh.",
  "subscription.error.quota_busy":
    "Account operations are busy. Quota query wait timed out. Refresh shortly.",
  "subscription.error.quota_refresh_failed":
    "Authorization refresh failed. Check your connection and retry; if it persists, refresh the account status.",
  "subscription.error.quota_unavailable": "Quota could not be loaded. Please refresh later.",
  "subscription.error.quota_not_authenticated": "Sign in again to check quota.",
  "subscription.accountIdUnknown": "Pending",
  "subscription.accountIdPending":
    "Account ID is not available yet. Refresh the account to retrieve it.",
  "subscription.error.account_id_pending":
    "The account is saved, but its ID is not available yet. Refresh the account to retry.",
  "subscription.error.account_metadata_pending":
    "Authorization and account saved. Identity and plan could not be read yet. Choose Refresh status in the account details.",
  "subscription.error.protocol_failed":
    "The official login component disconnected. Start authorization again.",
  "subscription.error.account_unavailable":
    "Existing account or authorization details could not be read. Check the subscription authorization logs.",
  "subscription.error.login_failed":
    "Official sign-in did not complete. Start authorization again.",
  "subscription.backgroundHint":
    "Authorization continues in the background when this window closes. Reopen it to resume, or choose Cancel to stop authorization.",
  "subscription.close": "Close",
  "subscription.closeLabel": "Close subscription sign-in dialog",
  "subscription.deleteNotice":
    "Removes only this account's local record and credentials. Official websites and other clients stay signed in. Use Sign out separately when needed.",
  "subscription.add": "Sign in with ChatGPT",
  "subscription.title": "Sign in to ChatGPT",
  "subscription.type": "ChatGPT / Codex",
  "subscription.description":
    "Connect through the official OpenAI page to use your Codex subscription access.",
  "subscription.namePlaceholder": "Optional; defaults to account identity",
  "subscription.browserHint": "Sign in and authorize in your browser.",
  "subscription.deviceHint": "Enter a device code on the official page.",
  "subscription.secureStorage": "Credentials stay in your local system credential store.",
  "subscription.mode": "Sign-in method",
  "subscription.browser": "Browser authorization",
  "subscription.device": "Device code",
  "subscription.waiting": "Waiting for authorization on the official page",
  "subscription.code": "One-time device code",
  "subscription.waitHint":
    "Complete authorization in your browser. The official component may still be exchanging tokens; closing the page does not interrupt it. Account details are fetched after the completion notice. This session waits up to 10 minutes.",
  "subscription.open": "Open official authorization page",
  "subscription.copyLink": "Copy sign-in link",
  "subscription.linkCopied": "Sign-in link copied",
  "subscription.copyLinkHint":
    "Paste this link into another browser or browser profile on this computer. Do not share it.",
  "subscription.copyLinkFailed": "Could not copy the link. Try again.",
  "subscription.exchanging":
    "Authorization received. Exchanging tokens and fetching account details…",
  "subscription.verifying": "Official authorization complete. Fetching account details…",
  "subscription.saving": "Saving local credentials and account…",
  "subscription.processingHint":
    "You can close the page or this window; processing continues in the background. Do not restart authorization.",
  "subscription.completed": "Authorization complete. Account saved.",
  "subscription.cancelled": "Authorization cancelled.",
  "subscription.starting": "Starting sign-in…",
  "subscription.start": "Start authorization",
  "subscription.done": "Done",
  "subscription.relogin": "Sign in again",
  "subscription.refresh": "Refresh status",
  "subscription.logout": "Sign out",
  "subscription.planUnknown": "Unknown plan",
  "subscription.identityUnknown": "Unknown identity",
  "subscription.cleanupWarning":
    "Account saved; previous authorization cleanup needs a retry. Signing out or deleting the account retries cleanup.",
  "subscription.authentication": "Authentication",
  "subscription.editDescription":
    "Change this subscription account’s display name. Use “Sign in again” to update authorization.",
  "subscription.state.authenticated": "Authorized",
  "subscription.state.logged_out": "Signed out",
  "subscription.state.reauth_required": "Sign-in required",
  "subscription.state.unknown": "Authorization status unconfirmed",
  "subscription.error.codex_unavailable":
    "Official Codex CLI not found. Install Codex and make sure the codex command is available.",
  "subscription.error.login_expired": "Authorization timed out. Start again.",
  "subscription.error.busy": "Another account operation is in progress. Try again shortly.",
  "subscription.error.identity_changed":
    "The identity differs from the original account. Sign in with the original account or add a new account.",
  "subscription.error.cleanup_failed":
    "Authorization cleanup is incomplete. The next sign-in will retry cleanup first. Check the system credential store and try again.",
  "subscription.error.browser_failed":
    "Could not open the browser. Use “Open official authorization page” to retry.",
  "subscription.error.operation_failed":
    "Operation incomplete. Check your network and sign-in component, then try again.",
} as const
