package codex

import (
	"runtime/debug"
	"strconv"

	"filippo.io/age"
)

const maxAuthWorkFactor = 20
const reclaimAuthWorkFactor = 18

type authScryptIdentity struct {
	*age.ScryptIdentity
	highCost bool
}

func (i *authScryptIdentity) Unwrap(stanzas []*age.Stanza) ([]byte, error) {
	for _, stanza := range stanzas {
		if stanza.Type == "scrypt" && len(stanza.Args) == 2 {
			factor, err := strconv.Atoi(stanza.Args[1])
			i.highCost = i.highCost || err == nil && factor >= reclaimAuthWorkFactor && factor <= maxAuthWorkFactor
		}
	}
	return i.ScryptIdentity.Unwrap(stanzas)
}

func (i *authScryptIdentity) reclaim() {
	if i.highCost {
		// 仅在实际高成本解密后归还至少 256 MiB 的临时 KDF 工作区；
		// 不调低官方安全参数，也不在请求热路径／定时器中盲目强制 GC。
		debug.FreeOSMemory()
	}
}
