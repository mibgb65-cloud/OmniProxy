import { Bot } from "lucide-react"

const brandIcons: Record<
  string,
  { file: string; monochrome?: boolean; darkTile?: boolean; fullTile?: boolean }
> = {
  openai: { file: "openai.svg", monochrome: true },
  anthropic: { file: "anthropic.svg", monochrome: true },
  google: { file: "google-color.svg" },
  deepseek: { file: "deepseek-color.svg" },
  zhipu: { file: "zhipu-color.svg" },
  kimi: { file: "kimi-color.svg", darkTile: true },
  minimax: { file: "minimax-color.svg" },
  mimo: { file: "xiaomimimo.svg", monochrome: true },
  stepfun: { file: "stepfun-color.svg" },
  atria: { file: "atria.png" },
  river: { file: "river.png", fullTile: true },
}

function resolveBrandIcon(kind: string, endpoint?: string) {
  if (Object.hasOwn(brandIcons, kind)) return brandIcons[kind]
  if (kind !== "openai-compatible" || !endpoint || endpoint.length > 2048) return undefined

  // 既有 Atria 账号属于自定义服务商，仅按官方主机名识别品牌，避免相似域名误匹配。
  try {
    const url = new URL(endpoint)
    if (
      url.protocol === "https:" &&
      url.hostname === "api.atria-asi.ai" &&
      !url.port &&
      !url.username &&
      !url.password
    ) {
      return brandIcons.atria
    }
  } catch {
    // 无效地址只影响装饰图标，继续使用通用图标，配置校验仍由后端负责。
    return undefined
  }
  return undefined
}

export function ProviderBrandIcon({ kind, endpoint }: { kind: string; endpoint?: string }) {
  const brandIcon = resolveBrandIcon(kind, endpoint)
  return (
    <span
      className={`flex size-9 shrink-0 items-center justify-center rounded-lg ${brandIcon?.fullTile ? "overflow-hidden" : brandIcon?.monochrome || brandIcon?.darkTile ? "bg-zinc-950" : brandIcon ? "border-border border bg-white" : "bg-slate-700 text-white"}`}
      aria-hidden="true"
    >
      {brandIcon ? (
        <img
          src={`/provider-icons/${brandIcon.file}`}
          alt=""
          className={`${brandIcon.fullTile ? "size-full" : "size-6"} object-contain ${brandIcon.monochrome ? "brightness-0 invert" : ""}`}
        />
      ) : (
        <Bot className="size-5" />
      )}
    </span>
  )
}
