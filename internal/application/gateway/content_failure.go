package gateway

import (
	"errors"

	"omniproxy/internal/core/agentprotocol"
	"omniproxy/internal/core/proxy"
)

func protocolContentFailure(err error) error {
	if errors.Is(err, agentprotocol.ErrUnsupportedImage) {
		return proxy.UnsupportedImage()
	}
	return proxy.Failure("unsupported_content", 400)
}
