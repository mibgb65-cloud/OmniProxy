// Package provider manages provider configurations independently of the desktop shell.
package provider

// Definition describes a built-in provider kind and its authentication options.
type Definition struct {
	Kind               string   `json:"kind"`
	Name               string   `json:"name"`
	AuthMethods        []string `json:"authMethods"`
	PlannedAuthMethods []string `json:"plannedAuthMethods"`
}

const (
	AuthAPIKey            = "api_key"
	AuthSubscriptionLogin = "subscription_login"
)

var definitions = []Definition{
	{Kind: "openai", Name: "OpenAI", AuthMethods: []string{AuthAPIKey}, PlannedAuthMethods: []string{AuthSubscriptionLogin}},
	{Kind: "anthropic", Name: "Anthropic", AuthMethods: []string{AuthAPIKey}},
	{Kind: "google", Name: "Google", AuthMethods: []string{AuthAPIKey}},
	{Kind: "deepseek", Name: "DeepSeek", AuthMethods: []string{AuthAPIKey}},
	{Kind: "zhipu", Name: "Zhipu AI", AuthMethods: []string{AuthAPIKey}},
	{Kind: "kimi", Name: "Moonshot AI", AuthMethods: []string{AuthAPIKey}},
	{Kind: "minimax", Name: "MiniMax", AuthMethods: []string{AuthAPIKey}},
	{Kind: "mimo", Name: "Xiaomi MiMo", AuthMethods: []string{AuthAPIKey}},
	{Kind: "stepfun", Name: "StepFun", AuthMethods: []string{AuthAPIKey}},
}

// Catalog returns an independent snapshot of the nine built-in provider kinds.
func Catalog() []Definition {
	result := make([]Definition, len(definitions))
	for i, definition := range definitions {
		result[i] = definition
		result[i].AuthMethods = append([]string(nil), definition.AuthMethods...)
		result[i].PlannedAuthMethods = append([]string(nil), definition.PlannedAuthMethods...)
	}
	return result
}

func knownKind(kind string) bool {
	for _, definition := range definitions {
		if definition.Kind == kind {
			return true
		}
	}
	return false
}
