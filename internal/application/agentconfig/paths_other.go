//go:build !windows

package agentconfig

func desktopThirdPartyRoot(_, fallback string) (string, error) { return fallback, nil }
