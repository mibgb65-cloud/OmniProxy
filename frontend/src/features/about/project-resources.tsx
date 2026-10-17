import { CodeXml, FileText, Globe, ShieldAlert, ShieldCheck } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card"

const RESOURCES = [
  { label: "about.resources.website", icon: Globe },
  { label: "about.resources.repository", icon: CodeXml },
  { label: "about.resources.disclaimer", icon: ShieldAlert },
  { label: "about.resources.terms", icon: FileText },
  { label: "about.resources.privacy", icon: ShieldCheck },
] as const

export function ProjectResources() {
  const { t } = useTranslation()
  return (
    <Card className="gap-0 rounded-2xl py-0 shadow-none">
      <CardHeader className="px-5 pt-4 pb-2">
        <CardTitle className="text-sm">{t("about.resources.title")}</CardTitle>
      </CardHeader>
      <CardContent className="px-5 pb-1">
        <div className="divide-y">
          {RESOURCES.map(({ label, icon: Icon }) => (
            <button
              key={label}
              type="button"
              disabled
              className="flex min-h-12 w-full cursor-default items-center gap-3 py-3 text-left text-sm"
            >
              <Icon className="text-muted-foreground size-4 shrink-0" aria-hidden="true" />
              <span className="min-w-0 flex-1">{t(label)}</span>
              <span className="bg-muted text-muted-foreground shrink-0 rounded-full px-2 py-0.5 text-xs">
                {t("about.resources.pending")}
              </span>
            </button>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
