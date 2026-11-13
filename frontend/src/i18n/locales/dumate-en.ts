/**
 * Baidu DuMate account, import and point-usage copy (English).
 */
export const dumateEn = {
  "dumate.account.cookieType": "Baidu sign-in session",
  "dumate.account.imported": "Session imported",
  "dumate.account.needsReimport": "Session expired; import it again",
  "dumate.account.editHint": "A cookie session cannot be replaced here; import the session again.",
  "dumate.account.hint":
    "DuMate signs in with a Baidu account and spends personal points through smart routing; upstream model identity is not exposed.",
  "dumate.add": "Import DuMate account",
  "dumate.import.title": "Add DuMate account",
  "dumate.import.description":
    "Reuse the local DuMate desktop sign-in, or paste the Cookie request header from a browser.",
  "dumate.import.desktop": "Import local desktop session",
  "dumate.import.desktopHint":
    "Reads auth.json and .cookie-key under %APPDATA%\\qianfan-desktop-app only; the client's files are never modified.",
  "dumate.import.cookie": "Paste browser cookie",
  "dumate.import.cookieHint":
    "Sign in at console.bce.baidu.com, open DevTools → Network, reload the page and copy the Cookie request header of any request. The key sign-in cookies are httpOnly, so document.cookie cannot see them.",
  "dumate.import.cookiePlaceholder": "bce-sessionid=...; bce-user-info=...",
  "dumate.import.submit": "Import",
  "dumate.import.importing": "Checking the session…",
  "dumate.import.created": "Added a new account: {{name}}",
  "dumate.import.updated": "Updated the session of the same account: {{name}}",
  "dumate.import.storage":
    "The cookie is stored only in the local protected credential store; it never enters the database or logs.",
  "dumate.import.close": "Close",
  "dumate.import.error.desktop_login_missing":
    "No local DuMate session was found. Install and sign in to DuMate first, or use the cookie method.",
  "dumate.import.error.desktop_login_invalid":
    "The local sign-in file cannot be read (its format may have changed). Sign in to DuMate again, or use the cookie method.",
  "dumate.import.error.cookie_invalid": "The cookie cannot be parsed, or bce-sessionid is missing.",
  "dumate.import.error.sign_in_expired": "The session expired. Sign in again and re-import.",
  "dumate.import.error.identity_changed":
    "This session belongs to a different Baidu account; add it as its own account. Existing accounts are never overwritten.",
  "dumate.import.error.not_dumate_provider": "This provider is not Baidu DuMate.",
  "dumate.import.error.not_dumate_account": "This account is not a DuMate session account.",
  "dumate.import.error.invalid_account": "The account no longer exists.",
  "dumate.import.error.credential_unavailable": "The protected local session cannot be read.",
  "dumate.import.error.console_unavailable":
    "The Baidu console is unreachable; check the network or proxy settings.",
  "dumate.import.error.rate_limited": "The Baidu console is rate limited. Try again later.",
  "dumate.import.error.invalid_response": "The Baidu console returned an unrecognized response.",
  "dumate.import.error.request_rejected": "The Baidu console rejected the request.",
  "dumate.import.error.operation_failed": "Import failed. Try again.",
  "dumate.points.title": "Points and check-in",
  "dumate.points.remaining": "Available points",
  "dumate.points.used": "Used / total",
  "dumate.points.usage": "{{used}} / {{total}}",
  "dumate.points.remainingValue": "{{points}} points",
  "dumate.points.expiring": "{{points}} points expire on {{time}}",
  "dumate.points.checkin": "Checked in today ({{days}} days total)",
  "dumate.points.checkinOpen": "Not checked in today",
  "dumate.points.checkinUnknown": "Check-in status unknown",
  "dumate.points.checkinButton": "Check in",
  "dumate.points.checking": "Checking in…",
  "dumate.points.checkinClaimed": "Checked in{{credit}} ({{days}} days total)",
  "dumate.points.checkinDone": "Already checked in today ({{days}} days total)",
  "dumate.points.checkinPending":
    "Check-in submitted; points may arrive later. Refresh in a moment.",
  "dumate.points.checkinFailed": "Check-in failed: {{reason}}",
  "dumate.points.refresh": "Refresh points",
  "dumate.points.loading": "Reading points…",
  "dumate.points.fetchedAt": "Fetched {{time}}",
  "dumate.points.never": "Not fetched yet",
  "dumate.points.stale":
    "Showing the last successful result ({{time}}); the current query failed: {{reason}}",
  "dumate.points.unavailable": "Points query failed: {{reason}}",
  "dumate.reason.sign_in_expired": "the session expired",
  "dumate.reason.console_unavailable": "the console is unreachable",
  "dumate.reason.invalid_response": "the console returned an unexpected response",
  "dumate.reason.rate_limited": "the console is rate limited",
  "dumate.reason.request_rejected": "the console rejected the request",
  "dumate.reason.credential_unavailable": "the local session is unavailable",
  "dumate.reason.verify_failed": "the credit could not be verified",
  "dumate.reason.claim_failed": "the check-in request failed",
  "dumate.reason.operation_failed": "an unknown error occurred",
} as const
