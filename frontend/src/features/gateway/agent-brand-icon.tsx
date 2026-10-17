import { agentClients, type AgentClientId } from "./agent-clients"

export function AgentBrandIcon({ client }: { client: AgentClientId }) {
  return (
    <span
      aria-hidden="true"
      className={`flex size-14 shrink-0 items-center justify-center rounded-2xl border ${client === "codex" ? "border-zinc-800 bg-zinc-950" : client.startsWith("claude") ? "border-[#D97757]/20 bg-[#D97757]/10" : "bg-muted/30"}`}
    >
      <img
        src={
          client === "mimo-code"
            ? "/provider-icons/xiaomimimo.svg"
            : client.startsWith("harness-")
              ? "/provider-icons/deepseek-color.svg"
              : `/agent-icons/${agentClients[client].icon}`
        }
        alt=""
        width={32}
        height={32}
        className={`size-8 ${client === "codex" ? "brightness-0 invert" : client === "opencode" || client === "kilocode" || client === "mimo-code" ? "dark:invert" : ""}`}
      />
    </span>
  )
}
