import { Bot } from "lucide-react"

const brandIcons: Record<string, { file: string; monochrome?: boolean; darkTile?: boolean }> = {
  openai: { file: "openai.svg", monochrome: true },
  anthropic: { file: "anthropic.svg", monochrome: true },
  google: { file: "google-color.svg" },
  deepseek: { file: "deepseek-color.svg" },
  zhipu: { file: "zhipu-color.svg" },
  kimi: { file: "kimi-color.svg", darkTile: true },
  minimax: { file: "minimax-color.svg" },
  mimo: { file: "xiaomimimo.svg", monochrome: true },
  stepfun: { file: "stepfun-color.svg" },
}

export function ProviderBrandIcon({ kind }: { kind: string }) {
  const brandIcon = brandIcons[kind]
  return (
    <span
      className={`flex size-9 shrink-0 items-center justify-center rounded-lg ${brandIcon?.monochrome || brandIcon?.darkTile ? "bg-zinc-950" : brandIcon ? "border-border border bg-white" : "bg-slate-700 text-white"}`}
      aria-hidden="true"
    >
      {brandIcon ? (
        <img
          src={`/provider-icons/${brandIcon.file}`}
          alt=""
          className={`size-6 ${brandIcon.monochrome ? "brightness-0 invert" : ""}`}
        />
      ) : (
        <Bot className="size-5" />
      )}
    </span>
  )
}
