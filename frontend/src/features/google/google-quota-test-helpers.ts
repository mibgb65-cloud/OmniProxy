import type { AccountItem } from "../../services/omniproxy"
import type { GoogleSubscriptionQuota } from "../../services/google-subscription"
import type { GoogleAccountModel } from "../../services/google-subscription"

export const googleModelsFixture: GoogleAccountModel[] = [
  { id: "gemini-current-high", name: "Gemini Current (High)", gatewaySupported: true },
  { id: "gemini-current-low", name: "Gemini Current (Low)", gatewaySupported: true },
  { id: "claude-current", name: "Claude Current", gatewaySupported: false },
]

export const googleQuotaAccount: AccountItem = {
  id: "google-quota-test",
  providerId: "builtin_google",
  name: "Google 额度测试",
  enabled: true,
  authMethod: "subscription_login",
  authState: "authenticated",
  authAccountId: "1001",
  authEmail: "test@example.invalid",
  authPlan: "g1-pro-tier",
  createdAt: "2026-10-16T00:00:00Z",
  updatedAt: "2026-10-16T00:00:00Z",
}

export const googleQuotaFixture: GoogleSubscriptionQuota = {
  fetchedAt: "2026-10-16T00:00:00Z",
  groups: [
    {
      name: "Gemini Models",
      buckets: [
        {
          id: "gemini-weekly",
          name: "Weekly Limit Remaining",
          window: "weekly",
          remainingPercent: 80,
          resetsAt: "2099-10-23T00:00:00Z",
        },
        {
          id: "gemini-5h",
          name: "Five Hour Limit Remaining",
          window: "5h",
          remainingPercent: 75,
          resetsAt: "2099-10-16T05:00:00Z",
        },
      ],
    },
    {
      name: "Claude and GPT models",
      buckets: [
        {
          id: "3p-weekly",
          name: "Weekly Limit Remaining",
          window: "weekly",
          remainingPercent: 60,
          resetsAt: "2099-10-23T00:00:00Z",
        },
        {
          id: "3p-5h",
          name: "Five Hour Limit Remaining",
          window: "5h",
          remainingPercent: 50,
          resetsAt: "2099-10-16T05:00:00Z",
        },
      ],
    },
  ],
}
