import { Globe2 } from "lucide-react"
import { useTranslation } from "react-i18next"

/** WorkBuddy 只支持浏览器授权：用一张静态卡片说明登录方式，不提供设备码选项。 */
export function SubscriptionBrowserLoginCard() {
  const { t } = useTranslation()
  return (
    <div className="border-primary/65 bg-primary/[0.04] flex flex-col gap-3 rounded-xl border p-4">
      <Globe2 className="text-foreground size-5" aria-hidden="true" />
      <span className="space-y-1.5">
        <span className="block text-sm font-medium">{t("subscription.browser")}</span>
        <span className="text-muted-foreground block text-xs leading-5">
          {t("subscription.browserHint")}
        </span>
      </span>
    </div>
  )
}
