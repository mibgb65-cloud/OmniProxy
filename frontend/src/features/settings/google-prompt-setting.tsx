import { useId } from "react"
import { useTranslation } from "react-i18next"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/ui/card"
import { Switch } from "../../components/ui/switch"
import { useDesktopSettings, useSetGooglePromptAsWritten } from "./use-desktop-settings"

/**
 * GooglePromptSetting 控制 Google 订阅通道是否按原文发送 system prompt。
 * 默认打散 Claude Code／Codex 等 agent 的几行身份与计费标记：Google 会把这些行
 * 当成额度耗尽拒绝整个请求，而打散后模型仍按原意阅读。
 */
export function GooglePromptSetting() {
  const { t } = useTranslation()
  const toggleId = useId()
  const settings = useDesktopSettings()
  const save = useSetGooglePromptAsWritten()
  const asWritten = settings.data?.googlePromptAsWritten ?? false
  const pending = save.isPending || settings.isLoading
  const error = settings.error ?? save.error

  return (
    <Card className="gap-0 rounded-2xl py-4 shadow-none">
      <CardHeader className="px-4">
        <CardTitle>{t("settings.googlePrompt.title")}</CardTitle>
        <CardDescription>{t("settings.googlePrompt.description")}</CardDescription>
      </CardHeader>
      <CardContent className="px-4">
        <label
          htmlFor={toggleId}
          className={`flex items-center justify-between gap-4 ${pending ? "cursor-default opacity-60" : "cursor-pointer"}`}
        >
          <span className="min-w-0">
            <span className="block font-medium">{t("settings.googlePrompt.asWritten")}</span>
            <span className="text-muted-foreground mt-1 block text-xs">
              {t("settings.googlePrompt.hint")}
            </span>
          </span>
          <Switch
            id={toggleId}
            checked={asWritten}
            disabled={pending}
            onCheckedChange={(checked) => save.mutate(checked)}
          />
        </label>
        {error ? (
          <p className="text-destructive mt-2 text-xs" role="alert">
            {t("settings.googlePrompt.failed", { error: error.message })}
          </p>
        ) : null}
      </CardContent>
    </Card>
  )
}
