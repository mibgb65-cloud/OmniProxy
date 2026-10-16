import { AgentClientGroupTile, type AgentClientGroupProps } from "./agent-client-group-tile"

export function ClaudeClientsTile(props: AgentClientGroupProps) {
  return (
    <AgentClientGroupTile
      {...props}
      title="agents.claudeGroup"
      description="agents.claudeGroupDescription"
      clients={["claude", "claude-desktop"]}
    />
  )
}
