import { agentClients, type AgentClientId } from "./agent-clients"

export function AgentBrandIcon({ client }: { client: AgentClientId }) {
  return (
    <span
      aria-hidden="true"
      className={`flex size-14 shrink-0 items-center justify-center rounded-2xl border ${client === "codex" ? "border-zinc-800 bg-zinc-950" : "border-[#D97757]/20 bg-[#D97757]/10"}`}
    >
      <img
        src={`/agent-icons/${agentClients[client].icon}`}
        alt=""
        width={32}
        height={32}
        className={`size-8 ${client === "codex" ? "brightness-0 invert" : ""}`}
      />
    </span>
  )
}
