import { useTranslation } from "react-i18next"
import type { AccountItem } from "../../services/omniproxy"
import { Button } from "../../components/ui/button"
import { AccountTokenSummary } from "./account-token-summary"
import { useAccountTokenUsage } from "./use-account-token-usage"

// 本机留存用量：没有官方额度／余额接口的账号（River、自定义提供商等）用它展示已用 Token。
// 请求进行中的状态不在这里追加占位行（会改变卡片高度），改由卡片头部的「正在使用」圆环表达。
export function AccountUsagePanel({ account }: { account: AccountItem }) {
  const { t } = useTranslation()
  const query = useAccountTokenUsage(account)

  return (
    <div className="min-w-0 space-y-2">
      <AccountTokenSummary usage={query.data} error={query.isError} />
      {query.isPending ? (
        <p role="status" className="text-muted-foreground text-xs">
          {t("activity.accountUsageLoading")}
        </p>
      ) : query.isError ? (
        <div role="alert" className="flex flex-wrap items-center gap-2 text-xs">
          <span className="text-destructive">{t("activity.accountUsageFailed")}</span>
          <Button
            size="sm"
            variant="outline"
            disabled={query.isFetching}
            onClick={() => void query.refetch()}
          >
            {t("activity.accountUsageRetry")}
          </Button>
        </div>
      ) : null}
    </div>
  )
}
