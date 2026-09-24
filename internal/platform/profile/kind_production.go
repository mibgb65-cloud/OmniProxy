//go:build production

package profile

// defaultKind is the profile used when the binary is built with the
// `production` build tag (what `wails3 build` uses by default).
const defaultKind = Production
