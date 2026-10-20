import { AppService } from "../../bindings/omniproxy/internal/transport/wails"

export type {
  Quota as GoogleSubscriptionQuota,
  QuotaGroup as GoogleQuotaGroup,
  QuotaBucket as GoogleQuotaBucket,
  AccountModel as GoogleAccountModel,
} from "../../bindings/omniproxy/internal/provider/antigravity"

export const readGoogleSubscriptionQuota = async (id: string) =>
  AppService.GoogleSubscriptionQuota(id)

export const readCachedGoogleSubscriptionQuota = async (id: string) =>
  AppService.GoogleSubscriptionCachedQuota(id)

export const readGoogleSubscriptionModels = async (id: string) =>
  AppService.GoogleSubscriptionModels(id)
