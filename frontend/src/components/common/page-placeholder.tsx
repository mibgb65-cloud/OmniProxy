import type { ReactNode } from "react"
import { useTranslation } from "../../hooks/use-translation"
import type { TranslationKey } from "../../i18n/translation"

interface PagePlaceholderProps {
  /** Translation key for the page title. */
  titleKey: TranslationKey
  /** Translation key for the page description. */
  descriptionKey: TranslationKey
  children?: ReactNode
}

/**
 * Placeholder shell for pages whose backend feature has not landed yet.
 * Keeps routing and navigation stable while features are built incrementally.
 */
export function PagePlaceholder({ titleKey, descriptionKey, children }: PagePlaceholderProps) {
  const { t } = useTranslation()
  return (
    <section className="mx-auto max-w-4xl">
      <h1 className="text-2xl font-semibold tracking-tight">{t(titleKey)}</h1>
      <p className="text-muted-foreground mt-1 text-sm">{t(descriptionKey)}</p>
      {children ? <div className="mt-6">{children}</div> : null}
    </section>
  )
}
