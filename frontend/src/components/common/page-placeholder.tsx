import type { ReactNode } from "react"

interface PagePlaceholderProps {
  title: string
  description: string
  children?: ReactNode
}

/**
 * Placeholder shell for pages whose backend feature has not landed yet.
 * Keeps routing and navigation stable while features are built incrementally.
 */
export function PagePlaceholder({ title, description, children }: PagePlaceholderProps) {
  return (
    <section className="mx-auto max-w-4xl">
      <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
      <p className="text-muted-foreground mt-1 text-sm">{description}</p>
      {children ? <div className="mt-6">{children}</div> : null}
    </section>
  )
}
