import { useTranslation } from "react-i18next"
import { RuntimeCard } from "./runtime-card"
import { ProjectResources } from "./project-resources"

export function AboutView({ embedded = false }: { embedded?: boolean }) {
  const { t } = useTranslation()
  return (
    <section className="w-full min-w-0 space-y-4">
      {!embedded ? (
        <header>
          <h1 className="text-2xl font-semibold tracking-tight">{t("about.title")}</h1>
          <p className="text-muted-foreground mt-1 text-sm">{t("about.description")}</p>
        </header>
      ) : null}
      <ProjectResources />
      <RuntimeCard />
    </section>
  )
}
