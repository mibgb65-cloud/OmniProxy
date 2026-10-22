import { useMutation } from "@tanstack/react-query"
import { CodeXml, ExternalLink, FileText, Globe, ShieldAlert, ShieldCheck } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card"
import { openProjectResource, type ProjectResource } from "../../services/project-resources"

const RESOURCES = [
  { id: "website", label: "about.resources.website", icon: Globe },
  { id: "repository", label: "about.resources.repository", icon: CodeXml },
  { id: "disclaimer", label: "about.resources.disclaimer", icon: ShieldAlert },
  { id: "terms", label: "about.resources.terms", icon: FileText },
  { id: "privacy", label: "about.resources.privacy", icon: ShieldCheck },
] as const

export function ProjectResources() {
  const { t } = useTranslation()
  const open = useMutation({
    mutationFn: (resource: ProjectResource) => openProjectResource(resource),
    gcTime: 0,
  })
  return (
    <Card className="gap-0 rounded-2xl py-0 shadow-none">
      <CardHeader className="px-5 pt-4 pb-2">
        <CardTitle className="text-sm">{t("about.resources.title")}</CardTitle>
        <p className="text-muted-foreground text-xs">{t("about.resources.browserHint")}</p>
      </CardHeader>
      <CardContent className="px-5 pb-1">
        <div className="divide-y">
          {RESOURCES.map(({ id, label, icon: Icon }) => (
            <button
              key={id}
              type="button"
              disabled={open.isPending}
              onClick={() => open.mutate(id)}
              className="hover:text-primary focus-visible:ring-ring flex min-h-12 w-full cursor-pointer items-center gap-3 rounded-sm py-3 text-left text-sm focus-visible:ring-2 focus-visible:outline-none disabled:cursor-wait disabled:opacity-60"
            >
              <Icon className="text-muted-foreground size-4 shrink-0" aria-hidden="true" />
              <span className="min-w-0 flex-1">{t(label)}</span>
              <ExternalLink
                className="text-muted-foreground size-3.5 shrink-0"
                aria-hidden="true"
              />
            </button>
          ))}
        </div>
        {open.isError ? (
          <p role="alert" className="text-destructive pb-3 text-xs">
            {t("about.resources.openFailed")}
          </p>
        ) : null}
      </CardContent>
    </Card>
  )
}
