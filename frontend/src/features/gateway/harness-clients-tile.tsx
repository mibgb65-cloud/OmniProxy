import { AgentClientGroupTile, type AgentClientGroupProps } from "./agent-client-group-tile"

export function HarnessClientsTile(props: AgentClientGroupProps) {
  return (
    <AgentClientGroupTile
      {...props}
      title="agents.harness"
      description="agents.harnessDescription"
      clients={["harness-desktop", "harness-web"]}
    />
  )
}
