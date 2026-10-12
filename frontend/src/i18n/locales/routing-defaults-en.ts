import type { routingDefaultsZh } from "./routing-defaults-zh"

export const routingDefaultsEn: Record<keyof typeof routingDefaultsZh, string> = {
  "routing.defaults.title": "Default model mappings",
  "routing.defaults.description":
    "Discovered models map automatically by model ID. Choose a preferred provider for each model; changes save automatically.",
  "routing.defaults.count": "{{count}} models",
  "routing.defaults.loading": "Loading default model mappings…",
  "routing.defaults.loadFailed": "Could not load default mappings. Refresh and retry.",
  "routing.defaults.saveFailed":
    "Could not save the preferred provider. Your current setting is kept. Refresh and retry.",
  "routing.defaults.saving": "Saving the preferred provider for {{model}}…",
  "routing.defaults.saved": "Preferred provider for {{model}} saved.",
  "routing.defaults.automatic": "Default mapping",
  "routing.defaults.overridden": "Custom rule override",
  "routing.defaults.target": "Currently using: {{provider}} · {{model}}",
  "routing.defaults.preferLabel": "Preferred provider for {{model}}",
  "routing.defaults.autoSelect": "Choose provider automatically",
  "routing.defaults.providerCount": "{{count}} available providers",
  "routing.defaults.ruleHint": "An enabled custom rule controls this model.",
  "routing.defaults.savedUnavailable": "{{name}} (currently unavailable)",
  "routing.defaults.fallback": "Preferred provider is unavailable; using {{provider}}.",
  "routing.defaults.unavailable":
    "Preferred provider is unavailable. You can restore automatic selection.",
  "routing.defaults.search": "Search default model mappings",
  "routing.defaults.noMatches": "No default models match your search.",
  "routing.defaults.empty":
    "No model catalog yet. Add an available account or sync official models to generate mappings.",
  "routing.defaults.policy":
    "Custom rules take precedence. Default mappings prefer your chosen provider, then other available providers supporting the same model ID. Accounts within a provider follow creation order.",
  "routing.defaults.preferredMatch": "Preferred provider selected",
}
