import { useTranslation } from "react-i18next"
import { RuntimeCard } from "./runtime-card"
import { UpdateCard } from "./update-card"

export function AboutView() {
  const { t } = useTranslation()
  return (
    <section className="w-full min-w-0 space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">{t("about.title")}</h1>
        <p className="text-muted-foreground mt-1 text-sm">{t("about.description")}</p>
      </header>
      <RuntimeCard />
      <UpdateCard />
    </section>
  )
}
