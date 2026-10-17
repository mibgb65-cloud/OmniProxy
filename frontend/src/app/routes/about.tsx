import { createFileRoute } from "@tanstack/react-router"
import { OpenPreferencesRoute } from "../../features/settings/open-preferences-route"

export const Route = createFileRoute("/about")({
  component: () => <OpenPreferencesRoute tab="about" />,
})
