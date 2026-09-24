//go:build !production

package profile

// defaultKind is the profile used when OMNIPROXY_PROFILE is unset and the
// binary was not built with the `production` build tag. `wails3 dev` and
// `go run` builds are development builds; `wails3 build` (and
// `go build -tags production`) select production via kind_production.go.
const defaultKind = Development
