package modelcatalog

import "strings"

// ServiceTier 只描述上游明确公布的服务档位，不代表用户已经选择该计费模式。
type ServiceTier struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func validateServiceTiers(tiers []ServiceTier) error {
	if len(tiers) > 8 {
		return ErrInvalid
	}
	seen := make(map[string]bool, len(tiers))
	for _, tier := range tiers {
		if tier.ID == "" || len(tier.ID) > 32 || strings.ContainsFunc(tier.ID, func(r rune) bool {
			return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-')
		}) || seen[tier.ID] || strings.TrimSpace(tier.Name) == "" || !textValid(tier.Name, 80) || !textValid(tier.Description, 500) {
			return ErrInvalid
		}
		seen[tier.ID] = true
	}
	return nil
}
