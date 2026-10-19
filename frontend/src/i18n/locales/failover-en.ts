export const failoverEn = {
  "failover.quotaExhausted": "Subscription quota exhausted; waiting for the window to reset",
  "failover.duplicate": "This account uses credentials already tried and was skipped.",
  "failover.title": "Account failover",
  "failover.description":
    "Try another account within the same provider, billing mode and region. Fixed accounts stay fixed.",
  "failover.boundary":
    "Try up to {{count}} accounts; never retry after a successful upstream response.",
  "failover.saving": "Saving…",
  "failover.failed": "Could not load or save. Please try again.",
  "failover.retry": "Retry loading failover settings",
  "failover.cooling": "{{count}} account resources temporarily unavailable",
  "failover.healthy": "No account resources are cooling down",
  "failover.until": "May recover at {{time}}",
  "failover.recovering": "Checking recovery with the current request",
  "failover.account": "Saved account",
  "failover.more": "{{count}} more",
  "failover.attempts": "Account attempts",
  "failover.attempt": "Attempt {{index}} · {{account}}",
  "failover.switched": "Switched {{count}} times",
} as const
